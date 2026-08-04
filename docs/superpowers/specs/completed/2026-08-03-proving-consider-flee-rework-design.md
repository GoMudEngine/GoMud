# The Proving: consider lesson + prominent flee beat (2026-08-03)

## Problem (Malia's feedback, 2026-08-02)

Three complaints that turn out to be one intertwined defect in room 6262
"The Proving" (newcomer antechamber, quest 28):

1. "The dummy has way too much life." The Straw Effigy (mob 9614) has
   ~300k HP **by design** — the lesson ends in `flee`, never a kill, and
   it was raised 1000→100000 on 2026-07-19 precisely because a slow
   player killed it early and got stuck with no target.
2. `flee` "isn't taught / easy to miss." It IS taught — but as the third
   of three delayed `npc_say` lines after `trip`, delivered mid-combat
   under round spam. Miss that one line and you grind an immortal dummy
   forever, which is exactly what complaint 1 looks like from the
   player's chair.
3. `consider` is never taught anywhere in the tutorial, though the
   command exists and gives a graded gut-feel power read.

## Decision (user-approved 2026-08-03)

Keep the effigy's HP untouched (lowering it re-creates the July
stuck-player bug). Fix the *experience*: teach `consider`, make the
effigy's unbreakability an explicit plot point, and make `flee` a beat
nobody can miss.

## Design

### 1. New lesson step: `consider effigy`

Insert step `considered` between `checked_cooldowns` and `attacked` in
quest 28:

- The `cooldowns` response line now ends by prompting
  `consider effigy` instead of `attack effigy`.
- New `command_issued` trigger (`command: consider`, `noun: effigy`,
  has `28-checked_cooldowns`, missing `28-considered`) grants
  `28-considered`. Dewey confirms the read is true — the effigy is
  woven too tight to ever break — tells the player to remember that
  feeling, and prompts `attack effigy`.
- The `attack` trigger's gate changes from `28-checked_cooldowns` to
  `28-considered`.
- Steps list gains the `considered` entry; the `checked_cooldowns` and
  `considered` hints updated to match.

The effigy's power score is durability-dominated, so `consider` reports
"You are severely outmatched" — which the lesson leans into: that read
is *why* flee exists.

### 2. Flee becomes a called-back, re-nagged beat

- The post-`trip` three-line burst is reworked: the final line calls
  back to the consider read ("your gut told you — this one cannot be
  finished") before prompting `flee`.
- **Re-nag**: Dewey's per-mob behavior tree gains a `mob_idle` branch —
  while any player in his room holds `28-tripped` but not `28-fled`,
  a cooldown-decorated `say` repeats the flee prompt every few rounds.
  Nobody can grind the effigy for long without being told again. The
  branch sits above `try_goal_planner` in the selector.

### 3. New engine condition: `player_in_room_has_quest`

The re-nag needs "player in room HAS `28-tripped`" — only the
`player_in_room_missing_quest` variant exists (mob_idle has no
triggering player, so per-player `player_has_quest` cannot gate it).
Add the mirror condition in `internal/behaviortree/conditions_player.go`,
register it, document it in the package `context.md`, and cover it in
the conditions tests. ANDing has/missing in-room conditions could match
different players in a shared room; the antechamber is a solo ephemeral
instance, so this is safe here — the doc note in context.md says so.

## Out of scope

- Effigy HP/stats: unchanged.
- Pothole Coulee dummies (9109/9163): already quick ~50 HP kills;
  untouched.
- Teaching `consider` again post-tutorial: not now.

## Verification

- `go test ./internal/behaviortree` (CI fallback if Defender quarantines
  the fresh test binary).
- Boot test in an isolated worktree (quest/mob YAML loads clean).
- Mandatory in-game adversarial playtest of the route-1 Proving flow,
  including the "player ignores the flee line and keeps swinging" path,
  before merge (merge to master = shipped).
