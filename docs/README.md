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
| [`guides/TESTING_GUIDE.md`](guides/TESTING_GUIDE.md) | Reproducible race-enabled test baseline, local and CI |

## Audits & findings

Point-in-time reports. Each is true as of its date and is **not** kept current —
verify against the code before acting on one.

| Path | What's in it |
|------|--------------|
| [`audits/`](audits/) | Tech-debt and test-coverage audits, the code-smell review queue, upstream cherry-pick triage, playtest findings |
| [`audits/META_REVIEW_AGENT_WORK_2026-08-08.md`](audits/META_REVIEW_AGENT_WORK_2026-08-08.md) | Adversarial meta-review grading the Cursor and Jules agent work after 2026-08-04 |
| [`audits/ADVERSARIAL_REVIEW.md`](audits/ADVERSARIAL_REVIEW.md) | Jules' architectural teardown: global lock, autosave I/O, boot wiring, progression math |
| [`perf/`](perf/) | Performance baselines and profiling notes |

## History

| Path | What's in it |
|------|--------------|
| [`roadmaps/`](roadmaps/) | Long-form roadmaps: development plan, combat-state, mob-aliveness |
| [`roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`](roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md) | 37-finding remediation roadmap from the 2026-08-07 adversarial reviews. Note: its source doc `ADVERSARIAL_CODE_REVIEW_2026-08-07.md` is currently untracked in the repo root |
| [`superpowers/`](superpowers/) | Per-feature specs and implementation plans, live and completed |
| [`archive/`](archive/) | Retired documents and old bug screenshots |
| [`upstream/`](upstream/) | Upstream-facing material in both directions: artifacts inherited from the GoMud engine, and briefs for changes we want to send out to third-party tools |
| [`upstream/archify-sublabel-pr-brief.md`](upstream/archify-sublabel-pr-brief.md) | Ready-to-hand-off brief for a PR to `tt-a1i/archify`: four of its five renderers never measure `sublabel`/`tag` against the node box, so diagram text overflows while validation still passes |
| [`images/`](images/) | Screenshots used by the top-level README |

Per-package developer notes live beside the code, as `context.md` in each
`internal/` and `modules/` package — see the convention in the repo-root
`CLAUDE.md`.
