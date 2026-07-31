# Intentional Command Divergences

Documents all intentional differences between user and mob command
systems, with rationale for each.

The authoritative source of truth is `internal/actions/divergences.go`
(`userOnlyCommands` and `mobOnlyCommands` maps). This file summarises
the design intent behind each category; the code is the single source
of allowlist membership.

---

## Admin Commands (User-Only) — 34 commands

These commands exist for game-management purposes and have no game-world
meaning for mobs. They are gated by the admin permission system and are
never exposed to players or mob AI.

| Command | Notes |
|---------|-------|
| ai-flag | Toggle AI debug flags on mobs |
| ai-list | List mob AI state |
| badcommands | Report unrecognised command attempts |
| buff | Apply/remove buffs on targets |
| build | Zone builder tool |
| command | Inspect/reload command handlers |
| combatstats | Dump per-session combat statistics |
| deafen | Silence a player connection |
| devtool | Developer diagnostics |
| item | Admin item manipulation |
| locate | Find players or mobs by name |
| mob | Admin mob manipulation |
| modify | Edit live data objects |
| mudmail | Send in-game mail to players |
| mute | Mute a player |
| paz | Peace-and-zone admin command |
| prepare | Pre-stage content for deployment |
| questdebug | Inspect quest state for a player |
| questtoken | Grant/revoke quest tokens |
| redescribe | Rewrite a room description live |
| reload | Reload data files without restart |
| rename | Rename a player or object |
| room | Admin room manipulation |
| server | Server control (shutdown, reload) |
| setmotd | Update the message of the day |
| skillset | Force-set a skill rank |
| spawn | Spawn mobs or items |
| spell | Admin spell manipulation |
| syslogs | View server system logs |
| teleport | Teleport a player or mob |
| undeafen | Un-silence a player connection |
| unmute | Un-mute a player |
| zap | Instant-kill a target (admin) |
| zone | Zone administration tool |

---

## UI / Display Commands (User-Only) — 41 commands

These commands affect only the player's terminal view or account state.
They have no game-world effect and make no sense for mob AI to execute.

| Command | Notes |
|---------|-------|
| afk | Toggle away-from-keyboard flag |
| alias | Manage command aliases |
| bank | Display bank balance/history |
| biome | Show current biome information |
| bug | Submit a bug report |
| cancel | Cancel a pending action |
| character | Display character sheet |
| conditions | List active conditions/debuffs |
| consider | Gauge relative difficulty of a target |
| cooldowns | Show ability cooldown timers |
| default | Reset settings to defaults |
| help | Display help topics |
| hint | Toggle/display gameplay hints |
| history | Show command history |
| inbox | Read in-game mail |
| inventory | List carried items |
| keyring | Display held keys |
| killstats | Show kill/death statistics |
| macros | Manage client macros |
| map | Render the area minimap |
| motd | Show message of the day |
| mutations | List active mutations |
| online | Show online player list |
| password | Change account password |
| print | Print a raw string (debug) |
| printline | Print a line (debug) |
| pvp | Toggle PvP flag |
| quests | List active/completed quests |
| quit | Disconnect from the server |
| read | Read an item's text content |
| rep | Show reputation standings |
| report | Report a player for conduct |
| save | Force-save character to disk |
| set | Set client/character options |
| setdesc | Set character description |
| skills | List skill ranks |
| spells | List known spells |
| status | Full stat sheet display |
| suggest | Submit a suggestion |
| title | Set character title |
| who | Show players in current zone |

---

## Player Mechanics (User-Only)

Commands that implement systems that do not exist for mob AI.

| Command | Rationale |
|---------|-----------|
| assist | Party system — mobs do not form parties |
| party | Party system — mobs do not form parties |
| share | Party XP/loot share — mobs do not form parties |
| reply | Private messaging — mobs use `say`/`sayto` |
| whisper | Private messaging — mobs use `say`/`sayto` |
| target | Targeting UI — mobs use AI targeting logic |
| start | Character creation entry point |
| zombieact | Zombie player state handling |

---

## Mob AI Commands (Mob-Only) — 22 commands

Commands implemented only for mob AI scripts and behaviour routines.
Players do not have access to these; they are driven by the mob AI loop,
not by direct input.

| Command | Notes |
|---------|-------|
| aid | Assist a nearby ally in combat |
| befriend | Attempt to befriend a target |
| callforhelp | Broadcast a call-for-help to nearby mobs |
| charge | Rush at a target, initiating combat |
| consume | Eat/drink an item in inventory |
| converse | Initiate scripted NPC conversation |
| despawn | Remove the mob from the world |
| givequest | Grant a quest token to a player |
| hamstring | Cripple target movement speed |
| lookforaid | Scan for allies to request help from |
| lookfortrouble | Scan for hostile targets to engage |
| pathto | Navigate toward a destination room |
| portal | Create a temporary portal |
| replyto | Respond to a specific player |
| sayto | Direct speech at a specific target |
| saytoonly | Speak only to a specific target (no room echo) |
| wander | Roam randomly through connected rooms |

### Mob-Only: Pending Consolidation

These mob commands duplicate or overlap existing player systems and are
candidates for removal or unification in future stages.

| Command | Category | Plan |
|---------|----------|------|
| howl | mob-ai | Renamed taunt — unify with shared taunt action |
| backstab | mob-ai | Redundant with surprise strike system — remove |
| roar | mob-ai | Future player shout/intimidation system |
| throw | mob-ai | Future player ranged ability |
| alchemy | mob-alchemy | Consolidate with shared craft system |

---

## Pending Consolidation

These are the highest-priority targets for future unification work:

- **howl** → unify with `taunt` (same conviction damage mechanic, mob
  renames it for flavour but the action is identical)
- **backstab** → remove (redundant with surprise strike; the distinction
  is not meaningful for mob AI)
- **roar** → add player intimidation/shout system, then share the action
- **throw** → add player ranged ability, then share the action
- **alchemy** → consolidate with shared `craft` system introduced in
  Stage 5 of the command unification project

---

## Behavioral Asymmetries (By Design)

Some commands exist on both sides but are intentionally implemented
differently for players vs mobs. These are NOT bugs.

### Flee
- **Players**: enter a flee-attempt state that is resolved by the combat
  loop on the next tick. Failure is possible (opponent too strong, no
  exits available). The delay creates tension.
- **Mobs**: flee is resolved immediately. Mob AI needs instant escape to
  maintain flow control; a delayed resolution would break state machines.

### Sneak Initiation
- **Players**: gated by Skullduggery skill rank >= 1; failure triggers a
  cooldown preventing re-attempt for several rounds.
- **Mobs**: skill gate and failure cooldown are skipped entirely. Mob
  sneak decisions come from AI scripts, not player-style skill checks.

### Spell Initiation
- **Players**: roll `CalcInitiationChance` on cast attempt; failure
  triggers a cooldown. Rewards investment in Willpower and spellcasting.
- **Mobs**: initiation roll and cooldown are skipped. Mob casting
  decisions come from AI scripts; the randomness is in target selection
  and AI frequency, not initiation failure.

### Progression Events
- **Players**: fire `events.SkillUsed` (carries `UserId`); the event
  system applies probabilistic advancement.
- **Mobs**: call `OnSkillUse`/`OnStatUse` directly because the
  `events.SkillUsed` struct carries only `UserId` — there is no
  `MobInstanceId` field. This is a structural limitation of the event
  system, not an intentional design choice; it should be unified if the
  event struct is extended.

---

## Aliases (User-Only)

These user-side command names route to shared actions with variant
detection. The underlying shared action works for both players and mobs;
only the alias name is user-only.

| Alias | Routes To | Condition |
|-------|-----------|-----------|
| stomp | kick | Target is prone |
| knee | kick | Actor is grappling and in control |
| tailsweep | trip | Actor has the Tail mutation |

Mob AI scripts invoke the underlying command (`kick`, `trip`) directly
and the variant is detected inside the shared action handler by checking
combat state. The aliases exist purely as convenience shortcuts for
human typists.
