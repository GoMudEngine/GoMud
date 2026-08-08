# Synthetic playtest profiles (Chunk 0.3b)

Tracked templates for ephemeral playtest materialization. Runtime never reads
`_archive/prod-users`; that tree is an offline authoring reference only.

## IDs

| ID | Role | Intent |
|----|------|--------|
| `fresh` | user | Naked starter; newbie/onboarding rooms via manifest `start_room` |
| `early` | user | Basic kit past first lessons |
| `mid` | user | Mixed skills/spells |
| `veteran` | user | High-end kit (sanitized from a Meirok-class archive offline) |
| `specialist-caster` | user | Casting-focused kit |
| `admin` | admin | Admin-surface tests |

## Authoring rules

- No passwords, inbox, email, macros/aliases/ticks/triggers
- Fictional `username` / `character.name` only (materialize replaces username)
- Item/spell/skill/quest refs must exist in world data
- Prefer small inventories; overlays grant run-specific extras

## Session context (non-`fresh`)

`context.md` in this directory is a short MUD orientation for playtest agents
on kit profiles (`early`, `mid`, `veteran`, `specialist-caster`, `admin`).
Drivers should read it when `ephemeral.profile` is set and is not `fresh`.
`fresh` / creation-flow runs rely on in-game onboarding instead.

AI command pacing: `Network.AICommandsPerRound` (shipped **3** per
`Timing.RoundSeconds` round, currently ~4s) — see `context.md`.

## Container path

Runner image copies this directory to `/app/playtest/profiles`.
