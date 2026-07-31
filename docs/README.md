# DOGMud documentation

Everything that isn't code. Start with [`world.md`](world.md) if you want the
setting, or [`schemas/`](schemas/) if you want to author content.

## Start here

| Path | What's in it |
|------|--------------|
| [`world.md`](world.md) | The world-design document — lore, factions, zones, species |
| [`PATCH_NOTES.md`](PATCH_NOTES.md) | Dated shipping log of every change |
| [`PATH_TO_1.0.md`](PATH_TO_1.0.md) | Remaining work before the 1.0 tag |

## Reference

| Path | What's in it |
|------|--------------|
| [`schemas/`](schemas/) | YAML schema references (room, mob, item, spell, buff, dialogue, schedule, patrol) |
| [`architecture/`](architecture/) | System-level architecture notes and deliberate divergences from upstream |
| [`economy/`](economy/) | Living-economy design and tuning |
| [`balance/`](balance/) | Combat and progression tuning |
| [`worldbuilding/`](worldbuilding/) | Zone expansion plan, coordinate map, settlement canon, world atlas |

## Guides

| Path | What's in it |
|------|--------------|
| [`guides/CONTENT_GENERATION_GUIDE.md`](guides/CONTENT_GENERATION_GUIDE.md) | How the content-generation workflow fits together |
| [`guides/github_guide.md`](guides/github_guide.md) | Branch strategy and Git workflow |
| [`guides/DEPLOYMENT_GUIDE.md`](guides/DEPLOYMENT_GUIDE.md) | Deploying to the production droplet |
| [`guides/ADVERTISING_LISTINGS.md`](guides/ADVERTISING_LISTINGS.md) | Copy used for MUD-listing sites |

## Audits & findings

Point-in-time reports. Each is true as of its date and is **not** kept current —
verify against the code before acting on one.

| Path | What's in it |
|------|--------------|
| [`audits/`](audits/) | Tech-debt and test-coverage audits, the code-smell review queue, upstream cherry-pick triage, playtest findings |
| [`perf/`](perf/) | Performance baselines and profiling notes |

## History

| Path | What's in it |
|------|--------------|
| [`roadmaps/`](roadmaps/) | Long-form roadmaps: development plan, combat-state, mob-aliveness |
| [`superpowers/`](superpowers/) | Per-feature specs and implementation plans, live and completed |
| [`archive/`](archive/) | Retired documents and old bug screenshots |
| [`upstream/`](upstream/) | Artifacts inherited from the upstream GoMud engine |
| [`images/`](images/) | Screenshots used by the top-level README |

Per-package developer notes live beside the code, as `context.md` in each
`internal/` and `modules/` package — see the convention in the repo-root
`CLAUDE.md`.
