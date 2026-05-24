# Mob Aliveness 2.10 — Deferred Parity Gaps for Review

Surfaced by the chunk 2.10 audit
(`docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`).
This doc holds every row classified as **Gap: defer** during the audit
walk. Each entry below lists a proposed verdict; user triage picks one
of:

- **accept-proposed-verdict** — carry through as proposed
- **change-verdict** — adjust per user note
- **drop-entirely** — remove without memory entry
- **fix-now-anyway** — pull back into chunk 2.10 (triggers a Stage D
  micro-task)

After triage, Stage F memory writes capture the per-verdict followups.

**Audit summary:**
- Player-side audit: 5 Gap: defer rows surfaced (this doc)
- Mob-side audit: 0 Gap: defer rows surfaced

---

## Triage results (2026-05-23)

User triaged all 5 entries:
- `surprise_attack`: change-verdict → memory-entry-only (audit row was wrong; mobs already have parity; capture refactor opportunity)
- `picklock`: change-verdict → wontfix (intentional design misalignment)
- `lock`: change-verdict → patch-as-followup-chunk (bundle with `unlock` as forager-chest workflow)
- `unlock`: change-verdict → patch-as-followup-chunk (bundle with `lock`)
- `throw`: accept (memory-entry-only) with dependency on future ranged weapon system

Stage F memory writes follow.

---

## Player-side deferred gaps

### Direction: mob-side missing (player has the verb, mob doesn't)

#### surprise_attack

- **Direction:** mob-side missing
- **Surface today:** `internal/usercommands/surprise_attack.go` (235 LoC)
  is a helper function (`executeSurpriseAttack`) called from the attack
  path when a player initiates combat from hidden state. It resolves
  one burst swing per equipped weapon — primary, offhand, and each extra
  arm — applying escalating hit penalties per arm, a
  dexterity+skullduggery surprise multiplier, and half-mitigation bypass
  (crit-like). After all swings, skill progression fires on Skullduggery.
  The entire function is wired directly into `usercommands/attack.go`;
  no standalone command registers it.
- **Why deferred:** 235 LoC in the helper, plus the integration point in
  `attack.go` that gates it on hidden state. A mob equivalent would need:
  a new btree action or `attack.go` mob-path branch that checks whether
  the mob is hidden before combat begins, the same multi-weapon sweep
  (mobs have extra arms too), the same surprise multiplier logic (needs
  `MobActor` stat access), and a decision on whether mobs should get the
  half-mitigation bypass or a tuned alternative. Well above the 30-LoC
  quick-patch threshold; needs an explicit design choice on how aggressive
  this makes stealth mobs.
- **Sketch of fix:** Extract the existing helper into
  `internal/actions/surprise_attack.go` (following the mutation\_\* lift
  pattern). Add a mob-side caller that fires when `lookfortrouble` or a
  btree `attack` node detects the mob is hidden. Wire a btree action
  `surprise_attack` that wraps the action and returns Success/Failure so
  authors can opt specific mobs (rogues, assassin-type mobs) into the
  burst-opener behavior.
- **Proposed verdict:** patch-as-followup-chunk
- **Estimated size:** M (actions lift ~150 LoC, mob wrapper + btree
  action ~60 LoC, integration + tests ~80 LoC)
- **Final verdict (user triage 2026-05-23):** **change-verdict to memory-entry-only.** User confirmed mobs ALREADY have surprise-attack behavior (per-weapon attack from hidden state when special-move cooldown is available, identical to the player path) — see `internal/mobcommands/attack.go:64` and the Awareness_Cascades hook. This is not a parity gap; the A1 audit row was factually wrong and is being corrected separately. What remains is a refactor opportunity: unify the mechanism (extract a shared `actions.SurpriseAttack` helper) so the two paths can't drift. User strongly prefers the unification, but it's not blocking.

#### picklock

- **Direction:** mob-side missing
- **Surface today:** `internal/usercommands/picklock.go` (363 LoC).
  The player flow is a multi-turn interactive prompt: it renders a
  pin-tumbler ASCII minigame, consumes lockpick items on failures,
  checks the player's keyring for a cached sequence, applies any
  `statmods.Picklock` presolve bonus, and unlocks the exit/container
  on success. The 363 LoC include both the command function and the
  `GetLockRender` / `sequenceMatches` table-rendering helpers.
- **Why deferred:** The interactive prompt loop is fundamentally
  player-UI-only (it uses `user.StartPrompt` / `user.ClearPrompt` which
  have no mob equivalent). A mob-side picklock would be a simplified,
  non-interactive variant: one opposed skill roll per attempt, consuming
  a lockpick item on failure. That stripped-down path is itself ~40-60
  LoC, but it needs a design decision: should mobs carry lockpicks as
  inventory items (with all the implications for loot drops and btree
  authoring), or should the mob have an innate "lockpick" skill that
  bypasses the item requirement? Also needs a btree action with args for
  which exit to target and a retry-count limit so mobs don't loop forever.
- **Sketch of fix:** New `internal/actions/picklock.go` with a
  `MobPicklock(actor Actor, exitName string) PicklockResult` function
  that does a single Skullduggery roll, consumes a lockpick if carried
  (or fails if none), and mutates exit lock state on success. New btree
  action `try_picklock` with an `exit` arg. Authors add it to thief /
  pursuer archetypes. Player command collapses the pre-check path to call
  the action, keeping the interactive prompt for the minigame portion.
- **Proposed verdict:** patch-as-followup-chunk
- **Estimated size:** M (action ~50 LoC, btree action ~40 LoC, design
  doc for lockpick-item-vs-innate decision ~30 LoC)
- **Final verdict (user triage 2026-05-23):** **change-verdict to wontfix.** Intentional misalignment by design. The player picklock command is an interactive minigame with up/down gates — that surface only makes sense for a human player. A mob equivalent would by necessity be a single-roll simplification, which isn't worth maintaining as a separate "mob picklock" verb when the design intent is that picklock IS the player-only minigame.

#### lock

- **Direction:** mob-side missing
- **Surface today:** `internal/usercommands/lock.go` (156 LoC). The
  player looks up a container or exit by name in the current room, checks
  whether the character has the matching key on their keyring or in their
  backpack (using `HasKey` + `FindKeyInBackpack`), and calls
  `SetLocked()` / `room.SetExitLock(exitName, true)`. If the key comes
  from the backpack, it is consumed and added to the keyring for future
  use. Sound and room broadcast follow.
- **Why deferred:** The player implementation is moderately compact (156
  LoC) but a mob equivalent raises a design question: under what world
  conditions should a mob lock a door behind itself? Locking is a
  defensive/patrol behavior with no obvious btree trigger today (there is
  no "mob just passed through an exit" event). Implementing it correctly
  requires a btree action with room-exit targeting, a mob permission model
  for which exits a mob is authorized to lock, and a design decision on
  whether the mob must carry a key item or can lock "innately." Without a
  concrete use case in current world content, the risk of authoring
  complexity outweighs the benefit.
- **Sketch of fix:** New btree action `lock_exit` (or `lock_container`)
  that mirrors the player flow: mob must carry a matching key item, looks
  up the exit/container in the current room, calls `SetLocked()`. Pair
  with an `on_exit` btree event so a mob can lock the door after passing
  through it (the patrol-guard pattern). Authored only for specific
  guardian mobs where the lock-behind behavior is intentional.
- **Proposed verdict:** memory-entry-only
- **Estimated size:** S-M (action ~60 LoC, btree action ~30 LoC, but
  needs the `on_exit` event hook which may not exist yet)
- **Final verdict (user triage 2026-05-23):** **change-verdict to patch-as-followup-chunk.** Concrete world use case: foragers keep their extra goods in locked chests. A future chunk would extend forager behavior to unlock the chest they own with a carried key → place/remove items in/from the chest → lock it again. Bundle with `unlock` in the same chunk — same NPCs are the standard-bearer for both verbs.

#### unlock

- **Direction:** mob-side missing
- **Surface today:** `internal/usercommands/unlock.go` (151 LoC). The
  mirror image of `lock.go`: same keyring + backpack lookup, same exit /
  container path, calls `SetUnlocked()` / `room.SetExitLock(exitName,
  false)`. The failure message hints at `picklock` as an alternative.
- **Why deferred:** Mob unlock is the counterpart to mob lock, and the
  same design questions apply: which mobs carry keys, which exits are they
  authorized to unlock, and what btree event triggers the behavior (mob
  needs to pass through a locked exit it holds a key for). The likeliest
  use case is a patrol guard unlocking a door on its route, locking it
  behind itself, then unlocking it again on the return pass — a full
  patrol-lock loop that needs both `lock_exit` and `unlock_exit` btree
  actions plus a btree-authoring pattern. Both verbs belong in the same
  followup chunk; splitting them would leave the pattern half-implemented.
- **Sketch of fix:** Pair with the `lock` followup chunk. New btree
  action `unlock_exit` (or `unlock_container`) that checks mob inventory
  for a matching key and calls `SetUnlocked()`. The patrol-guard YAML
  archetype demonstrates the lock-pass-unlock loop.
- **Proposed verdict:** memory-entry-only
- **Estimated size:** S-M (action ~60 LoC; effectively free once `lock`
  is done, since the logic is symmetric — same chunk)
- **Final verdict (user triage 2026-05-23):** **change-verdict to patch-as-followup-chunk.** Bundled with `lock` — same forager-locked-chest workflow chunk. Both verbs needed together; splitting would leave the pattern half-done.

#### throw

- **Direction:** mob-side missing
- **Surface today:** `internal/usercommands/throw.go` (216 LoC).
  Requires the player to be in combat, checks the special-move cooldown,
  finds a `Throwable` subtype item in backpack or bandolier, checks for
  spoilage (potions), consumes the item, then runs an AoE loop over all
  hostile room mobs using an opposed Dexterity + Skullduggery roll. On
  fumble (z-score <= -2.0), the item detonates on the thrower. On hit,
  applies `DamageMultiplier`-scaled physical damage and/or `BuffIds` to
  each mob.
- **Why deferred:** Mobs carrying and prioritizing throwable items
  (grenades, alchemical throwables) is a genuine design question. Unlike
  lockpicks (utility item, carried by specific thief-type mobs), grenades
  used by mobs would affect balance across a wide range of encounters.
  The mob would need inventory scanning logic to find a Throwable subtype,
  a decision on whether the friendly-fire AoE affects other mobs in the
  room (currently skips charmed companions on the player side), and a
  btree action with a combat-in-progress gate. The fumble-on-self path
  creates additional complexity for mob health management. This is a
  medium-complexity feature with non-trivial balance implications that
  belong in a dedicated chunk rather than a quick patch.
- **Sketch of fix:** New `internal/actions/throw.go` lifting the AoE
  resolution logic (the inner loop is already mostly target-agnostic).
  New btree action `try_throw` that finds the best Throwable in mob
  inventory, checks combat state + cooldown, and fires the action.
  Design decision: does the mob AoE also hit player companions in the
  room? (Player version skips charmed mobs — symmetry suggests yes,
  but that punishes the player unfairly.) Needs a config knob or
  explicit mob-AoE flag.
- **Proposed verdict:** memory-entry-only
- **Estimated size:** M (actions lift ~120 LoC, btree action ~40 LoC,
  balance design ~30 LoC of spec, plus the friendly-fire design decision)
- **Final verdict (user triage 2026-05-23):** **accept-proposed-verdict (memory-entry-only) with dependency note.** Deferred until DOGMud builds a real ranged weapon system; throw is conceptually adjacent to ranged combat and the design pass for one should inform the other. Memory entry should note this dependency.
