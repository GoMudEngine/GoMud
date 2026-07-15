package achievements

import (
	"embed"
	"net/http"

	achdefs "github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

//go:embed files/*
var files embed.FS

// module is a thin plugin that only serves the achievements catalog web page.
// The poll (internal/hooks) and the `achievements` command (internal/usercommands)
// live elsewhere; this exists solely for the web surface.
type module struct {
	plug *plugins.Plugin
}

func init() {
	m := module{plug: plugins.New(`achievements`, `1.0`)}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.Web.WebPage(`Achievements`, `/achievements`, `achievements.html`, true, m.webData)
}

type webAchievement struct {
	Name        string
	Description string
	Points      int
}

type webCategory struct {
	Name         string
	Achievements []webAchievement
}

// webData returns the achievement catalog grouped by category for the web page.
func (m *module) webData(r *http.Request) map[string]any {
	order := []string{
		achdefs.CategoryCombat, achdefs.CategoryExploration, achdefs.CategoryWealth,
		achdefs.CategoryProgression, achdefs.CategoryQuests,
	}
	var cats []webCategory
	for _, c := range order {
		var list []webAchievement
		for _, d := range achdefs.All() {
			if d.Category == c {
				list = append(list, webAchievement{Name: d.Name, Description: d.Description, Points: d.Points})
			}
		}
		if len(list) > 0 {
			cats = append(cats, webCategory{Name: capitalize(c), Achievements: list})
		}
	}
	return map[string]any{"categories": cats}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
