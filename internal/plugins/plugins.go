package plugins

import (
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

//
// This package defines the basic nature of GoMud plugins
// To add new plugins, they must be dropped in this folder and the server re-compiled.
//

// pluginRegistry holds all plugins, provides a `fs.ReadFileFS` interface

var (
	registrationOpen = true
	registry         = pluginRegistry{}
	txtCleanRegex    = regexp.MustCompile(`[^a-zA-Z0-9\._]+`)

	// writeFolderPath is repointed at the real plugin-data directory by Load().
	// Until then it is the OS temp dir, so anything a plugin persists before
	// Load runs is written somewhere it will never be read back from —
	// writeFolderReady exists so that mistake is logged rather than silent.
	writeFolderPath  = os.TempDir()
	writeFolderReady = false

	// exportedFunctionOwners maps an exported function id to the plugin that
	// claimed it, so a collision is caught at registration. Ids are global.
	exportedFunctionOwners = map[string]string{}
)

const (
	// Use forward slashes: embed.FS always uses forward slashes, even on Windows.
	dataFilesFolder         = `datafiles/`
	dataOverlaysFilesFolder = `data-overlays/`
)

type pluginRegistry []*Plugin

// Plugin struct
type dependency struct {
	name    string
	version string
}

type Plugin struct {
	name    string
	version string

	dependencies []dependency

	Callbacks PluginCallbacks

	exportedFunctions map[string]any

	Config PluginConfig

	// helper for embedded files
	files PluginFiles

	Web WebConfig
}

func New(name string, version string) *Plugin {

	// name can only contain alphanumeric or underscores.
	reg, _ := regexp.Compile("[^a-zA-Z0-9_]+")
	name = reg.ReplaceAllString(name, "_")

	if !registrationOpen {
		return nil
	}

	p := &Plugin{
		name:         name,
		version:      version,
		dependencies: []dependency{},
		Config: PluginConfig{
			pluginName: name,
		},
		Callbacks: newPluginCallbacks(),
		Web:       newWebConfig(),
	}

	registry = append(registry, p)
	return p
}

func (p pluginRegistry) GetExportedFunction(funcName string) (any, bool) {
	for _, pItem := range registry {

		if pItem.exportedFunctions == nil {
			continue
		}

		if f, ok := pItem.exportedFunctions[funcName]; ok {
			return f, ok
		}

	}
	return nil, false
}

// Receive functions to satisfy the web.WebPlugin interface
func (p pluginRegistry) NavLinks() map[string]string {

	allLinks := map[string]string{}

	for _, pItem := range p {
		maps.Copy(allLinks, pItem.Web.navLinks)
	}

	return allLinks
}

func (p pluginRegistry) HandleIAC(connectionId uint64, iacCmd []byte) bool {

	for _, pItem := range p {
		if pItem.Callbacks.iacHandler == nil {
			continue
		}
		if pItem.Callbacks.iacHandler(connectionId, iacCmd) {
			return true
		}
	}

	return false
}

func (p pluginRegistry) WebRequest(r *http.Request) (html string, templateData map[string]any, ok bool) {

	// Use path.Clean (not filepath.Clean) so URL paths stay with forward slashes on Windows.
	reqPath := path.Clean(r.URL.Path) // Example: / or /info/faq

	rootFilePath := `html/public/`
	for _, pItem := range p {

		pageData, ok := pItem.Web.pages[reqPath]
		if !ok {
			continue
		}

		// Use forward-slash concatenation (not util.FilePath) because embed.FS keys use forward slashes.
		b, err := pItem.files.ReadFile(rootFilePath + pageData.Filepath)

		if err != nil {
			continue
		}

		html = string(b)

		if pageData.DataFunction != nil {
			templateData = pageData.DataFunction(r)
		}

		return html, templateData, true
	}

	return html, templateData, false
}

// Iterator for all plugin file systems.
// This allows you to process each one individually.
func (p pluginRegistry) AllFileSubSystems(yield func(fs.ReadFileFS) bool) {

	for _, pItem := range p {
		if !yield(pItem.files) {
			return
		}
	}

}

// Reads the first available file found in a plugin and uses it.
// This means only one plugin wins!
func (p pluginRegistry) ReadFile(name string) ([]byte, error) {
	for _, p := range registry {

		if embedPath, ok := p.files.filePaths[name]; ok {
			b, err := p.files.fileSystem.ReadFile(embedPath)
			if err == nil {
				return b, nil
			}
		}
	}

	return nil, fs.ErrNotExist
}

func (p pluginRegistry) Open(name string) (fs.File, error) {

	for _, p := range registry {

		if embedPath, ok := p.files.filePaths[name]; ok {
			return p.files.fileSystem.Open(embedPath)

		}
	}

	return nil, fs.ErrNotExist

}

func (p pluginRegistry) Stat(name string) (fs.FileInfo, error) {

	for _, p := range registry {

		if embedPath, ok := p.files.filePaths[name]; ok {
			return fs.Stat(p.files.fileSystem, embedPath)
		}
	}

	return nil, fs.ErrNotExist

}

func (p *Plugin) Requires(modname string, modversion string) {
	// modname can only contain alphanumeric or underscores.
	reg, _ := regexp.Compile("[^a-zA-Z0-9_]+")
	modname = reg.ReplaceAllString(modname, "_")
	p.dependencies = append(p.dependencies, dependency{modname, modversion})
}

// ExportFunction publishes f under stringId so other modules can reach it via
// GetExportedFunction.
//
// Exported ids are a FLAT GLOBAL NAMESPACE across every plugin —
// GetExportedFunction walks the whole registry and returns the first match — so
// two modules exporting the same id would silently resolve to whichever
// registered first. That is a mis-wire that produces no error and no log, and
// changes behaviour with registration order, so it is rejected at registration
// instead. Qualify your ids (e.g. "weather.GetFront", not "GetFront").
func (p *Plugin) ExportFunction(stringId string, f any) {

	if reflect.TypeOf(f).Kind() != reflect.Func {
		panic("Non function passed to ExportFunction")
	}

	if owner, taken := exportedFunctionOwners[stringId]; taken {
		panic(fmt.Sprintf(
			"plugin %q exports function id %q, which plugin %q already exports; ids are global, so qualify them",
			p.name, stringId, owner))
	}
	exportedFunctionOwners[stringId] = p.name

	if p.exportedFunctions == nil {
		p.exportedFunctions = map[string]any{}
	}
	p.exportedFunctions[stringId] = f
}

// Registers a UserCommand and callback
func (p *Plugin) AddUserCommand(command string, handlerFunc usercommands.UserCommand, allowWhenDowned bool, isAdminOnly bool) {

	if p.Callbacks.userCommands == nil {
		p.Callbacks.userCommands = map[string]usercommands.CommandAccess{}
	}

	p.Callbacks.userCommands[command] = usercommands.CommandAccess{
		Func:              handlerFunc,
		AllowedWhenDowned: allowWhenDowned,
		AdminOnly:         isAdminOnly,
	}
}

// Registers a MobCommand and callback
func (p *Plugin) AddMobCommand(command string, handlerFunc mobcommands.MobCommand, allowWhenDowned bool) {

	if p.Callbacks.mobCommands == nil {
		p.Callbacks.mobCommands = map[string]mobcommands.CommandAccess{}
	}

	p.Callbacks.mobCommands[command] = mobcommands.CommandAccess{
		Func:              handlerFunc,
		AllowedWhenDowned: allowWhenDowned,
	}

}

// Adds an embedded file system to the plugin
func (p *Plugin) AttachFileSystem(f embed.FS) error {

	p.files.fileSystem = f

	p.files.filePaths = make(map[string]string)

	// Walk the directory tree rooted at "datafiles"
	err := fs.WalkDir(p.files.fileSystem, `.`, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // propagate the error
		}

		// If it's not a directory, add the path to the list
		if d.IsDir() {
			return nil
		}

		// Handle datafiles folder.
		dfPos := strings.Index(path, dataFilesFolder)
		if dfPos != -1 {
			// map the short path to long embedded path
			p.files.filePaths[path[dfPos+len(dataFilesFolder):]] = path
			return nil
		}

		// Handle data-overlays folder.
		// This is a special folder that overlays data onto other data
		dfPos = strings.Index(path, dataOverlaysFilesFolder)
		if dfPos != -1 {
			// map the short path to long embedded path
			// Put data-overlays/ prefix on for purposes of filefinding later.
			p.files.filePaths[dataOverlaysFilesFolder+path[dfPos+len(dataOverlaysFilesFolder):]] = path
			return nil
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("plug.AddFiles() for %s: %w", p.name, err)
	}

	return nil
}

func (p *Plugin) WriteBytes(identifier string, bytes []byte) error {

	if !writeFolderReady {
		mudlog.Warn(`plugin.WriteBytes`, `name`, p.name, `identifier`, identifier,
			`error`, `write before plugins.Load(); persisting to the OS temp dir, where it will not be read back`)
	}

	// Fix up identifier
	fileName := strings.ToLower(txtCleanRegex.ReplaceAllString(identifier, "-")) + `.plugin.dat`

	// Fix up folderpath
	folderPath := util.FilePath(writeFolderPath, `/`, strings.ToLower(txtCleanRegex.ReplaceAllString(fmt.Sprintf(`%s-v%s`, p.name, p.version), "-")))

	// Create full path
	fullPath := util.FilePath(folderPath, `/`, fileName)

	if _, err := os.Stat(fullPath); err != nil {
		if err = os.MkdirAll(folderPath, 0777); err != nil {
			mudlog.Error(`plugin.WriteBytes`, `name`, p.name, `path`, folderPath, `error`, err)
			return err
		}
	}

	if err := os.WriteFile(fullPath, bytes, 0777); err != nil {
		mudlog.Error(`plugin.WriteBytes`, `name`, p.name, `path`, fullPath, `error`, err)
		return err
	}

	return nil
}

func (p *Plugin) ReadBytes(identifier string) ([]byte, error) {

	// Fix up identifier
	fileName := strings.ToLower(txtCleanRegex.ReplaceAllString(identifier, "-")) + `.plugin.dat`

	// Fix up folderpath
	folderPath := util.FilePath(writeFolderPath, `/`, strings.ToLower(txtCleanRegex.ReplaceAllString(fmt.Sprintf(`%s-v%s`, p.name, p.version), "-")))

	// Create full path
	fullPath := util.FilePath(folderPath, `/`, fileName)

	bytes, err := os.ReadFile(fullPath)
	if err != nil && !os.IsNotExist(err) {
		mudlog.Warn(`plugin.ReadBytes`, `name`, p.name, `path`, fullPath, `error`, err)
	}

	return bytes, err
}

func (p *Plugin) WriteStruct(identifier string, in any) error {

	b, err := yaml.Marshal(in)
	if err != nil {
		return err
	}

	if err := p.WriteBytes(identifier, b); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) ReadIntoStruct(identifier string, out any) error {
	b, err := p.ReadBytes(identifier)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(b, out); err == nil {
		return err
	}

	return nil
}

func Load(dataFilesPath string) {

	writeFolderPath = util.FilePath(dataFilesPath, `/`, `plugin-data`)
	writeFolderReady = true

	registrationOpen = false

	pluginCt := 0
	for idx := len(registry) - 1; idx >= 0; idx-- {

		p := registry[idx]

		skipPlugin := false

		for _, dep := range p.dependencies {

			dependenciesMet := false

			for _, regCheckPlugin := range registry {
				// Later improve version matching.
				if regCheckPlugin.name == dep.name && regCheckPlugin.version == dep.version {
					dependenciesMet = true
				}
			}

			if !dependenciesMet {
				mudlog.Error("plugins", "Could not load plugin", p.name, "error", fmt.Sprintf("dependency not found: %s v%s", dep.name, dep.version))
				registry = append(registry[:idx], registry[idx+1:]...)
				skipPlugin = true
				break
			}

		}

		if skipPlugin {
			continue
		}

		pluginCt++

		for cmd, info := range p.Callbacks.userCommands {
			usercommands.RegisterCommand(cmd, info.Func, info.AllowedWhenDowned, info.AllowedInCombat, info.AdminOnly)
		}

		for cmd, info := range p.Callbacks.mobCommands {
			mobcommands.RegisterCommand(cmd, info.Func, info.AllowedWhenDowned)
		}

		// Check for config.yaml override and set missing values accordingly
		OLPath := util.FilePath(`data-overlays`, `/`, `config.yaml`)
		if b, err := p.files.ReadFile(OLPath); err == nil {
			var dataMap map[string]any
			if yaml.Unmarshal(b, &dataMap) == nil {

				overlayMap := map[string]any{}
				for k, v := range dataMap {
					overlayMap[fmt.Sprintf(`Modules.%s.%s`, p.name, k)] = v
				}
				configs.AddOverlayOverrides(overlayMap)

			}
		}

		if p.Callbacks.onLoad != nil {
			p.Callbacks.onLoad()
		}
	}

	if pluginCt == 0 {
		mudlog.Warn("plugins", "loadedCount", pluginCt, "error", "Did you forget to run \"go generate\" before compiling the server? This error can be ignored.")
	} else {
		mudlog.Info("plugins", "loadedCount", pluginCt)
	}
}

func Save() {

	pluginCt := 0
	for _, p := range registry {

		if p.Callbacks.onSave != nil {
			p.Callbacks.onSave()
			pluginCt++
		}

	}
	mudlog.Info("plugins", "saveCount", pluginCt)
}

func GetPluginRegistry() pluginRegistry {
	return registry
}

func OnNetConnect(n NetConnection) {

	for _, p := range registry {
		if p.Callbacks.onNetConnect != nil {
			p.Callbacks.onNetConnect(n)
		}
	}

}

func ReadFile(dfPath string) ([]byte, error) {
	return registry.ReadFile(dfPath)
}
