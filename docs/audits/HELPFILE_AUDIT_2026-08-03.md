# Helpfile Audit — 2026-08-03

Six-agent sweep: coverage matrix of every registered player + admin command
vs the help trees, plus a staleness read of all 458 player helpfiles
(mechanics files in full, flavor files by targeted grep) and all 49 admin
helpfiles, each claim spot-verified against current code. This doc is the
consolidated result, batched for fixing. Evidence file:line lives with each
finding below; ✅ = already fixed.

## How help resolution actually works (load-bearing for the index fixes)

`help <topic>` never consults the command registry: it runs the topic
through `keywords.TryHelpAlias` (built from `help-aliases:` in
`_datafiles/world/dogmud/keywords.yaml` + module data-overlays), then does a
direct template lookup under `templates/help/`, falling back to spell-name
matching. The bare `help` MENU is populated only from keywords.yaml's
`help:` topic lists. Consequences: any template is reachable by exact name
even if unindexed; the menu can list topics with no file ("Missing"); and
the menu and reality drift independently. Modules register commands via
`AddUserCommand` + their own keywords overlays — the static registry in
usercommands.go is NOT the full command surface.

---

## ✅ Batch 1 — DELETE: orphans and fossils (DONE 08-03; +minor-shield.template, whose spell no longer exists — 10 total)

Files documenting things that do not exist. Delete outright; where a live
twin exists, the twin already covers the topic.

| File | Why |
|---|---|
| `attack.md` | Pre-rewrite duplicate of attack.template; fabricated level-based crit formula (`attackerLevel` appears nowhere in code) |
| `brawling.template` | No brawling skill; `tackle`/`disarm`/`recover` commands don't exist |
| `trading.template` | Titled "trading", describes brawling, level-gated, dead `Smarts` stat; real docs live at trade.template |
| `skulduggery.template` | Single-L typo twin of skullduggery.template; documents nonexistent `bump`/`backstab` |
| `scribe.template` | `scribe` command never shipped; level-gated design |
| `peep.template` | Peep skill removed (`roomdetails.go:238` comment); no command |
| `portal.template` | `portal` is mob-only (`divergences.go:187`); documents a player skill that doesn't exist |
| `protection.template` | Entire skill fabricated: `aid` is mob-only, `rank`/`pray` exist nowhere |
| `dual-wield.template` | Describes removed leveled skill (random-weapon tiers); real mechanic is a continuous Weapon-Combat-scaled penalty already correctly documented in combat.template. Replace with a help-alias dual-wield → combat |

## ✅ Batch 2 — REWRITE: player helpfiles stating wrong facts (DONE 08-03)

| File | Wrong → right |
|---|---|
| `remove.template` | **Describes the reverse action** ("wields or wears an item from your backpack") — remove un-equips to backpack (remove.go:62) |
| `heal.template` | "Minor Heal", cost 3 → spell is **Mend Flesh**, cost 25 (spells/heal.yaml) |
| `iron-will.template` | Cost 6 → 45 (iron-will.yaml:8) |
| `minor-shield.template` | Unified "armor rating on the status sheet" + raw formula → sums into physical mitigation (one of three channels); status shows a qualitative word; never affects hit chance |
| `taunt.template` | Conviction-zero "downed, unable to fight" state → conviction just degrades effectiveness smoothly (ResourceMultiplier); no such state |
| `special.template` | Shared cooldown lists only bash/kick/trip/grapple/cast → also taunt, warcry, rally, throttle, hamstring, reload, gore, maul, rake, pounce, mutation actives |
| `set-prompt.template` | Phantom `{tp}`/`{sp}` tokens from the removed training-point system (not in ProcessPromptString) |
| `alchemy.template` | Recipe table: wrong names (Healing Poultice→Healing Salve, Stamina Draught→Stamina Tonic), one nonexistent recipe (Greater Healing Poultice), wrong minimums (Berserker 38→18), wrong/missing ingredients, "small-vial" isn't an item (Glass Vial, tag `bottle`) |
| `blacksmithing.template` | Steel Ingot skill 10→4 + missing pine-pitch; Chain Links 15→7 |
| `essence-of-growth.template` | Missing wild-hare-meat ingredient |
| `fire-resistance-draught.template` | "small vial"→bottle; missing oak-bark |
| `lake-iron-hook-spear.template` | Missing sinew |
| `leather-bandolier.template` | Capacity 6 → 12 |
| `blood-boil` / `conviction-spike` / `conviction-barrage` | "armor and toughness resist" → physical-defense spells are DODGED (opposed by Dexterity, spell_resolution.go:974); armor only reduces damage after a hit. Check the same wording in kinetic-shove/pyretic-surge/hemorrhagic-burst/veil-rend ("constitution" is also not a stat) |
| `rhetoric.template` | Exposes raw 1.0x–3.0x multiplier numbers (style) and they may not match the current formula — verify then describe qualitatively |

## ✅ Batch 3 — INDEX repair (DONE 08-03; in-game index render verified)

- `help trade`/`barter`/`haggle` alias to nonexistent `bartering` file while
  trade.template sits unreachable → repoint alias at trade (and see
  bartering below).
- Player `rename` is filed in the admin help bucket (invisible to players);
  admin `renameitem` is indexed nowhere.
- Ghost topics with no file: `deposit`, `store`, `unstore`, `withdraw`
  (alias to bank/storage), `submit` (automatic mechanic, not a command —
  drop or alias to surrender).
- ~~Module topics fileless~~ CORRECTION: all four module templates exist
  in module `files/datafiles/templates/help/` mounts — the coverage agent
  missed module template overlays. auction.template had two errors of its
  own (standalone `bid`, nonexistent "Auction skill"), fixed.
- 9 of 16 skills absent from the menu's Skills section (weapon-combat,
  unarmed-combat, ranged-combat, spellcasting, rhetoric, skullduggery,
  bartering, salvage, manifestation) — 8 have templates, index them.
- ~20 complete templates indexed nowhere: achievements, guild, bounty,
  petition, sleep, venom-coat, cocoon, mail, loot, chat, newbie, channels,
  deletecharacter, fine, payfine, rep, gc, stomp, knee — index the
  player-facing ones.
- `tailsweep` (registered trip variant) → help-alias to trip.
- `companions` → help-alias to companion.

## ✅ Batch 4 — WRITE: missing content (DONE 08-03)

- `bartering.template` — the only skill with no file at all.
- Chunk-4 sections (the original queue item): eat/drink grapple-block note
  (+ the new focused-work refusals), cast "combat and concentration"
  paragraph, per-spell disruption line, attack "position in grapples"
  subsection.
- `companion.template` — add the conviction-reservation cost (assess.template
  already documents it; the main companion file never mentions it).
- `drink.template` — modernize: bandolier-first search, drinkable
  preference, aging/potency, busy/grapple refusals.
- Minimal `leaderboard`/`auction`/`time`/`follow` templates (Batch 3 dep).

## ✅ Batch 5 — ADMIN fixes (DONE 08-03)

- ✅ `redescribe` template-path bug (loaded nonexistent `command.rename`) —
  fixed `006a01fd1`.
- `command.server.md` → rewrite as .template (only Markdown file in the
  tree) and add missing subcommands: `set day`/`set night`, `ansi-strip`,
  `ansi-mono`, `ansi-normal`, `info`.
- `command.room.template` — add `room set biome`, `room copy idlemessages`,
  `room copy mutator`.
- `command.spawn.template` — document `spawn container [name]`.
- `command.paz.template` — "pax" typo; "hp/mp" → Health/Conviction (same
  wording fix in `command.zap.template`).
- `command.mute/deafen.template` — typos; `command.goal.template` — `rm`
  alias; `command.prepare.template` — any argument works, not just "all".

## Clean bill of health (verified, no action)

Admin coverage is 1:1 (49/49, no orphans); the 2026-06/07 simulation
systems' admin help is excellent. High-traffic player mechanics files all
verified current: armor, combat, defense, craft, salvage, spells, status,
stats, stamina, sleep, flee, grapple, kick/knee/stomp/trip, track, sneak,
steal, warcry, quests, mutations, party, pvp, ranged-combat, toxicity,
storage, assess, chrysalis-* (9), conjure-* (5), raise-* and the psionic
spell set. Zero references to the six retired mutation actives anywhere.
No level/XP language outside the Batch 1/2 files.

## Idea (not scheduled)

The recipe-table errors (alchemy, blacksmithing, 3 per-recipe files) are
hand-copied YAML data that drifted. Generating the ingredient/minimum lines
from the recipe YAMLs at render time would make this class of error
impossible — same pattern as the per-spell help template.
