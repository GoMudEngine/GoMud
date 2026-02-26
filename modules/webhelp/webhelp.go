package webhelp

import (
	"embed"
	"net/http"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/ansitags"
)

var (

	//////////////////////////////////////////////////////////////////////
	// NOTE: The below //go:embed directive is important!
	// It embeds the relative path into the var below it.
	//////////////////////////////////////////////////////////////////////

	//go:embed files/*
	files embed.FS
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {
	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	w := WebHelpModule{
		plug: plugins.New(`webhelp`, `1.0`),
	}

	//
	// Add the embedded filesystem
	//
	if err := w.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	w.plug.Web.WebPage(`Help`, `/help`, `help.html`, true, w.getHelpCategories)
	w.plug.Web.WebPage(`Help Topic`, `/help-details`, `help-details.html`, false, w.getHelpCommand)
}

//////////////////////////////////////////////////////////////////////
// NOTE: What follows is all custom code. For this module.
//////////////////////////////////////////////////////////////////////

// Using a struct gives a way to store longer term data.
type WebHelpModule struct {
	plug *plugins.Plugin
}

func (w *WebHelpModule) getHelpCategories(r *http.Request) map[string]any {

	data := map[string]any{}

	allCategories := []string{}
	allHelp := map[string][]keywords.HelpTopic{}

	for _, command := range keywords.GetAllHelpTopicInfo() {

		if command.AdminOnly {
			continue
		}

		// Skip topics that don't have a help template file,
		// matching the in-game help command behavior.
		templateFile := `help/` + keywords.TryHelpAlias(command.Command)
		if !templates.Exists(templateFile) {
			continue
		}

		category := command.Category
		if command.Type == "skill" {
			category = "skills"
		}

		if _, ok := allHelp[category]; !ok {
			allCategories = append(allCategories, category)
			allHelp[category] = []keywords.HelpTopic{}
		}

		allHelp[category] = append(allHelp[category], command)
	}

	sort.Strings(allCategories)
	for name, arr := range allHelp {

		sort.Slice(arr, func(i, j int) bool {
			return arr[i].Command > arr[j].Command
		})

		allHelp[name] = arr
	}

	data[`categories`] = allCategories
	data[`topics`] = allHelp

	return data

}

func (w *WebHelpModule) getHelpCommand(r *http.Request) map[string]any {

	searchTerm := r.URL.Query().Get("search")
	searchTerm = strings.TrimSpace(searchTerm)

	data := map[string]any{}
	data[`error`] = ``
	data[`topic`] = searchTerm

	if searchTerm == `` || searchTerm == `help` { // skip empty searches and circular help searches
		data[`error`] = `"` + searchTerm + `" Not Found`
	} else {
		contents, err := usercommands.GetHelpContents(searchTerm)
		if err != nil {
			data[`error`] = `"` + searchTerm + `" Not Found`
		} else {
			data[`contents`] = ansitags.Parse(contents, ansitags.HTML)
		}
	}

	return data

}
