package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/markdown"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/ansitags"
	"github.com/mattn/go-runewidth"
	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/util"
)

type AnsiFlag uint8

const (
	AnsiTagsDefault AnsiFlag          = iota // Do not parse tags
	AnsiTagsParse                            // Parse ansi tags before returning contents of template
	AnsiTagsStrip                            // strip out all ansi tags and leave text plain
	AnsiTagsMono                             // Parse ansi tags, but strip out all color information
	AnsiTagsNone    = AnsiTagsDefault        // alias to default

	ForceScreenReaderUserId = -1

	divider = "\n<ansi fg=\"6\">  .--.      .-'.      .--.      .--.      .--.      .--.      .`-.      .--.\n" +
		"<ansi fg=\"187\">:::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\"></ansi>\n" +
		"'      `--'      `.-'      `--'      `--'      `--'      `-.'      `--'      `</ansi>\n"
)

type cacheEntry struct {
	tpl           *template.Template
	ansiPreparsed bool
	modified      time.Time
}

func (t *cacheEntry) older(compareTime time.Time) bool {
	return t.modified.Before(compareTime)
}

var (
	cacheLock            sync.Mutex
	templateCache        = make(map[string]cacheEntry)
	templateConfigCache  sync.Map // key: int userId -> value: templateConfig (concurrency-safe)
	forceAnsiFlags       = AnsiTagsParse
	ansiLock             sync.RWMutex
	ansiAliasFileModTime time.Time

	fileSystems []fs.ReadFileFS
)

func RegisterFS(f fs.ReadFileFS) {
	fileSystems = append(fileSystems, f)
}

func readFile(path string) (b []byte, err error) {

	for _, f := range fileSystems {
		if b, err = f.ReadFile(path); err == nil {
			return b, nil
		}
	}

	return b, err
}

func Exists(name string) bool {

	// First check registered file systems (plugins?)
	path := util.FilePath(`templates`, `/`, name+`.template`)
	for _, f := range fileSystems {
		if fsFile, err := f.Open(path); err == nil {
			fsFile.Close()
			return true
		}
	}

	// Now check datafiles
	var fullPath string = util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, path)
	_, err := os.Stat(fullPath)

	return err == nil
}

// Configure a forced ansi flag setting
func SetAnsiFlag(flag AnsiFlag) {
	forceAnsiFlags = flag
}

type templateConfig struct {
	ScreenReader bool // If they are using a screen reader, attempt to load a SR friendly template
	AnsiFlags    AnsiFlag
}

type templateDetails struct {
	name       string
	path       string
	preProcess func(string) string
}

func ClearTemplateConfigCache(userId int) {
	// templateConfigCache is a sync.Map; Delete is concurrency-safe, no lock needed.
	templateConfigCache.Delete(userId)
}

// getTemplateConfig returns the cached render config for userId, building and
// caching it on first use. Safe for concurrent use (templateConfigCache is a
// sync.Map). On a simultaneous first-use miss two callers may both build and
// Store; harmless, the value is deterministic for a given userId.
func getTemplateConfig(userId int) templateConfig {
	if v, ok := templateConfigCache.Load(userId); ok {
		return v.(templateConfig)
	}
	cfg := templateConfig{}
	if userId > 0 {
		if tmpU := users.GetByUserId(userId); tmpU != nil {
			cfg.ScreenReader = tmpU.ScreenReader
		}
	} else if userId == ForceScreenReaderUserId {
		cfg.ScreenReader = true
	}
	templateConfigCache.Store(userId, cfg)
	return cfg
}

func processMarkdown(in string) string {
	markdown.SetFormatter(markdown.ANSITags{})
	p := markdown.NewParser(in)
	return "\n" + divider + "\n" + p.Parse().String(0) + "\n"
}

func Process(fname string, data any, receivingUserId ...int) (string, error) {
	ansiLock.RLock()
	defer ansiLock.RUnlock()

	var parseAnsiTags bool = false

	var ansitagsParseBehavior []ansitags.ParseBehavior = make([]ansitags.ParseBehavior, 0, 2)

	if forceAnsiFlags != AnsiTagsDefault {
		//	ansiFlags = append(ansiFlags, forceAnsiFlags)
	}

	userId := 0
	if len(receivingUserId) > 0 {
		userId = receivingUserId[0]
	}

	tplConfig := getTemplateConfig(userId)

	var buf bytes.Buffer

	// Contains each template to attempt to load, in order.
	// This allows us to attempt optional adjusted template paths first.
	filesToAttempt := []templateDetails{}

	// Try a screen-reader friendly template first?
	if tplConfig.ScreenReader {
		filesToAttempt = append(filesToAttempt,
			templateDetails{
				name: fname,
				path: util.FilePath(`templates/`, fname+`.screenreader.template`), // All templates must end with .template
			},
		)
	}

	filesToAttempt = append(filesToAttempt,
		templateDetails{
			name:       fname,
			path:       util.FilePath(`templates/`, fname+`.md`), // All templates must end with .template
			preProcess: processMarkdown,
		},
	)

	filesToAttempt = append(filesToAttempt,
		templateDetails{
			name: fname,
			path: util.FilePath(`templates/`, fname+`.template`), // All templates must end with .template
		},
	)

	for _, tplInfo := range filesToAttempt {

		if fileBytes, err := readFile(tplInfo.path); err == nil {

			tpl, err := template.New(tplInfo.name).Funcs(funcMap).Parse(string(fileBytes))
			if err != nil {
				mudlog.Error("template parse error", "module", true, "filepath", tplInfo.path, "error", err)
				return string(fileBytes), err
			}

			err = tpl.Execute(&buf, data)
			if err != nil {
				mudlog.Error("could not parse template file", "module", true, "filepath", tplInfo.path, "error", err)
				return "[TEMPLATE ERROR]", err
			}

			// return the final data as a string, parse ansi tags if needed (No need to parse if it was preparsed)
			if parseAnsiTags {
				return ansitags.Parse(buf.String(), ansitagsParseBehavior...), nil
			}

			if tplInfo.preProcess != nil {
				return tplInfo.preProcess(buf.String()), nil
			}
			return buf.String(), nil
		}

		//
		// nothing able to load from the plugin files, lets try the normal filesystem.
		//

		fullPath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, tplInfo.path)

		// Get the file contents
		fileContents, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// parse the file contents as a template
		tpl, err := template.New(tplInfo.name).Funcs(funcMap).Parse(string(fileContents))
		if err != nil {
			mudlog.Error("template parse error", "module", false, "filepath", fullPath, "error", err)
			return string(fileContents), err
		}

		// execute the template and store the results into a buffer

		err = tpl.Execute(&buf, data)
		if err != nil {
			mudlog.Error("could not parse template file", "module", false, "filepath", fullPath, "error", err)
			return "[TEMPLATE ERROR]", err
		}

		if tplConfig.ScreenReader {
			strippedOut := util.StripCharsForScreenReaders(buf.String())

			// return the final data as a string, parse ansi tags if needed (No need to parse if it was preparsed)
			if parseAnsiTags {
				return ansitags.Parse(strippedOut, ansitagsParseBehavior...), nil
			}
			return strippedOut, nil
		}

		// return the final data as a string, parse ansi tags if needed (No need to parse if it was preparsed)
		if parseAnsiTags {
			return ansitags.Parse(buf.String(), ansitagsParseBehavior...), nil
		}

		if tplInfo.preProcess != nil {
			return tplInfo.preProcess(buf.String()), nil
		}
		return buf.String(), nil
	}

	//
	// If template never found, return details.
	//
	allFiles := []string{}
	for _, tplInfo := range filesToAttempt {
		allFiles = append(allFiles, tplInfo.path)
	}
	mudlog.Error("template files not found", "files", strings.Join(allFiles, `, `))
	return fmt.Sprintf(`[TEMPLATE READ ERROR: FNF (%s) `, strings.Join(allFiles, `, `)), fmt.Errorf(`Files not found: %s`, strings.Join(allFiles, `, `))
}

func ProcessText(text string, data any, ansiFlags ...AnsiFlag) (string, error) {

	var parseAnsiTags bool = false

	var ansitagsParseBehavior []ansitags.ParseBehavior = make([]ansitags.ParseBehavior, 0, 5)

	if forceAnsiFlags != AnsiTagsDefault {
		//	ansiFlags = append(ansiFlags, forceAnsiFlags)
	}

	for _, flag := range ansiFlags {
		switch flag {
		case AnsiTagsStrip:
			ansitagsParseBehavior = append(ansitagsParseBehavior, ansitags.StripTags)
		case AnsiTagsMono:
			ansitagsParseBehavior = append(ansitagsParseBehavior, ansitags.Monochrome)
		case AnsiTagsParse:
			parseAnsiTags = true
		}
	}

	// parse the file contents as a template
	tpl, err := template.New("").Funcs(funcMap).Parse(text)
	if err != nil {
		return text, err
	}

	// execute the template and store the results into a buffer
	var buf bytes.Buffer
	err = tpl.Execute(&buf, data)
	if err != nil {
		mudlog.Error("could not parse template text", "error", err)
		return "[TEMPLATE TEXT ERROR]", err
	}

	// return the final data as a string, parse ansi tags if needed (No need to parse if it was preparsed)
	if parseAnsiTags {
		return ansitags.Parse(buf.String(), ansitagsParseBehavior...), nil
	}

	return buf.String(), nil
}

const cellPadding int = 1

type TemplateTable struct {
	Title              string
	Header             []string
	Rows               [][]string
	TrueHeaderCellSize []int
	TrueCellSize       [][]int
	ColumnCount        int
	ColumnWidths       []int
	Formatting         [][]string
	formatRowCount     int
}

func (t TemplateTable) GetHeaderCell(column int) string {

	cellStr := t.Header[column]
	repeatCt := t.ColumnWidths[column] - t.TrueHeaderCellSize[column]
	if repeatCt > 0 {
		cellStr += strings.Repeat(` `, repeatCt)
	}

	return cellStr
}

func (t TemplateTable) GetCell(row int, column int) string {

	cellStr := t.Rows[row][column]
	repeatCt := t.ColumnWidths[column] - t.TrueCellSize[row][column]
	if repeatCt > 0 {
		cellStr += strings.Repeat(` `, repeatCt)
	}

	if t.formatRowCount > 0 {
		cellFormat := t.Formatting[row%t.formatRowCount][column]
		if cellFormat[0:1] == `:` {
			return colorpatterns.ApplyColorPattern(cellStr, cellFormat[1:])
		}
		return fmt.Sprintf(t.Formatting[row%t.formatRowCount][column], cellStr)
	}
	return cellStr
}

func GetTable(title string, headers []string, rows [][]string, formatting ...[]string) TemplateTable {

	var table TemplateTable = TemplateTable{
		Title:              title,
		Header:             headers,
		Rows:               rows,
		TrueHeaderCellSize: []int{},
		TrueCellSize:       [][]int{},
		ColumnCount:        len(headers),
		ColumnWidths:       make([]int, len(headers)),
		Formatting:         formatting,
	}

	hdrColCt := len(headers)
	rowCt := len(rows)
	table.formatRowCount = len(formatting)
	table.TrueHeaderCellSize = make([]int, hdrColCt)
	table.TrueCellSize = make([][]int, rowCt)

	// Get the longest element
	for i := 0; i < hdrColCt; i++ {
		sz := runewidth.StringWidth(headers[i])
		if sz+1 > table.ColumnWidths[i] {
			table.ColumnWidths[i] = sz
		}
		table.TrueHeaderCellSize[i] = sz
	}

	// Get the longest element
	for r := 0; r < rowCt; r++ {
		rowColCt := len(rows[r])
		table.TrueCellSize[r] = make([]int, rowColCt)

		if hdrColCt < rowColCt {
			for i := hdrColCt; i < rowColCt; i++ {
				table.Header = append(table.Header, ``)
			}
			hdrColCt = len(table.Header)
		}

		for c := 0; c < hdrColCt; c++ {
			sz := runewidth.StringWidth(ansitags.Parse(rows[r][c], ansitags.StripTags))
			if sz+1 > table.ColumnWidths[c] {
				table.ColumnWidths[c] = sz
			}
			table.TrueCellSize[r][c] = sz
		}
	}

	if table.formatRowCount > 0 {
		var formatRowCols int
		for i := 0; i < table.formatRowCount; i++ {

			formatRowCols = len(table.Formatting[i])

			// Make sure there are enough formatting entries
			if formatRowCols < hdrColCt {

				for j := formatRowCols; j < hdrColCt; j++ {
					table.Formatting[j] = append(table.Formatting[j], `%s`)
				}

			}

		}

	}

	return table
}

func AnsiParse(input string) string {
	ansiLock.RLock()
	defer ansiLock.RUnlock()

	if forceAnsiFlags == AnsiTagsDefault {
		return input
	}

	if forceAnsiFlags == AnsiTagsParse {
		return ansitags.Parse(input)
	}

	if forceAnsiFlags == AnsiTagsStrip {
		return ansitags.Parse(input, ansitags.StripTags)
	}

	if forceAnsiFlags == AnsiTagsMono {
		return ansitags.Parse(input, ansitags.Monochrome)
	}

	return ansitags.Parse(input)
}

// Loads the ansi aliases from the config file
// Only if the file has been modified since the last load
func LoadAliases(f ...fileloader.ReadableGroupFS) {

	// Get the file info
	fInfo, err := os.Stat(util.FilePath(string(configs.GetFilePathsConfig().DataFiles) + `/ansi-aliases.yaml`))
	// check if filemtime is not ansiAliasFileModTime
	if err != nil || fInfo.ModTime() == ansiAliasFileModTime {
		return
	}

	ansiLock.Lock()
	defer ansiLock.Unlock()

	start := time.Now()

	ansiAliasFileModTime = fInfo.ModTime()
	if err = ansitags.LoadAliases(util.FilePath(string(configs.GetFilePathsConfig().DataFiles) + `/ansi-aliases.yaml`)); err != nil {
		mudlog.Info("ansitags.LoadAliases()", "changed", true, "Time Taken", time.Since(start), "error", err.Error())
	}

	OLPath := util.FilePath(`data-overlays`, `/`, `ansi-aliases.yaml`)
	for _, files := range f {
		if b, err := files.ReadFile(OLPath); err == nil {

			data := make(map[string]map[string]string, 100)
			if err := yaml.Unmarshal(b, &data); err != nil {
				continue
			}

			for aliasGroup, aliases := range data {
				if aliasGroup != `colors` && aliasGroup != `color256` {
					continue
				}
				for alias, valStr := range aliases {
					if valInt, err := strconv.Atoi(valStr); err == nil {
						ansitags.SetAlias(alias, valInt)
					}
				}
			}

		}
	}

	mudlog.Info("ansitags.LoadAliases()", "changed", true, "Time Taken", time.Since(start))
}
