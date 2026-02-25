package leaderboards

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	t := LeaderboardModule{
		plug: plugins.New(`leaderboards`, `1.0`),
	}

	//
	// Add the embedded filesystem
	//
	if err := t.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	//
	// Register any user/mob commands
	//
	t.plug.AddUserCommand(`leaderboard`, t.leaderboardCommand, true, false)

	//
	// Register callbacks for load/unload
	//
	t.plug.Callbacks.SetOnLoad(t.loadLBs)
	t.plug.Callbacks.SetOnSave(t.saveLBs)

	t.plug.Web.WebPage(`Leaderboards`, `/leaderboards`, `leaderboards.html`, true, t.webLeaderboardData)

	events.RegisterListener(events.NewRound{}, t.newRoundHandler)

}

//////////////////////////////////////////////////////////////////////
// NOTE: What follows is all custom code. For this module.
//////////////////////////////////////////////////////////////////////

// Using a struct gives a way to store longer term data.
type LeaderboardModule struct {

	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin

	lastCalculated time.Time // When the LB's were last generated

	LBSize            int
	GoldEnabled       bool
	ExperienceEnabled bool
	KillsEnabled      bool

	LB_Gold       leaderboardData `yaml:"LB_Gold,omitempty"`
	LB_Experience leaderboardData `yaml:"LB_Experience,omitempty"`
	LB_Kills      leaderboardData `yaml:"LB_Kills,omitempty"`
}

func (l *LeaderboardModule) webLeaderboardData(r *http.Request) map[string]any {

	data := map[string]any{}

	data[`leaderboards`] = l.getCurrentLeaderboards()

	return data

}

func (l *LeaderboardModule) loadLBs() {

	l.plug.ReadIntoStruct(`latest-leaderboards`, &l)

	l.GoldEnabled = true
	l.LB_Gold = leaderboardData{Name: `Gold`, ValueColor: `experience`}

	l.ExperienceEnabled = true
	l.LB_Experience = leaderboardData{Name: `Experience`, ValueColor: `gold`}

	l.KillsEnabled = true
	l.LB_Kills = leaderboardData{Name: `Kills`, ValueColor: `red-bold`}
}

func (l *LeaderboardModule) saveLBs() {
	l.plug.WriteStruct(`latest-leaderboards`, l)
}

func (l *LeaderboardModule) leaderboardCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	for _, lb := range l.getCurrentLeaderboards() {

		title := fmt.Sprintf(`%s Leaderboard`, lb.Name)

		headers := []string{`Rank`, `Character`, `Profession`, lb.Name}

		rows := [][]string{}

		valueFormatting := `%s`
		if lb.ValueColor != `` {
			valueFormatting = `<ansi fg="` + lb.ValueColor + `">%s</ansi>`
		}

		formatting := []string{
			`<ansi fg="red">%s</ansi>`,
			`<ansi fg="username">%s</ansi>`,
			`<ansi fg="white-bold">%s</ansi>`,
			valueFormatting,
		}

		for i, entry := range lb.Top {

			if entry.UserId == 0 {
				continue
			}

			newRow := []string{`#` + strconv.Itoa(i+1), entry.CharacterName, entry.CharacterClass, util.FormatNumber(entry.ScoreValue)}

			rows = append(rows, newRow)
		}

		searchResultsTable := templates.GetTable(title, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText("\n")
		user.SendText(tplTxt)

	}
	return true, nil
}

func (l *LeaderboardModule) Reset(maxSize int) {
	l.LB_Gold.Reset(maxSize)
	l.LB_Experience.Reset(maxSize)
	l.LB_Kills.Reset(maxSize)
}

func (l *LeaderboardModule) RefreshConfig() {

	l.LBSize = 10
	if size, ok := l.plug.Config.Get(`Size`).(int); ok {
		l.LBSize = size
	}

	if goldEnabled, ok := l.plug.Config.Get(`GoldEnabled`).(bool); ok {
		l.GoldEnabled = goldEnabled
	}

	if xpEnabled, ok := l.plug.Config.Get(`ExperienceEnabled`).(bool); ok {
		l.ExperienceEnabled = xpEnabled
	}

	if killsEnabled, ok := l.plug.Config.Get(`KillsEnabled`).(bool); ok {
		l.KillsEnabled = killsEnabled
	}
}

func (l *LeaderboardModule) Update() {
	start := time.Now()

	l.Reset(l.LBSize)

	userCount := 0
	characterCount := 0

	for _, u := range users.GetAllActiveUsers() {

		userCount++
		characterCount++

		if l.GoldEnabled {
			l.LB_Gold.Consider(u.UserId, *u.Character, u.Character.Gold+u.Character.Bank)
		}


		if l.KillsEnabled {
			l.LB_Kills.Consider(u.UserId, *u.Character, u.Character.KD.TotalKills)
		}

		for _, char := range characters.LoadAlts(u.UserId) {

			characterCount++

			if l.GoldEnabled {
				l.LB_Gold.Consider(u.UserId, char, char.Gold+char.Bank)
			}


			if l.KillsEnabled {
				l.LB_Kills.Consider(u.UserId, char, char.KD.TotalKills)
			}

		}

	}

	// Check offline users
	users.SearchOfflineUsers(func(u *users.UserRecord) bool {

		userCount++
		characterCount++

		if l.GoldEnabled {
			l.LB_Gold.Consider(u.UserId, *u.Character, u.Character.Gold+u.Character.Bank)
		}


		if l.KillsEnabled {
			l.LB_Kills.Consider(u.UserId, *u.Character, u.Character.KD.TotalKills)
		}

		for _, char := range characters.LoadAlts(u.UserId) {

			characterCount++

			if l.GoldEnabled {
				l.LB_Gold.Consider(u.UserId, char, char.Gold+char.Bank)
			}


			if l.KillsEnabled {
				l.LB_Kills.Consider(u.UserId, char, char.KD.TotalKills)
			}

		}

		return true
	})

	mudlog.Info("leaderboard.Update()", "user-processed", userCount, "characters-processed", characterCount, "Time Taken", time.Since(start))

	l.lastCalculated = time.Now()
}

func (l *LeaderboardModule) newRoundHandler(e events.Event) events.ListenerReturn {
	/*
		// Don't really care about the event data for this

		evt, typeOk := e.(events.NewRound)
		if !typeOk {
			return false // Return false to stop halt the event chain for this event
		}
	*/
	if time.Since(l.lastCalculated).Minutes() >= 15 {
		l.Update()
	}

	return events.Continue
}

func (l *LeaderboardModule) getCurrentLeaderboards() []leaderboardData {

	l.RefreshConfig()

	if l.lastCalculated.IsZero() {
		l.Update()
	}

	ret := []leaderboardData{}

	if l.GoldEnabled {
		ret = append(ret, l.LB_Gold)
	}

	if l.ExperienceEnabled {
		ret = append(ret, l.LB_Experience)
	}

	if l.KillsEnabled {
		ret = append(ret, l.LB_Kills)
	}

	return ret
}

type leaderboardEntry struct {
	UserId         int    `yaml:"UserId,omitempty"`
	CharacterName  string `yaml:"CharacterName,omitempty"`
	CharacterClass string `yaml:"CharacterClass,omitempty"`
	ScoreValue     int    `yaml:"ScoreValue,omitempty"`
}

type leaderboardData struct {
	Name        string
	ValueColor  string             // Numeric 256 color or ansitags alias
	Top         []leaderboardEntry `yaml:"Top,omitempty"`
	MaxSize     int
	LowestValue int
}

func (l *leaderboardData) Reset(size int) {
	l.MaxSize = size
	l.Top = make([]leaderboardEntry, l.MaxSize)
	l.LowestValue = 0
}

func (l *leaderboardData) Consider(userId int, char characters.Character, val int) {
	if val == 0 {
		return
	}

	if val < l.LowestValue && l.Top[l.MaxSize-1].UserId != 0 {
		return
	}

	addPosition := -1
	for i := 0; i < l.MaxSize; i++ {

		if l.Top[i].UserId == 0 {
			addPosition = i
			break
		}

		if val > l.Top[i].ScoreValue {
			addPosition = i
			break
		}

	}

	if addPosition > -1 {

		for i := l.MaxSize - 2; i >= addPosition; i-- {
			l.Top[i+1] = l.Top[i]
		}

		// just accept it
		l.Top[addPosition] = leaderboardEntry{
			UserId:         userId,
			CharacterName:  char.Name,
			CharacterClass: skills.GetProfession(char.GetAllSkillRanks()),
			ScoreValue:     val,
		}

		if l.LowestValue == 0 || val < l.LowestValue {
			l.LowestValue = val
		}

	}
}
