package fileloader

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"sync/atomic"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type FileType uint8
type SaveOption uint8

// implements fs.ReadFileFS
// implements an iterator function as well
type ReadableGroupFS interface {
	fs.ReadFileFS
	AllFileSubSystems(yield func(fs.ReadFileFS) bool)
}

type LoadableSimple interface {
	Validate() error  // General validation (or none)
	Filepath() string // Relative file path to some base directory - can include subfolders
}

type Loadable[K comparable] interface {
	Id() K // Must be a unique identifier for the data
	LoadableSimple
}

const (
	// Save options
	SaveCareful SaveOption = iota // Save a backup and rename vs. just overwriting
)

func LoadFlatFile[T LoadableSimple](path string) (T, error) {

	var loaded T

	path = filepath.FromSlash(path)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return loaded, errors.Wrap(err, `filepath: `+path)
	}

	if fileInfo.IsDir() {
		return loaded, errors.New(`filepath: ` + path + ` is a directory`)
	}

	fExt := filepath.Ext(path)
	if fExt != `.yaml` {
		return loaded, errors.New(`invalid file type: ` + path)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return loaded, errors.Wrap(err, `filepath: `+path)
	}

	err = yaml.Unmarshal(bytes, &loaded)
	if err != nil {
		return loaded, errors.Wrap(err, `filepath: `+path)
	}

	// Make sure the Filepath it claims is correct in case we need to save it later
	if !strings.HasSuffix(path, filepath.FromSlash(loaded.Filepath())) {
		return loaded, errors.New(fmt.Sprintf(`filesystem path "%s" did not end in Filepath() "%s" for type %T`, path, loaded.Filepath(), loaded))
	}

	// validate the structure
	if err := loaded.Validate(); err != nil {
		return loaded, errors.Wrap(err, `filepath: `+path)
	}

	return loaded, nil
}

// LoadAllFlatFilesSimple doesn't require a unique Id() for each item
func LoadAllFlatFilesSimple[T LoadableSimple](basePath string, filePattern ...string) ([]T, error) {

	loadedData := make([]T, 0, 128)

	fileSuffix := `.yaml` // Only support yaml
	suffixLen := len(fileSuffix)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if len(path) < suffixLen {
			return nil
		}

		if path[len(path)-suffixLen:] != fileSuffix {
			return nil
		}

		if len(filePattern) > 0 {
			fileName := filepath.Base(path)
			if ok, _ := filepath.Match(filePattern[0], fileName); !ok {
				return nil
			}
		}

		loaded, err := LoadFlatFile[T](path)

		if err != nil {
			return err
		}

		loadedData = append(loadedData, loaded)

		return nil
	})

	return loadedData, err
}

// Will check the ID() of each item to make sure it's unique
func LoadAllFlatFiles[K comparable, T Loadable[K]](basePath string, filePattern ...string) (map[K]T, error) {

	basePath = filepath.FromSlash(basePath)

	loadedData := make(map[K]T)

	fileSuffix := `.yaml` // Only support yaml
	suffixLen := len(fileSuffix)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if len(path) < suffixLen {
			return nil
		}

		if path[len(path)-suffixLen:] != fileSuffix {
			return nil
		}

		if len(filePattern) > 0 {
			fileName := filepath.Base(path)
			if ok, _ := filepath.Match(filePattern[0], fileName); !ok {
				return nil
			}
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, `filepath: `+path)
		}

		var loaded T

		err = yaml.Unmarshal(bytes, &loaded)
		if err != nil {
			return errors.Wrap(err, `filepath: `+path)
		}

		if !strings.HasSuffix(path, filepath.FromSlash(loaded.Filepath())) {
			return errors.New(fmt.Sprintf(`filesystem path "%s" did not end in Filepath() "%s" for type %T`, path, loaded.Filepath(), loaded))
		}

		if err := loaded.Validate(); err != nil {
			return errors.Wrap(err, `filepath: `+path)
		}

		if _, ok := loadedData[loaded.Id()]; ok {
			return errors.New(fmt.Sprintf(`duplicate id %v for type %T`, loaded.Id(), loaded))
		}

		loadedData[loaded.Id()] = loaded

		return nil
	})

	return loadedData, err
}

// Returns the number of files saved and error
func SaveFlatFile[T LoadableSimple](basePath string, dataUnit T, saveOptions ...SaveOption) error {

	// Normalize slashes
	basePath = filepath.FromSlash(basePath)

	carefulSave := false
	if len(saveOptions) > 0 {
		for _, saveOption := range saveOptions {
			if saveOption == SaveCareful {
				carefulSave = true
			}
		}
	}

	// Get filepath from interface
	path := filepath.Join(basePath, dataUnit.Filepath())
	fExt := filepath.Ext(path)

	// Use filepath to determine file marshal type
	if fExt != `.yaml` {
		return errors.New(fmt.Sprint(`SaveFlatFile`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, `unsupported file type`))
	}

	os.MkdirAll(filepath.Dir(path), os.ModePerm)

	bytes, err := yaml.Marshal(dataUnit)
	if err != nil {
		return errors.New(fmt.Sprint(`SaveFlatFile`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err))
	}

	saveFilePath := path
	if carefulSave { // careful save first saves a {filename}.new file
		saveFilePath += `.new`
	}

	//
	// write to .new suffix in case of power loss etc.
	//
	if err := os.WriteFile(saveFilePath, bytes, 0777); err != nil {
		return errors.New(fmt.Sprint(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err))
	}

	if carefulSave {
		//
		// Once the file is written, rename it to remove the .new suffix and overwrite the old file
		//
		if err := os.Rename(saveFilePath, path); err != nil {
			return errors.New(fmt.Sprint(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err))
		}
	}

	return nil
}

// Returns the number of files saved and error
func SaveAllFlatFiles[K comparable, T Loadable[K]](basePath string, data map[K]T, saveOptions ...SaveOption) (int, error) {

	// Normalize slashes
	basePath = filepath.FromSlash(basePath)

	var saveCt int32

	workerCt := runtime.GOMAXPROCS(0)

	var wg sync.WaitGroup
	tData := make(chan T, 1)

	carefulSave := false
	if len(saveOptions) > 0 {
		for _, saveOption := range saveOptions {
			if saveOption == SaveCareful {
				carefulSave = true
			}
		}
	}

	// Spin up workers
	for i := 0; i < workerCt; i++ {

		wg.Add(1)

		go func(dataIn chan T, waitGroup *sync.WaitGroup) {
			defer waitGroup.Done()

			var bytes []byte
			var err error
			var ct int32 = 0

			for dataUnit := range dataIn {

				// Get filepath from interface
				path := filepath.Join(basePath, dataUnit.Filepath())
				fExt := filepath.Ext(path)

				// Use filepath to determine file marshal type
				if fExt != `.yaml` {
					mudlog.Error(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, `unsupported file type`)
					continue
				}

				bytes, err = yaml.Marshal(dataUnit)
				if err != nil {
					mudlog.Error(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err)
					continue
				}

				saveFilePath := path
				if carefulSave { // careful save first saves a {filename}.new file
					saveFilePath += `.new`
				}

				//
				// write to .new suffix in case of power loss etc.
				//
				if err := os.WriteFile(saveFilePath, bytes, 0777); err != nil {
					mudlog.Error(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err)
					continue
				}

				if carefulSave {
					//
					// Once the file is written, rename it to remove the .new suffix and overwrite the old file
					//
					if err := os.Rename(saveFilePath, path); err != nil {
						mudlog.Error(`SaveAllFlatFiles`, `basePath`, basePath, `type`, fmt.Sprintf(`%T`, *new(T)), `path`, path, `err`, err)
						continue
					}
				}

				// count saves
				ct++
			}

			atomic.AddInt32(&saveCt, ct)

		}(tData, &wg)
	}

	// Feed all of the data to workers
	for _, d := range data {
		tData <- d
	}

	// Close the channel and wait for workers to finish
	close(tData)

	wg.Wait()

	return int(saveCt), nil
}

func CopyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
