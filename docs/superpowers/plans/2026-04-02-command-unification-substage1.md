# Command Unification — Substage 1: Actor Interface + Registry Audit + Pattern

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define the shared Actor interface, implement it for users and mobs, extract say/emote/go as proof of pattern, build the registry audit, and remove the dead `train` command.

**Architecture:** New `internal/actions/` package with an Actor interface that both UserRecord and Mob implement. Shared action functions take an Actor and handle the mechanical work. User/mob commands become thin wrappers that handle actor-specific concerns (mute checks, darkness awareness, GMCP, player-presence optimization) then delegate to the shared action. A startup registry audit compares command sets and warns about unintentional gaps.

**Tech Stack:** Go, testify/assert, existing event system

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/actions/actor.go` | Actor interface definition |
| `internal/actions/actor_user.go` | UserRecord adapter implementing Actor |
| `internal/actions/actor_mob.go` | Mob adapter implementing Actor |
| `internal/actions/emote_aliases.go` | Shared emote alias map (deduplicated) |
| `internal/actions/say.go` | Shared say logic |
| `internal/actions/emote.go` | Shared emote logic |
| `internal/actions/divergences.go` | Intentional divergence allowlist + registry audit |
| `internal/actions/actions_test.go` | Test infrastructure + parity tests |
| `internal/usercommands/say.go` | Thin wrapper (modify existing) |
| `internal/usercommands/emote.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/say.go` | Thin wrapper (modify existing) |
| `internal/mobcommands/emote.go` | Thin wrapper (modify existing) |
| `internal/usercommands/train.go` | DELETE |
| `internal/usercommands/usercommands.go` | Remove `train` from registry |

---

### Task 1: Create Actor Interface

**Files:**
- Create: `internal/actions/actor.go`

- [ ] **Step 1: Create the actions package and Actor interface**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Actor is the shared interface for any entity that can perform actions
// in the game world — players and mobs alike. The shared action functions
// in this package operate on Actors, and the usercommands/mobcommands
// packages provide thin wrappers that handle actor-specific concerns.
type Actor interface {
	// GetCharacter returns the character data (stats, buffs, inventory, etc.)
	GetCharacter() *characters.Character

	// GetRoom returns the room the actor is currently in.
	GetRoom() *rooms.Room

	// SendText sends a message only to this actor.
	// For players: delivers to their terminal.
	// For mobs: no-op (mobs have no connection).
	SendText(msg string)

	// SendRoomText broadcasts a message to all players in the actor's room.
	// If excludeSelf is true, the actor (if a player) is excluded.
	// Respects darkness for mob actors (nightvision-only in dark rooms).
	SendRoomText(msg string, excludeSelf bool)

	// SendRoomCommunication broadcasts a communication message to the room.
	// Like SendRoomText but respects mute/deafen flags on recipients.
	SendRoomCommunication(msg string, excludeSelf bool)

	// GetName returns the actor's display name.
	GetName() string

	// IsPlayer returns true for player actors, false for mob actors.
	IsPlayer() bool

	// GetUserId returns the user ID (0 for mobs).
	GetUserId() int

	// GetMobInstanceId returns the mob instance ID (0 for players).
	GetMobInstanceId() int
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/actions/actor.go
git commit -m "feat: define Actor interface in internal/actions"
```

---

### Task 2: Implement Actor for UserRecord

**Files:**
- Create: `internal/actions/actor_user.go`

- [ ] **Step 1: Create the UserRecord adapter**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// UserActor wraps a UserRecord to implement the Actor interface.
type UserActor struct {
	User *users.UserRecord
	Room *rooms.Room
}

func NewUserActor(user *users.UserRecord, room *rooms.Room) *UserActor {
	return &UserActor{User: user, Room: room}
}

func (a *UserActor) GetCharacter() *characters.Character {
	return a.Character()
}

func (a *UserActor) GetRoom() *rooms.Room {
	return a.Room
}

func (a *UserActor) SendText(msg string) {
	a.User.SendText(msg)
}

func (a *UserActor) SendRoomText(msg string, excludeSelf bool) {
	if excludeSelf {
		a.Room.SendText(msg, a.User.UserId)
	} else {
		a.Room.SendText(msg)
	}
}

func (a *UserActor) SendRoomCommunication(msg string, excludeSelf bool) {
	if excludeSelf {
		a.Room.SendTextCommunication(msg, a.User.UserId)
	} else {
		a.Room.SendTextCommunication(msg)
	}
}

func (a *UserActor) GetName() string {
	return a.User.Character.Name
}

func (a *UserActor) IsPlayer() bool {
	return true
}

func (a *UserActor) GetUserId() int {
	return a.User.UserId
}

func (a *UserActor) GetMobInstanceId() int {
	return 0
}
```

Note: `GetCharacter()` should call `a.User.Character` — fix the method body:

```go
func (a *UserActor) GetCharacter() *characters.Character {
	return a.User.Character
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/actions/actor_user.go
git commit -m "feat: implement Actor interface for UserRecord"
```

---

### Task 3: Implement Actor for Mob

**Files:**
- Create: `internal/actions/actor_mob.go`

- [ ] **Step 1: Create the Mob adapter**

The key difference: Mob.Character is a value (not pointer), and room text
needs darkness-aware broadcasting. The darkness logic from
`mobcommands/darkness.go` needs to be accessible here.

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobActor wraps a Mob to implement the Actor interface.
type MobActor struct {
	Mob  *mobs.Mob
	Room *rooms.Room
}

func NewMobActor(mob *mobs.Mob, room *rooms.Room) *MobActor {
	return &MobActor{Mob: mob, Room: room}
}

func (a *MobActor) GetCharacter() *characters.Character {
	return &a.Mob.Character
}

func (a *MobActor) GetRoom() *rooms.Room {
	return a.Room
}

func (a *MobActor) SendText(msg string) {
	// Mobs have no connection — no-op.
}

func (a *MobActor) SendRoomText(msg string, excludeSelf bool) {
	sendRoomTextDarknessAware(a.Room, msg)
}

func (a *MobActor) SendRoomCommunication(msg string, excludeSelf bool) {
	// Mobs don't respect mute/deafen — use regular room text.
	sendRoomTextDarknessAware(a.Room, msg)
}

func (a *MobActor) GetName() string {
	return a.Mob.Character.Name
}

func (a *MobActor) IsPlayer() bool {
	return false
}

func (a *MobActor) GetUserId() int {
	return 0
}

func (a *MobActor) GetMobInstanceId() int {
	return a.Mob.InstanceId
}

// sendRoomTextDarknessAware is the shared darkness helper, moved from
// mobcommands/darkness.go so the actions package can use it.
func sendRoomTextDarknessAware(room *rooms.Room, msg string, excludeUserIds ...int) {
	if room.GetVisibility() >= 1 {
		room.SendText(msg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		excluded := false
		for _, eid := range excludeUserIds {
			if uid == eid {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(msg)
		}
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/actions/actor_mob.go
git commit -m "feat: implement Actor interface for Mob"
```

---

### Task 4: Shared Emote Aliases

**Files:**
- Create: `internal/actions/emote_aliases.go`

- [ ] **Step 1: Move the duplicated emote alias map to the shared package**

```go
package actions

// EmoteAliases maps emote shortcut names to their display text.
// Previously duplicated in both usercommands/emote.go and
// mobcommands/emote.go — now a single source of truth.
var EmoteAliases = map[string]string{
	"armcross": "crosses their arms.",
	"backflip": "does a backflip.",
	"beam":     "beams with pride.",
	"blink":    "blinks in surprise.",
	"blush":    "blushes slightly.",
	"bounce":   "bounces up and down.",
	"bow":      "bows gracefully.",
	"brood":    "broods in the corner.",
	"chew":     "chews thoughtfully.",
	"cheer":    "cheers loudly.",
	"chuckle":  "chuckles softly.",
	"clap":     "claps enthusiastically.",
	"cringe":   "cringes in embarrassment.",
	"cry":      "cries softly.",
	"dance":    "starts dancing.",
	"daydream": "daydreams wistfully.",
	"doze":     "dozes off for a moment.",
	"drum":     "drums their fingers.",
	"duck":     "ducks to avoid something.",
	"eyeroll":  "rolls their eyes.",
	"eyebrow":  "raises an eyebrow.",
	"facepalm": "facepalms in disbelief.",
	"flail":    "flails their arms.",
	"flex":     "flexes their muscles.",
	"flinch":   "flinches unexpectedly.",
	"flirt":    "is feeling flirty.",
	"flutter":  "flutters their eyelashes.",
	"frown":    "frowns deeply.",
	"giggle":   "giggles softly.",
	"glare":    "glares menacingly.",
	"grin":     "grins cheekily.",
	"groan":    "groans in frustration.",
	"headache": "rubs their temples. They seem to be getting a headache.",
	"hum":      "hums a familiar tune.",
	"jump":     "jumps in excitement.",
	"juggle":   "juggles a few items skillfully.",
	"laugh":    "laughs heartily.",
	"listen":   "listens intently.",
	"meditate": "meditates peacefully.",
	"murmur":   "murmurs something under their breath.",
	"nod":      "nods in agreement.",
	"pace":     "paces back and forth.",
	"point":    "points at something.",
	"ponder":   "is pondering something.",
	"pout":     "pouts adorably.",
	"prance":   "prances around.",
	"roar":     "roars mightily.",
	"salute":   "salutes respectfully.",
	"scratch":  "scratches their head.",
	"shake":    "shakes their head.",
	"shiver":   "shivers from the cold... or perhaps something else.",
	"shudder":  "shudders in fear.",
	"shrug":    "shrugs nonchalantly.",
	"shush":    "shushes everyone.",
	"sigh":     "sighs deeply.",
	"sing":     "sings a tune.",
	"sit":      "sits down for a think.",
	"skip":     "skips joyfully.",
	"slap":     "slaps their forehead.",
	"slouch":   "slouches lazily.",
	"smile":    "smiles warmly.",
	"snicker":  "snickers quietly.",
	"sniff":    "sniffs the air.",
	"snore":    "snores loudly.",
	"spin":     "spins around dizzyingly.",
	"stand":    "stands up straight.",
	"stomp":    "stomps their foot.",
	"stretch":  "stretches their limbs.",
	"stumble":  "stumbles a bit.",
	"swim":     "swims around.",
	"tap":      "taps their foot impatiently.",
	"think":    "thinks hard.",
	"tilt":     "tilts their head curiously.",
	"tremble":  "trembles in anticipation.",
	"trip":     "trips over their own feet.",
	"twirl":    "twirls around with a flourish.",
	"wave":     "waves.",
	"whine":    "whines pitifully.",
	"whistle":  "whistles a catchy melody.",
	"yawn":     "yawns sleepily.",
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/actions/emote_aliases.go
git commit -m "refactor: extract shared emote aliases to internal/actions"
```

---

### Task 5: Shared Say Action

**Files:**
- Create: `internal/actions/say.go`
- Modify: `internal/usercommands/say.go`
- Modify: `internal/mobcommands/say.go`

The shared action handles the mechanical "produce say text in the room."
Actor-specific concerns stay in the wrappers.

**What stays in user wrapper:** Muted check, drunk text, self-message ("You say"), ANSI fg="saytext" (player color), Communication event with SourceUserId.

**What stays in mob wrapper:** PlayerCt() < 1 early-out, darkness-aware audio broadcasting (named vs anonymous), ANSI fg="saytext-mob" (mob color), Communication event with SourceMobInstanceId.

**What's shared:** The sneaking check (both sides check Hidden buff and use "someone says" when hidden).

Given the divergences in how say actually works (different ANSI colors, different broadcast methods, darkness-aware audio for mobs), the shared say function should handle the common pattern while the wrappers handle the presentation differences.

- [ ] **Step 1: Create the shared say action**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// SayResult contains the data from a Say action for the wrapper to use.
type SayResult struct {
	IsSneaking bool
	Text       string // The (possibly modified) text being said
}

// Say handles the shared logic for speaking aloud in a room.
// It checks the hidden buff and fires the exit echo + communication event.
// The caller (user/mob wrapper) is responsible for:
//   - Any pre-checks (muted, drunk text, player presence)
//   - Formatting and delivering room text (ANSI colors differ)
//   - The self-message (users only)
func Say(actor Actor, text string) SayResult {
	isSneaking := actor.GetCharacter().HasBuffFlag(buffs.Hidden)

	room := actor.GetRoom()
	room.SendTextToExits(`You hear someone talking.`, true)

	events.AddToQueue(events.Communication{
		SourceUserId:        actor.GetUserId(),
		SourceMobInstanceId: actor.GetMobInstanceId(),
		CommType:            `say`,
		Name:                actor.GetName(),
		Message:             text,
	})

	return SayResult{
		IsSneaking: isSneaking,
		Text:       text,
	}
}

// FormatSayText formats the say message for room display.
// nameColor is "username" for players, "mobname" for mobs.
// textColor is "saytext" for players, "saytext-mob" for mobs.
func FormatSayText(name string, text string, isSneaking bool, nameColor string, textColor string) string {
	if isSneaking {
		return util.SplitStringNL(
			fmt.Sprintf(`someone says, "<ansi fg="%s">%s</ansi>"`, textColor, text), 80)
	}
	return util.SplitStringNL(
		fmt.Sprintf(`<ansi fg="%s">%s</ansi> says, "<ansi fg="%s">%s</ansi>"`, nameColor, name, textColor, text), 80)
}
```

- [ ] **Step 2: Update user say wrapper**

Replace `internal/usercommands/say.go`:

```go
package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Say(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Muted {
		user.SendText(`You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	if user.Character.HasBuffFlag(buffs.Drunk) {
		rest = drunkify(rest)
	}

	actor := actions.NewUserActor(user, room)
	result := actions.Say(actor, rest)

	// Room message with player-specific ANSI colors
	msg := actions.FormatSayText(user.Character.Name, result.Text, result.IsSneaking, "username", "saytext")
	room.SendTextCommunication(msg, user.UserId)

	// Self message
	selfMsg := fmt.Sprintf(`You say, "<ansi fg="saytext">%s</ansi>"`, result.Text)
	user.SendText(util.SplitStringNL(selfMsg, 80))

	return true, nil
}
```

Keep the existing `drunkify` function unchanged in the same file.

Note: add the missing `buffs` import:
```go
"github.com/GoMudEngine/GoMud/internal/buffs"
```

- [ ] **Step 3: Update mob say wrapper**

Replace `internal/mobcommands/say.go`:

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Say(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if room.PlayerCt() < 1 {
		return true, nil
	}

	actor := actions.NewMobActor(mob, room)
	result := actions.Say(actor, rest)

	// Darkness-aware room message with mob-specific ANSI colors
	if result.IsSneaking {
		msg := actions.FormatSayText("", result.Text, true, "", "saytext-mob")
		room.SendText(util.SplitStringNL(msg, 80))
	} else {
		anonMsg := actions.FormatSayText("", result.Text, true, "", "saytext-mob")
		namedMsg := actions.FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob")
		sendAudioRoomText(room, mob,
			util.SplitStringNL(anonMsg, 80),
			util.SplitStringNL(namedMsg, 80))
	}

	return true, nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 5: Commit**

```bash
git add internal/actions/say.go internal/usercommands/say.go internal/mobcommands/say.go
git commit -m "refactor: extract shared Say action, user/mob wrappers delegate"
```

---

### Task 6: Shared Emote Action

**Files:**
- Create: `internal/actions/emote.go`
- Modify: `internal/usercommands/emote.go`
- Modify: `internal/mobcommands/emote.go`

- [ ] **Step 1: Create shared emote action**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// EmoteResult contains data from an Emote action.
type EmoteResult struct {
	IsAlias   bool
	AliasText string // The resolved alias text, empty if not an alias
}

// Emote checks if the input is an emote alias and returns the result.
// The caller handles formatting and delivery (ANSI colors and broadcast
// method differ between players and mobs).
func Emote(rest string) EmoteResult {
	if aliasText, ok := EmoteAliases[rest]; ok {
		return EmoteResult{IsAlias: true, AliasText: aliasText}
	}
	return EmoteResult{IsAlias: false}
}

// FormatEmoteText formats an emote for room display.
// nameColor is "username" for players, "mobname" for mobs.
func FormatEmoteText(name string, emoteText string, nameColor string) string {
	return util.SplitStringNL(
		fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="20">%s</ansi>`, nameColor, name, emoteText), 80)
}
```

- [ ] **Step 2: Update user emote wrapper**

Replace `internal/usercommands/emote.go` — remove the local `emoteAliases` map, use shared:

```go
package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Emote(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(rest) == 0 {
		user.SendText("You emote.")
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> emotes.`, user.Character.Name),
			user.UserId,
		)
		return true, nil
	}

	result := actions.Emote(rest)

	// Emote aliases bypass mute (pre-written, not player text)
	if result.IsAlias {
		user.SendText(actions.FormatEmoteText(user.Character.Name, result.AliasText, "username"))
		room.SendText(
			actions.FormatEmoteText(user.Character.Name, result.AliasText, "username"),
			user.UserId,
		)
		return true, nil
	}

	if user.Muted {
		user.SendText(`You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	if rest[0] == '@' && len(rest) > 1 {
		rest = rest[1:]
	} else {
		user.SendText(actions.FormatEmoteText(user.Character.Name, rest, "username"))
	}

	room.SendTextCommunication(
		actions.FormatEmoteText(user.Character.Name, rest, "username"),
		user.UserId,
	)

	return true, nil
}
```

- [ ] **Step 3: Update mob emote wrapper**

Replace `internal/mobcommands/emote.go` — remove the local `emoteAliases` map, use shared:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Emote(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if room.PlayerCt() < 1 {
		return true, nil
	}

	if len(rest) == 0 {
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> emotes.`, mob.Character.Name))
		return true, nil
	}

	result := actions.Emote(rest)

	if result.IsAlias {
		sendRoomText(room,
			actions.FormatEmoteText(mob.Character.Name, result.AliasText, "mobname"))
		return true, nil
	}

	sendRoomText(room,
		actions.FormatEmoteText(mob.Character.Name, rest, "mobname"))

	return true, nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 5: Commit**

```bash
git add internal/actions/emote.go internal/usercommands/emote.go internal/mobcommands/emote.go
git commit -m "refactor: extract shared Emote action, deduplicate emote aliases"
```

---

### Task 7: Registry Audit

**Files:**
- Create: `internal/actions/divergences.go`
- Modify: `internal/usercommands/usercommands.go` (export command list)
- Modify: `internal/mobcommands/mobcommands.go` (already exports via GetAllMobCommands)

- [ ] **Step 1: Add GetAllUserCommands to usercommands package**

In `internal/usercommands/usercommands.go`, add after the existing
`GetCmdSuggestions` function:

```go
// GetAllUserCommands returns a list of all registered user command names.
func GetAllUserCommands() []string {
	result := []string{}
	for cmd := range userCommands {
		result = append(result, cmd)
	}
	return result
}

// IsAdminCommand returns true if the command is admin-only.
func IsAdminCommand(cmd string) bool {
	if info, ok := userCommands[cmd]; ok {
		return info.AdminOnly
	}
	return false
}
```

- [ ] **Step 2: Create divergences.go with allowlist and audit function**

```go
package actions

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// CommandLister is implemented by both usercommands and mobcommands packages.
type CommandLister interface {
	GetAllCommands() []string
}

// intentionalUserOnly lists user commands that intentionally have no mob
// equivalent, with a reason for each.
var intentionalUserOnly = map[string]string{
	// Admin commands
	"ai-flag":     "admin",
	"ai-list":     "admin",
	"badcommands": "admin",
	"buff":        "admin",
	"build":       "admin",
	"command":     "admin",
	"combatstats": "admin",
	"deafen":      "admin",
	"devtool":     "admin",
	"item":        "admin",
	"locate":      "admin",
	"mob":         "admin",
	"modify":      "admin",
	"mudmail":     "admin",
	"mute":        "admin",
	"paz":         "admin",
	"prepare":     "admin",
	"questdebug":  "admin",
	"questtoken":  "admin",
	"redescribe":  "admin",
	"reload":      "admin",
	"rename":      "admin",
	"room":        "admin",
	"server":      "admin",
	"setmotd":     "admin",
	"skillset":    "admin",
	"spawn":       "admin",
	"spell":       "admin",
	"syslogs":     "admin",
	"teleport":    "admin",
	"undeafen":    "admin",
	"unmute":      "admin",
	"zap":         "admin",
	"zone":        "admin",
	// UI commands (player terminal only)
	"afk":        "ui",
	"alias":      "ui",
	"bank":       "ui",
	"biome":      "ui",
	"bug":        "ui",
	"cancel":     "ui",
	"character":  "ui",
	"conditions": "ui",
	"consider":   "ui",
	"cooldowns":  "ui",
	"default":    "ui",
	"help":       "ui",
	"hint":       "ui",
	"history":    "ui",
	"inbox":      "ui",
	"inventory":  "ui",
	"keyring":    "ui",
	"killstats":  "ui",
	"macros":     "ui",
	"map":        "ui",
	"motd":       "ui",
	"mutations":  "ui",
	"online":     "ui",
	"password":   "ui",
	"print":      "ui",
	"printline":  "ui",
	"pvp":        "ui",
	"quests":     "ui",
	"quit":       "ui",
	"read":       "ui",
	"rep":        "ui",
	"report":     "ui",
	"save":       "ui",
	"set":        "ui",
	"setdesc":    "ui",
	"skills":     "ui",
	"spells":     "ui",
	"status":     "ui",
	"suggest":    "ui",
	"title":      "ui",
	"who":        "ui",
	// Player-only mechanics
	"assist":   "player-mechanic: party system",
	"party":    "player-mechanic: party system",
	"share":    "player-mechanic: party system",
	"reply":    "player-mechanic: whisper reply",
	"whisper":  "player-mechanic: private messaging",
	"target":   "player-mechanic: targeting UI",
	"start":    "player-mechanic: character creation",
	"zombieact": "player-mechanic: zombie state",
	// Aliases for existing commands
	"stomp":     "alias: kick",
	"knee":      "alias: kick",
	"tailsweep": "alias: trip",
}

// intentionalMobOnly lists mob commands that intentionally have no user
// equivalent.
var intentionalMobOnly = map[string]string{
	"aid":            "mob-ai: healing allies",
	"backstab":       "mob-ai: sneak attack",
	"befriend":       "mob-ai: charm mechanic",
	"callforhelp":    "mob-ai: summon nearby mobs",
	"charge":         "mob-ai: gap closer",
	"consume":        "mob-ai: eat items",
	"converse":       "mob-ai: dialogue system",
	"despawn":        "mob-ai: cleanup",
	"givequest":      "mob-ai: quest granting",
	"hamstring":      "mob-ai: debuff attack",
	"howl":           "mob-ai: area fear",
	"lookforaid":     "mob-ai: find healers",
	"lookfortrouble": "mob-ai: aggro scanning",
	"pathto":         "mob-ai: pathfinding",
	"portal":         "mob-ai: teleport mechanic",
	"replyto":        "mob-ai: dialogue response",
	"roar":           "mob-ai: intimidation",
	"sayto":          "mob-ai: targeted speech",
	"saytoonly":      "mob-ai: private speech",
	"throw":          "mob-ai: ranged attack",
	"wander":         "mob-ai: random movement",
}

// AuditCommandParity compares user and mob command registries and logs
// warnings for any command that exists on one side but not the other
// and is not in the intentional divergence allowlist.
func AuditCommandParity(userCommands []string, mobCommands []string) {
	userSet := make(map[string]bool, len(userCommands))
	for _, cmd := range userCommands {
		userSet[cmd] = true
	}
	mobSet := make(map[string]bool, len(mobCommands))
	for _, cmd := range mobCommands {
		mobSet[cmd] = true
	}

	var warnings []string

	// Check user commands missing from mob side
	for _, cmd := range userCommands {
		if mobSet[cmd] {
			continue
		}
		if _, ok := intentionalUserOnly[cmd]; ok {
			continue
		}
		warnings = append(warnings, "user command '"+cmd+"' has no mob equivalent (not in allowlist)")
	}

	// Check mob commands missing from user side
	for _, cmd := range mobCommands {
		if userSet[cmd] {
			continue
		}
		if _, ok := intentionalMobOnly[cmd]; ok {
			continue
		}
		warnings = append(warnings, "mob command '"+cmd+"' has no user equivalent (not in allowlist)")
	}

	sort.Strings(warnings)
	for _, w := range warnings {
		mudlog.Warning("CommandParity", w)
	}

	if len(warnings) == 0 {
		mudlog.Info("CommandParity",
			strings.Join([]string{
				"audit passed:",
				fmt.Sprintf("%d user commands", len(userCommands)),
				fmt.Sprintf("%d mob commands", len(mobCommands)),
				fmt.Sprintf("%d intentional user-only", len(intentionalUserOnly)),
				fmt.Sprintf("%d intentional mob-only", len(intentionalMobOnly)),
			}, " "))
	}
}
```

Add the missing `"fmt"` import at the top of the import block.

- [ ] **Step 3: Wire up the audit at startup**

Find where the server initializes commands (likely in `main.go` or an init
hook). Add after both command packages are loaded:

```go
actions.AuditCommandParity(
    usercommands.GetAllUserCommands(),
    mobcommands.GetAllMobCommands(),
)
```

Search for where `mobcommands` and `usercommands` are both imported and
find the right initialization spot. This may be in `internal/hooks/` or
the main server startup.

- [ ] **Step 4: Verify build and test**

Run: `go build ./...`
Expected: clean build. Server startup should log parity audit results.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/divergences.go internal/usercommands/usercommands.go
git commit -m "feat: command registry parity audit with intentional divergence allowlist"
```

---

### Task 8: Parity Test Framework

**Files:**
- Create: `internal/actions/actions_test.go`

- [ ] **Step 1: Create test infrastructure and first parity tests**

```go
package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmoteAliasLookup(t *testing.T) {
	result := Emote("bow")
	assert.True(t, result.IsAlias)
	assert.Equal(t, "bows gracefully.", result.AliasText)
}

func TestEmoteNonAlias(t *testing.T) {
	result := Emote("does something custom")
	assert.False(t, result.IsAlias)
	assert.Empty(t, result.AliasText)
}

func TestFormatSayTextNormal(t *testing.T) {
	msg := FormatSayText("Alice", "hello", false, "username", "saytext")
	assert.Contains(t, msg, "Alice")
	assert.Contains(t, msg, "hello")
	assert.Contains(t, msg, "says,")
}

func TestFormatSayTextSneaking(t *testing.T) {
	msg := FormatSayText("Alice", "hello", true, "username", "saytext")
	assert.Contains(t, msg, "someone says,")
	assert.NotContains(t, msg, "Alice")
}

func TestFormatEmoteText(t *testing.T) {
	msg := FormatEmoteText("Bob", "bows gracefully.", "mobname")
	assert.Contains(t, msg, "Bob")
	assert.Contains(t, msg, "bows gracefully.")
}

func TestAuditCommandParity_NoWarnings(t *testing.T) {
	// Both sides have the same commands — no warnings
	user := []string{"say", "emote", "go", "help"}
	mob := []string{"say", "emote", "go", "wander"}
	// help is UI (allowlisted), wander is mob-ai (allowlisted)
	// This should produce no warnings (captured via log, but
	// we're testing it doesn't panic)
	AuditCommandParity(user, mob)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/actions/ -v`
Expected: all tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/actions/actions_test.go
git commit -m "test: parity test framework for shared actions"
```

---

### Task 9: Remove Dead Train Command

**Files:**
- Delete: `internal/usercommands/train.go`
- Modify: `internal/usercommands/usercommands.go` (remove registry entry)

- [ ] **Step 1: Remove train from the command registry**

In `internal/usercommands/usercommands.go`, delete this line from the
`userCommands` map:

```go
		`train`:       {Train, false, false, false}, // Can't train in combat
```

- [ ] **Step 2: Delete the train command file**

```bash
rm internal/usercommands/train.go
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/train.go internal/usercommands/usercommands.go
git commit -m "chore: remove vestigial train command (does nothing)"
```

---

### Task 10: Verify Everything Together

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 2: Run all tests**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/combat/ -v`
Expected: all tests pass

- [ ] **Step 3: Manual smoke test**

Start the server locally and verify:
- `say hello` works as a player (should see "You say" + room broadcast)
- Mob NPCs still say things (dialogue, idle chatter)
- Emote aliases work (`bow`, `nod`, etc.)
- Startup log shows command parity audit results
- No warnings for commands that are in the allowlist

- [ ] **Step 4: Final commit if any fixups needed**

```bash
git add -A
git commit -m "fix: substage 1 smoke test fixups"
```
