# Code Cleanup 1.4b: Help Template Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the gap in help discoverability by adding a completeness test for command help templates, creating missing stubs, and indexing unindexed commands in `keywords.yaml`.

**Architecture:** The `usercommands` package already imports `devtools` (for zone consistency checks), so the new test cannot live in `devtools` — it would create an import cycle. Instead, the test lives in `internal/usercommands/helpfile_completeness_test.go` where it can access the package-private `userCommands` map directly. Three independent tasks, each with its own commit.

**Tech Stack:** Go, YAML

**Spec:** `docs/superpowers/specs/2026-04-16-code-cleanup-1.4b-help-template-audit-design.md`

---

## Pre-flight: Background facts

Facts already verified during plan preparation (you don't need to re-verify):

- `internal/usercommands/admin.devtool.go` imports `internal/devtools`. Therefore `devtools` importing `usercommands` creates a cycle. Hence the test lives in `usercommands`, not `devtools`.
- The registry is `var userCommands map[string]CommandAccess` in `internal/usercommands/usercommands.go`. `CommandAccess.AdminOnly` is the admin flag (bool).
- Existing `internal/usercommands/usercommands_test.go` has a `TestMain` that initializes loggers, so the new test file picks up that setup automatically.
- User help path: `_datafiles/world/dogmud/templates/help/<name>.template`
- Admin help path: `_datafiles/world/dogmud/templates/admincommands/help/command.<name>.template` (some are `.md` — e.g., `command.server.md`)
- Existing pattern for test: `internal/devtools/helpfile_completeness_test.go` (Spells/Recipes/Mutations/Skills variants). We mirror its style.

### Alias allowlist (pre-computed)

Commands that point to the same `UserCommand` func and share a help file:

| Command | Shares help with |
|---------|------------------|
| `companions` | `companion` |
| `stomp` | `kick` |
| `knee` | `kick` |
| `tailsweep` | `trip` |
| `rep` | `report` |

### Skip-list (pre-computed)

Internal/debug commands that don't need player-facing help:

- `default` — context-sensitive dispatcher (never typed directly)
- `noop` — does literally nothing
- `start` — account-creation flow only
- `zombieact` — internal zombie AI decision loop
- `print` — debug echo
- `printline` — debug echo

### Missing help templates (pre-computed)

After applying the alias allowlist, **9 user commands and 3 admin commands** have no help file at all:

**User commands missing help:** `cancel`, `forage`, `gearup`, `grapple`, `pet`, `save`, `submit`, `suicide`, `target`

**Admin commands missing help:** `questdebug`, `undeafen`, `unmute`

### Commands missing from keywords.yaml (pre-computed)

After applying the alias allowlist and skip-list, 44 commands currently aren't indexed in `_datafiles/world/dogmud/keywords.yaml`. The category plan is in Task 3.

---

## Task 1: Add `TestHelpFileCompleteness_Commands` (failing)

**Files:**
- Create: `internal/usercommands/helpfile_completeness_test.go`

- [ ] **Step 1: Create the test file**

Write this exact content to `internal/usercommands/helpfile_completeness_test.go`:

```go
package usercommands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// helpDataRoot walks up from the test working directory to find the
// _datafiles folder and returns the dogmud world root.
func helpDataRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "_datafiles")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Join(candidate, "world", "dogmud")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("cannot find _datafiles directory from %s", dir)
		}
		dir = parent
	}
}

// helpFileExistsAt checks whether a file exists at one of the given paths.
// Returns true if any of them exists.
func helpFileExistsAt(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// commandHelpAliases maps a command name to another command name whose
// help file it shares. Used when two commands map to the same handler
// function and a single help template covers both.
var commandHelpAliases = map[string]string{
	"companions": "companion",
	"stomp":      "kick",
	"knee":       "kick",
	"tailsweep":  "trip",
	"rep":        "report",
}

// commandHelpSkip lists internal/debug commands that don't need
// player-facing help files. These are not typed directly by players
// (e.g. context dispatchers, no-ops, account-creation flow commands).
var commandHelpSkip = map[string]bool{
	"default":   true, // context-sensitive dispatcher
	"noop":      true, // does nothing
	"start":     true, // account creation only
	"zombieact": true, // internal zombie AI
	"print":     true, // debug echo
	"printline": true, // debug echo
}

// TestHelpFileCompleteness_Commands ensures every registered user command
// has a matching help template. Regular commands live at
// help/<name>.template. Admin commands live at
// admincommands/help/command.<name>.template (or .md).
func TestHelpFileCompleteness_Commands(t *testing.T) {
	root := helpDataRoot(t)
	userHelpDir := filepath.Join(root, "templates", "help")
	adminHelpDir := filepath.Join(root, "templates", "admincommands", "help")

	var missing []string
	for name, info := range userCommands {
		if commandHelpSkip[name] {
			continue
		}
		target := name
		if alias, ok := commandHelpAliases[name]; ok {
			target = alias
		}

		if info.AdminOnly {
			tmpl := filepath.Join(adminHelpDir, "command."+target+".template")
			md := filepath.Join(adminHelpDir, "command."+target+".md")
			if !helpFileExistsAt(tmpl, md) {
				missing = append(missing, name+" (admin) — expected "+
					filepath.Join("admincommands", "help", "command."+target+".template"))
			}
		} else {
			tmpl := filepath.Join(userHelpDir, target+".template")
			md := filepath.Join(userHelpDir, target+".md")
			if !helpFileExistsAt(tmpl, md) {
				missing = append(missing, name+" (user) — expected "+
					filepath.Join("help", target+".template"))
			}
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("commands missing help files (%d):\n  %s\n\n"+
			"Add a template (a short stub is fine) or add the command to commandHelpSkip "+
			"if it's truly internal, or commandHelpAliases if it shares a help file with "+
			"another command.",
			len(missing), strings.Join(missing, "\n  "))
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails with the expected list**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./internal/usercommands/... -run TestHelpFileCompleteness_Commands -v
```

Expected: FAIL. The error message should list exactly 12 commands missing help (9 user + 3 admin):
- `cancel (user)`
- `forage (user)`
- `gearup (user)`
- `grapple (user)`
- `pet (user)`
- `questdebug (admin)`
- `save (user)`
- `submit (user)`
- `suicide (user)`
- `target (user)`
- `undeafen (admin)`
- `unmute (admin)`

If the list differs from this, stop and investigate — a new command may have been registered or a template may have been deleted since plan was written.

- [ ] **Step 3: Commit the failing test**

Committing a test in its failing state is intentional — it documents the gap for every developer on the team to see.

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
git add internal/usercommands/helpfile_completeness_test.go
git commit -m "$(cat <<'EOF'
test: add TestHelpFileCompleteness_Commands (failing)

Registered user commands should each have a matching help template.
Test lives in usercommands (not devtools) because usercommands
already imports devtools, so putting the test in devtools would
create an import cycle.

Test currently fails — 12 commands have no help file. Fixed in
follow-up commits.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create missing help template stubs

**Files:**
- Create: 9 files in `_datafiles/world/dogmud/templates/help/`
- Create: 3 files in `_datafiles/world/dogmud/templates/admincommands/help/`

Each stub follows the established one-page help format. Content is short and accurate. The engineer should not guess at mechanics — each stub below was written based on reading the command's source file.

- [ ] **Step 1: Stub — `cancel`**

Create `_datafiles/world/dogmud/templates/help/cancel.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">cancel</ansi>

The <ansi fg="command">cancel</ansi> command stops an in-progress action, such as a fold-cast spell
you are holding, a crafting activity, or a multi-round command.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">cancel</ansi>              Abort whatever you are currently doing.

If you are holding spell folds, cancelling loses the conviction you
already spent. If no cancellable action is active, the command is a
no-op.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help cast</ansi>, <ansi fg="command">help craft</ansi>
```

- [ ] **Step 2: Stub — `forage`**

Create `_datafiles/world/dogmud/templates/help/forage.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">forage</ansi>

The <ansi fg="command">forage</ansi> command searches the room for natural materials — herbs,
reagents, food, and crafting components. Success depends on your
Perception and the biome you are in.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">forage</ansi>              Search the current room for materials.

Cannot be used in combat. Each room has a limited foraging yield
that replenishes over time.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help foraging</ansi>, <ansi fg="command">help biome</ansi>
```

- [ ] **Step 3: Stub — `gearup`**

Create `_datafiles/world/dogmud/templates/help/gearup.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">gearup</ansi>

The <ansi fg="command">gearup</ansi> command quickly equips the best items you own for each
empty equipment slot. Useful after reviving, looting, or switching
kits.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">gearup</ansi>              Equip best available item in each empty slot.

Cannot be used in combat. Items already equipped are not replaced —
remove them first if you want to swap.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help equip</ansi>, <ansi fg="command">help remove</ansi>, <ansi fg="command">help inventory</ansi>
```

- [ ] **Step 4: Stub — `grapple`**

Create `_datafiles/world/dogmud/templates/help/grapple.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">grapple</ansi>

The <ansi fg="command">grapple</ansi> command attempts to lock your opponent in a grappling
hold. Grappled foes are restrained: they cannot flee, their attacks
and defenses are penalised, and your <ansi fg="command">knee</ansi> special becomes available.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">grapple <target></ansi>   Attempt to grapple a target in combat.

Success is an opposed roll of Strength vs. the target's Strength and
Dexterity. Grapple-immune creatures (e.g. elementals, oozes) cannot
be grappled.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help attack</ansi>, <ansi fg="command">help kick</ansi>, <ansi fg="command">help trip</ansi>
```

- [ ] **Step 5: Stub — `pet`**

Create `_datafiles/world/dogmud/templates/help/pet.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">pet</ansi>

The <ansi fg="command">pet</ansi> command interacts with tameable creatures in the room.
Petting a friendly creature may build rapport toward taming or
befriending it.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">pet <target></ansi>       Pet a creature in the room.

Not every creature is tameable. Hostile or wild creatures may react
poorly.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help pets</ansi>, <ansi fg="command">help companion</ansi>
```

- [ ] **Step 6: Stub — `save`**

Create `_datafiles/world/dogmud/templates/help/save.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">save</ansi>

The <ansi fg="command">save</ansi> command writes your character's current state to disk.
The server also autosaves periodically, so you rarely need to run
this manually.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">save</ansi>                Save your character now.

Safe to run at any time.
```

- [ ] **Step 7: Stub — `submit`**

Create `_datafiles/world/dogmud/templates/help/submit.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">submit</ansi>

The <ansi fg="command">submit</ansi> command yields in combat — you stop fighting and offer
surrender. Enemies that accept your submission will stop attacking.
Some hostile creatures will not accept surrender and keep attacking.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">submit</ansi>              Yield combat to your current opponent(s).

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help flee</ansi>, <ansi fg="command">help attack</ansi>
```

- [ ] **Step 8: Stub — `suicide`**

Create `_datafiles/world/dogmud/templates/help/suicide.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">suicide</ansi>

The <ansi fg="command">suicide</ansi> command ends your character's current life. Your
character enters the death sequence as if killed in combat, subject
to the usual death penalties and respawn rules.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">suicide</ansi>             End your character's life.

Use with caution — death carries penalties.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help death</ansi>
```

- [ ] **Step 9: Stub — `target`**

Create `_datafiles/world/dogmud/templates/help/target.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">target</ansi>

The <ansi fg="command">target</ansi> command switches your active combat target without
starting a new attack. You must already be in combat.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">target <name></ansi>      Switch to attacking the named opponent.

Tab-completion works on mob and player names in the room. Party
members can coordinate focus fire by targeting the same enemy.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help attack</ansi>, <ansi fg="command">help assist</ansi>
```

- [ ] **Step 10: Admin stub — `questdebug`**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.questdebug.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">questdebug</ansi> <ansi fg="red">(admin)</ansi>

The <ansi fg="command">questdebug</ansi> command dumps quest-engine state for debugging quest
flows — active triggers, player quest memory, and engine step state.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">questdebug</ansi>          Show current quest engine debug info.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help questtoken</ansi>
```

- [ ] **Step 11: Admin stub — `undeafen`**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.undeafen.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">undeafen</ansi> <ansi fg="red">(admin)</ansi>

The <ansi fg="command">undeafen</ansi> command reverses a prior <ansi fg="command">deafen</ansi>, restoring a player's
ability to hear room text, shouts, and other communication.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">undeafen <player></ansi>  Restore hearing for the named player.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help deafen</ansi>
```

- [ ] **Step 12: Admin stub — `unmute`**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.unmute.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">unmute</ansi> <ansi fg="red">(admin)</ansi>

The <ansi fg="command">unmute</ansi> command reverses a prior <ansi fg="command">mute</ansi>, restoring a player's
ability to speak, shout, and broadcast.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">unmute <player></ansi>    Restore speech for the named player.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mute</ansi>
```

- [ ] **Step 13: Run the test — it should now pass**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./internal/usercommands/... -run TestHelpFileCompleteness_Commands -v
```

Expected: PASS.

If it fails, read the error message — either (a) a stub was saved to the wrong path, (b) a command was missed, or (c) the alias/skip lists need updating. Fix and re-run.

- [ ] **Step 14: Commit**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
git add _datafiles/world/dogmud/templates/help/cancel.template \
        _datafiles/world/dogmud/templates/help/forage.template \
        _datafiles/world/dogmud/templates/help/gearup.template \
        _datafiles/world/dogmud/templates/help/grapple.template \
        _datafiles/world/dogmud/templates/help/pet.template \
        _datafiles/world/dogmud/templates/help/save.template \
        _datafiles/world/dogmud/templates/help/submit.template \
        _datafiles/world/dogmud/templates/help/suicide.template \
        _datafiles/world/dogmud/templates/help/target.template \
        _datafiles/world/dogmud/templates/admincommands/help/command.questdebug.template \
        _datafiles/world/dogmud/templates/admincommands/help/command.undeafen.template \
        _datafiles/world/dogmud/templates/admincommands/help/command.unmute.template
git commit -m "$(cat <<'EOF'
docs: stub missing help templates for 12 commands

Adds short help templates for 9 user commands (cancel, forage,
gearup, grapple, pet, save, submit, suicide, target) and 3 admin
commands (questdebug, undeafen, unmute). Closes the gaps found by
TestHelpFileCompleteness_Commands.

Stubs document intent, usage, and cross-references. Complex mechanics
are left to follow-up documentation work.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Index unindexed commands in `keywords.yaml`

**Files:**
- Modify: `_datafiles/world/dogmud/keywords.yaml`

`keywords.yaml` drives the categorised help menu that players browse with `help`. The file uses these top-level groupings:

- `help.command.<category>` — regular player commands, one entry per help topic
- `help.skill.all` — skill names
- `help.admin.all` — admin-only commands

44 commands currently registered in `userCommands` are not listed in this index. The assignments below group them under existing categories.

### Assignments

| Command | Target category |
|---------|----------------|
| `afk` | `configuration` |
| `assess` | `combat` |
| `blinding-flash` | `combat` |
| `blinding-spit` | `combat` |
| `bug` | `general` |
| `cancel` | `combat` |
| `companion` | `character` |
| `disenchant` | `crafting` |
| `dismiss` | `character` |
| `forage` | `crafting` |
| `gearup` | `items` |
| `go` | `general` |
| `grapple` | `combat` |
| `healing-gel` | `combat` |
| `hint` | `information` |
| `mutations` | `character` |
| `pacifism-aura` | `combat` |
| `pet` | `character` |
| `plant` | `crafting` |
| `pvp` | `combat` |
| `reply` | `communication` |
| `report` | `communication` |
| `salvage` | `crafting` |
| `save` | `configuration` |
| `setdesc` | `configuration` |
| `sethome` | `configuration` |
| `shadow` | `combat` |
| `sneak` | `combat` |
| `sonic-shout` | `combat` |
| `sort` | `items` |
| `steal` | `combat` |
| `storage` | `items` |
| `submit` | `combat` |
| `suggest` | `general` |
| `suicide` | `character` |
| `target` | `combat` |
| `toxic-bite` | `combat` |
| `questdebug` | `admin.all` |
| `undeafen` | `admin.all` |
| `unmute` | `admin.all` |

The following are already covered by alias indirection (their parent command is already indexed) and do **not** need their own entry:

- `companions` → `companion` (once `companion` is added)
- `stomp`, `knee` → already covered by indexed `kick`
- `tailsweep` → already covered by indexed `trip`
- `rep` → covered by `report`

- [ ] **Step 1: Read the current `keywords.yaml`**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
cat _datafiles/world/dogmud/keywords.yaml
```

Familiarise yourself with indentation (4 spaces per level, `-` lists). Categories preserve insertion order — insert new entries in roughly alphabetical order within each category to match the existing style.

- [ ] **Step 2: Add entries under `help.command.configuration`**

Open `_datafiles/world/dogmud/keywords.yaml`. Find the section:

```yaml
    configuration:
      - alias
      - macros
      - set
      - password
```

Replace it with:

```yaml
    configuration:
      - afk
      - alias
      - macros
      - password
      - save
      - set
      - setdesc
      - sethome
```

- [ ] **Step 3: Add entries under `help.command.character`**

Find the section:

```yaml
    character:

      - conditions
      - cooldowns

      - inventory
      - title
      - keyring
      - skills
      - spells
      - status
      - killstats
      - encumbrance
      - death
      - character
      - pets
      - bury
```

Replace it with:

```yaml
    character:
      - bury
      - character
      - companion
      - conditions
      - cooldowns
      - death
      - dismiss
      - encumbrance
      - inventory
      - keyring
      - killstats
      - mutations
      - pet
      - pets
      - skills
      - spells
      - status
      - suicide
      - title
```

- [ ] **Step 4: Add entries under `help.command.communication`**

Find the section:

```yaml
    communication:
      - emote
      - say
      - shout
      - broadcast
      - talk
      - whisper
      - inbox
```

Replace it with:

```yaml
    communication:
      - broadcast
      - emote
      - inbox
      - reply
      - report
      - say
      - shout
      - talk
      - whisper
```

- [ ] **Step 5: Add entries under `help.command.combat`**

Find the section:

```yaml
    combat:
      - assist
      - attack
      - bash
      - break
      - cast
      - consider
      - cooldowns
      - defense
      - flee
      - kick
      - shoot
      - stamina
      - stand
      - rally
      - taunt
      - throw
      - trip
      - warcry
```

Replace it with:

```yaml
    combat:
      - assess
      - assist
      - attack
      - bash
      - blinding-flash
      - blinding-spit
      - break
      - cancel
      - cast
      - consider
      - cooldowns
      - defense
      - flee
      - grapple
      - healing-gel
      - kick
      - pacifism-aura
      - pvp
      - rally
      - shadow
      - shoot
      - sneak
      - sonic-shout
      - stamina
      - stand
      - steal
      - submit
      - target
      - taunt
      - throw
      - toxic-bite
      - trip
      - warcry
```

- [ ] **Step 6: Add entries under `help.command.crafting`**

Find the section:

```yaml
    crafting:
      - craft
```

Replace it with:

```yaml
    crafting:
      - craft
      - disenchant
      - forage
      - plant
      - salvage
```

- [ ] **Step 7: Add entries under `help.command.information`**

Find the section:

```yaml
    information:
      - biome
      - exits
      - help
      - look
      - online
      - species
      - who
      - history
```

Replace it with:

```yaml
    information:
      - biome
      - exits
      - help
      - hint
      - history
      - look
      - online
      - species
      - who
```

- [ ] **Step 8: Add entries under `help.command.items`**

Find the section:

```yaml
    items:
      - drop
      - drink
      - eat
      - equip
      - get
      - give
      - remove
      - show
      - stash
      - trash
      - use
      - read
      - put
```

Replace it with:

```yaml
    items:
      - drink
      - drop
      - eat
      - equip
      - gearup
      - get
      - give
      - put
      - read
      - remove
      - show
      - sort
      - stash
      - storage
      - trash
      - use
```

- [ ] **Step 9: Add entries under `help.command.general`**

Find the section:

```yaml
    general:
      - instances
      - arena
      - oasis
      - motd
      - online
      - quit
      - scan
```

Replace it with:

```yaml
    general:
      - arena
      - bug
      - go
      - instances
      - motd
      - oasis
      - online
      - quit
      - scan
      - suggest
```

- [ ] **Step 10: Add entries under `help.admin.all`**

Find the section:

```yaml
  admin:
    all:
      - ai-flag
      - ai-list
      - badcommands
      - buff
      - build
      - combatstats
      - command
      - deafen
      - devtool
      - item
      - locate
      - mob
      - modify
      - mudmail
      - mute
      - paz
      - prepare
      - questtoken
      - redescribe
      - reload
      - rename
      - room
      - server
      - setmotd
      - skillset
      - spawn
      - spell
      - syslogs
      - teleport
      - zap
      - zone
```

Replace it with:

```yaml
  admin:
    all:
      - ai-flag
      - ai-list
      - badcommands
      - buff
      - build
      - combatstats
      - command
      - deafen
      - devtool
      - item
      - locate
      - mob
      - modify
      - mudmail
      - mute
      - paz
      - prepare
      - questdebug
      - questtoken
      - redescribe
      - reload
      - rename
      - room
      - server
      - setmotd
      - skillset
      - spawn
      - spell
      - syslogs
      - teleport
      - undeafen
      - unmute
      - zap
      - zone
```

- [ ] **Step 11: Verify YAML parses cleanly**

Keywords.yaml is loaded on every server boot; a malformed file prevents startup.

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
python3 -c "import re; t = open('_datafiles/world/dogmud/keywords.yaml').read(); print('line count:', t.count(chr(10)))"
```

Expected: `line count: ~330` (up from 291). An easy sanity check only — doesn't validate YAML syntax. For real validation:

```bash
go build ./... && go test ./internal/keywords/... 2>&1 | head -30
```

If `internal/keywords` has no tests, run the commands test — it will load the data implicitly when fetched via help, but does not require YAML to parse. Proceed to the next step if build is clean.

- [ ] **Step 12: Run the full test suite**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./...
```

Expected: all pass, including `TestHelpFileCompleteness_Commands`.

- [ ] **Step 13: Commit**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
git add _datafiles/world/dogmud/keywords.yaml
git commit -m "$(cat <<'EOF'
docs: index 44 missing commands in keywords.yaml help menu

After the command-unification work, several dozen player commands
had help files but weren't listed under the categorised `help` menu.
Players could still run `help <cmd>` directly but couldn't discover
them by browsing.

Indexed under: configuration, character, communication, combat,
crafting, information, items, general, and admin.all. Alias commands
(companions, stomp, knee, tailsweep, rep) rely on their parent
command's entry and are not duplicated.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Final verification

**No files modified** — this task only verifies the three prior tasks land cleanly.

- [ ] **Step 1: Run the help completeness test alone**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./internal/usercommands/... -run TestHelpFileCompleteness_Commands -v
```

Expected: PASS.

- [ ] **Step 2: Full build**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
```

Expected: clean build.

- [ ] **Step 3: Full test suite**

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./...
```

Expected: all pass.

- [ ] **Step 4: Spot-check the help index visually**

Count the number of commands in each category after the change:

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
grep -c "^      - " _datafiles/world/dogmud/keywords.yaml
```

Expected: a count that includes all previously indexed entries plus the 44 new ones. Rough sanity check: it should be significantly higher than before (pre-change was ~135 list items in the whole file including non-help sections).

- [ ] **Step 5: Optional manual smoke test**

If convenient, start the server and verify in-game:

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go run .
```

In a MUD client:
1. Type `help` and verify the categories render without error.
2. Type `help grapple` (a newly-added stub) — should display the template cleanly.
3. Type `help attack` (an existing, untouched template) — verify nothing regressed.
4. Ctrl+C the server.

This step is optional but recommended.

- [ ] **Step 6: Report completion**

No commit. If everything passes, Stage 1.4b is done. Report:

- Test added: 1 (`TestHelpFileCompleteness_Commands`)
- Templates created: 12 (9 user + 3 admin)
- Keywords indexed: 44 commands across 9 categories
- Build clean, all tests green
- Any TODOs flagged during implementation (e.g., ambiguous commands, stale aliases)

---

## Important constraints

- **Zero behavior change.** The test is additive. Template stubs add new files without modifying any existing one. Keyword entries affect help rendering only.
- **No existing template files deleted.** The spec explicitly scopes this out. If an orphaned template is spotted in passing, leave it and raise it as a follow-up.
- **Stub content quality matters.** Each stub should be accurate. If a command's mechanics are unclear to the implementer, prefer adding to `commandHelpSkip` with a TODO note over writing a misleading stub. All pre-approved stubs in Task 2 were drafted from reading the command's source; they should be safe to commit as-is.
- **Admin help path differs.** Admin help templates live under `admincommands/help/command.<name>.template`, not `help/<name>.template`. The test and the stubs in Task 2 handle this correctly.
- **Alphabetical ordering within categories** is preserved in Task 3's target YAML blocks. If you're surprised by reordering of existing entries (e.g. `character` moved to the top), that's intentional cleanup of the previous non-alphabetical layout.

## What success looks like

After this plan:
- `go test ./internal/usercommands/... -run TestHelpFileCompleteness_Commands` passes
- All 44 previously unindexed commands now appear under appropriate categories in `keywords.yaml`
- 12 previously un-stubbed commands have help templates
- Players browsing `help` in-game see a richer, more complete categorised menu
- No existing content was modified or deleted beyond re-sorting list order
