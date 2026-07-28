package inputhandlers

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// mudSkillCount is the fixed number of player skills in DOGMud (see CLAUDE.md
// "Skills (10 total)"). Hardcoded because skills are a fixed design set.
const mudSkillCount = 10

// MSSPInputs is the plain-Go snapshot buildMSSPFields turns into wire fields.
// Kept dependency-free so the field logic is trivially unit-testable.
type MSSPInputs struct {
	Enabled  bool
	Name     string
	Website  string
	Genre    string
	Gameplay []string
	Status   string
	Language string
	Family   string
	Location string
	Created  string
	Contact  string
	Hostname string
	Port     string

	Players    int
	UptimeUnix int64

	Rooms   int
	Mobiles int
	Objects int
	Skills  int
	Races   int
}

// buildMSSPFields turns a snapshot into the ordered MSSP field list. Empty
// string values and non-positive world counts are omitted; PLAYERS/UPTIME and
// the 0/1 capability flags are always sent. Returns nil when disabled.
func buildMSSPFields(in MSSPInputs) []term.MSSPField {
	if !in.Enabled {
		return nil
	}

	fields := []term.MSSPField{}
	add := func(name string, values ...string) {
		vv := make([]string, 0, len(values))
		for _, v := range values {
			if v != "" {
				vv = append(vv, v)
			}
		}
		if len(vv) == 0 {
			return
		}
		fields = append(fields, term.MSSPField{Name: name, Values: vv})
	}
	addNum := func(name string, n int) {
		if n > 0 {
			add(name, strconv.Itoa(n))
		}
	}

	// Required / live
	add("NAME", in.Name)
	add("PLAYERS", strconv.Itoa(in.Players)) // 0 is meaningful, always sent
	add("UPTIME", strconv.FormatInt(in.UptimeUnix, 10))

	// Descriptive
	add("CODEBASE", "GoMud")
	add("WEBSITE", in.Website)
	add("CONTACT", in.Contact)   // empty => omitted (privacy)
	add("HOSTNAME", in.Hostname) // empty => omitted
	add("PORT", in.Port)         // empty => omitted
	add("CREATED", in.Created)
	add("LANGUAGE", in.Language)
	add("LOCATION", in.Location)
	add("FAMILY", in.Family)
	add("GENRE", in.Genre)
	add("GAMEPLAY", in.Gameplay...)
	add("STATUS", in.Status)

	// World
	addNum("ROOMS", in.Rooms)
	addNum("MOBILES", in.Mobiles)
	addNum("OBJECTS", in.Objects)
	addNum("SKILLS", in.Skills)
	addNum("RACES", in.Races)

	// Capability flags (0/1 are meaningful, always sent)
	add("ANSI", "1")
	add("GMCP", "1")
	add("MSP", "1")
	add("UTF-8", "1")
	add("VT100", "1")
	add("XTERM 256 COLORS", "1")
	add("MCCP", "0")
	add("SSL", "0")
	add("PAY TO PLAY", "0")
	add("PAY FOR PERKS", "0")

	return fields
}

// buildMSSPTextReply renders the field list as the plaintext MSSP block some
// crawlers (e.g. Grapevine's checker) request by sending "mssp-request" at
// the login prompt instead of negotiating the telnet option:
//
//	MSSP-REPLY-START
//	NAME<TAB>value
//	...
//	MSSP-REPLY-END
//
// Multi-value fields tab-join their values (consumers split the whole line on
// tabs as [name, values...]). Returns nil for an empty field list (disabled).
func buildMSSPTextReply(fields []term.MSSPField) []byte {
	if len(fields) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("MSSP-REPLY-START\r\n")
	for _, f := range fields {
		sb.WriteString(f.Name)
		for _, v := range f.Values {
			sb.WriteByte('\t')
			sb.WriteString(v)
		}
		sb.WriteString("\r\n")
	}
	sb.WriteString("MSSP-REPLY-END\r\n")
	return []byte(sb.String())
}

// MSSPTextRequestIntercept is the login-prompt hook for the plaintext MSSP
// variant: when the submitted "username" is mssp-request and MSSP is enabled,
// it writes the text block and closes the connection. Returns true when the
// input was consumed (the prompt sequence must stop).
func MSSPTextRequestIntercept(input string, clientInput *connections.ClientInput) bool {
	if !strings.EqualFold(strings.TrimSpace(input), "mssp-request") {
		return false
	}
	reply := buildMSSPTextReply(gatherMSSPFields())
	if reply == nil {
		return false // MSSP disabled — fall through to normal (failing) login
	}
	connections.SendTo(reply, clientInput.ConnectionId)
	mudlog.Info("MSSP", "info", "plaintext mssp-request served", "connectionId", clientInput.ConnectionId)
	connections.Remove(clientInput.ConnectionId)
	return true
}

// gatherMSSPFields reads live server state + config into a snapshot and builds
// the field list. Returns nil when MSSP is disabled.
func gatherMSSPFields() []term.MSSPField {
	cfg := configs.GetServerConfig()
	return buildMSSPFields(MSSPInputs{
		Enabled:  bool(cfg.MSSP.Enabled),
		Name:     string(cfg.MudName),
		Website:  string(cfg.MSSP.Website),
		Genre:    string(cfg.MSSP.Genre),
		Gameplay: []string(cfg.MSSP.Gameplay),
		Status:   string(cfg.MSSP.Status),
		Language: string(cfg.MSSP.Language),
		Family:   string(cfg.MSSP.Family),
		Location: string(cfg.MSSP.Location),
		Created:  string(cfg.MSSP.Created),
		Contact:  string(cfg.MSSP.Contact),
		Hostname: string(cfg.MSSP.Hostname),
		Port:     string(cfg.MSSP.Port),

		Players:    len(users.GetOnlineUserIds()),
		UptimeUnix: util.GetServerStartUnix(),

		Rooms:   len(rooms.GetAllRoomIds()),
		Mobiles: len(mobs.GetAllMobNames()),
		Objects: len(items.GetAllItemNames()),
		Skills:  mudSkillCount,
		Races:   len(species.GetAllSpecies()),
	})
}
