# Archify upstream PR brief: sublabel and tag overflow

Handoff note. Copy this into a clone of `tt-a1i/archify` and work from it.
Written 2026-08-04 against upstream `main` at version **2.13.0**.

## TL;DR

Four of archify's five renderers never measure `sublabel` or `tag` against the
node box, and never shrink them to fit, so over-long ones render on top of
neighbouring nodes while validation still reports a clean 9/9 showcase pass.

The fifth renderer, `workflow`, already does both correctly. **The fix is to
lift its existing pattern into shared code and apply it to the other four.**
This is not a new feature, it is an existing upstream behaviour that four
renderers were missed on.

## The problem

`sublabel` and `tag` render as a single `<text>` element with
`text-anchor="middle"` at a **hardcoded** font size, with no width measurement
and no wrapping:

| Renderer | sublabel | tag | Measured? | Shrink to fit? |
|---|---|---|---|---|
| `workflow` | computed | n/a | **yes** | **yes** |
| `architecture` | fixed 9 | fixed 7 | no | no |
| `sequence` | fixed 7 | n/a | no | no |
| `dataflow` | fixed 7 | fixed 7 | no | no |
| `lifecycle` | fixed 7 | fixed 7 | no | no |

All four rows verified against upstream `main` on 2026-08-04.

Two things make this worse than a cosmetic gap:

1. **Validation gives false confidence.** A diagram can pass
   `validate --quality showcase` with 9/9 checks, 0 errors and 0 warnings while
   text visibly overlaps on screen. The `label` field is measured; the fields
   next to it are not.
2. **The tool actively steers authors into it.** Every renderer's too-wide-label
   message advises moving text into the unchecked field:

   ```
   Label "..." is wider than component "..." — shorten the label,
   move detail to sublabel, or widen size.
   ```

   `render-architecture.mjs`, `render-dataflow.mjs`, `render-lifecycle.mjs` and
   `render-sequence.mjs` all say some form of "move detail to sublabel".

### Real-world impact

Authoring six showcase diagrams produced **21 overflowing fields across five
specs**, every one of which passed validation. Worst case was a lifecycle state
whose sublabel needed roughly 234px inside a 126px box. The overlap was only
discovered by a human looking at the rendered page.

### Reproduction

Take any `lifecycle`, `architecture`, `sequence` or `dataflow` example from
`examples/`, set a long sublabel on one node, and validate:

```bash
node bin/archify.mjs validate lifecycle examples/agent-run.lifecycle.json \
  --quality showcase --json
```

With `sublabel` set to something like
`"CheckStatProgression, one draw from a 10000 step range"` on a 126px state, the
receipt still reports 9/9 with zero errors. Render it and the text runs across
its neighbours.

For contrast, the same mutation on a `workflow` node is correctly rejected. See
the existing case at `test/layout-rules.test.mjs`:

```js
['workflow: node sublabel wider than its legible minimum', 'workflow',
  (d) => { d.nodes[0].sublabel = 'This supporting sentence is much too long for one workflow node'; },
  ['Sublabel', 'legible', 'increase node.width']],
```

## The fix

`render-workflow.mjs` already contains the whole solution in about 18 lines
(around lines 100 to 118 upstream). It is currently file-local:

```js
const nodeTextFit = {
  widthFactor: 0.6,
  horizontalPadding: 8,
  labelPreferred: 11,
  labelMinimum: 9,
  sublabelPreferred: 8,
  sublabelMinimum: 6,
};

function fittedNodeFontSize(text, width, preferred, minimum) {
  const units = Math.max(1, textUnits(text));
  const available = Math.max(1, width - nodeTextFit.horizontalPadding);
  const fitted = Math.min(preferred, available / (units * nodeTextFit.widthFactor));
  return Math.max(minimum, Math.floor(fitted * 10) / 10);
}

function minimumNodeTextWidth(text, minimum) {
  return textUnits(text) * minimum * nodeTextFit.widthFactor;
}
```

It is used in two places: `renderNode` calls `fittedNodeFontSize` so the text
shrinks toward a legible minimum, and `validateWorkflow` calls
`minimumNodeTextWidth` to raise a problem when even that minimum will not fit.

**Proposed change:**

1. Move `nodeTextFit`, `fittedNodeFontSize` and `minimumNodeTextWidth` into
   `renderers/shared/` (`utils.mjs` already exports `textUnits`, which they
   depend on). Parameterise the font constants: workflow uses
   preferred 8 / minimum 6 for sublabels, but `architecture` renders sublabels
   at 9, so the defaults cannot simply be shared verbatim.
2. In `architecture`, `sequence`, `dataflow` and `lifecycle`: use
   `fittedNodeFontSize` for the `sublabel` (and `tag`) `<text>` font size
   instead of the hardcoded value, and add the `minimumNodeTextWidth`
   validation next to the existing label check.
3. Update the four "move detail to sublabel" messages, since that advice now
   points at a field that is also measured. Suggest "shorten the label, or
   widen the node".
4. Add four cases to `test/layout-rules.test.mjs` mirroring the existing
   workflow one, one per renderer.

`sequence` needs a judgement call: participants are a fixed
`layout.participantW: 86`, with no author-facing width. So shrink-to-fit
applies, but the "increase node.width" half of the advice does not. The message
there should say to shorten the sublabel, full stop.

Rough size: about 150 lines including tests.

### Test and verify

```bash
npm test
```

Note the test script reaches outside the package (`node ../scripts/run-tests.mjs`,
`../scripts/check-release-identity.mjs`), so it only runs from a **full clone**.
The skill as distributed is just the `archify/` subdirectory and cannot run its
own suite.

Before and after, check the rendered SVG, not only the receipt. The whole point
of this bug is that the receipt was already green.

## Deliberately NOT in this PR

These were all worked around locally. They are design decisions that belong to
the maintainer, so they are better raised as issues than as a speculative PR.

**1. Autoplay dwell is hardcoded and not authorable.** Both
`STORY_FOLLOW_MIN_DWELL_MS` (guided chapter story) and `JOURNEY_DWELL_MS`
(node-by-node journey) are `1100` in `assets/template.html`. Because
`storyBeatDwell` takes `max(floor, VIEW_INTERVAL_MS / total)`, the floor always
wins for a three-or-more chapter story, so every beat gets 1.1 seconds. Users
reported this as far too fast to read a beat's note and look at the region it
just panned to. No schema field or CLI flag exposes it, so the only workaround
is patching the installed template and re-delivering every diagram. A
`meta.motion` or similar authored field would remove the need.

**2. Default theme follows the OS with no author override.** The fallback is
`prefers-color-scheme: light ? 'light' : 'dark'`, so a diagram intended to be
read dark opens light on a light-mode machine. Note for anyone patching this:
there are **two** resolution sites, the `<head>` bootstrap that sets
`data-theme` before first paint and the runtime's `resolveInitial()`. Patching
only the first gives a flash of dark that reverts to light.

**3. Repository evidence is architecture-only.** `--repo-root`,
`meta.repository` and node `sources` work for `architecture` and are rejected
for the other four types (`--repo-root` exits 2; `meta.repository` fails
`schema/additionalProperties`). The authoring-time guard against invented file
paths is valuable and there is no obvious reason it could not apply to
`sequence`, `dataflow` and `lifecycle`. This is a real feature request rather
than a bug, and probably a larger piece of work.

## Reference implementation of the check

`tools/archify/check_sublabel_fit.py` in the DOGMud repo is a standalone
checker written to work around this bug. It applies the renderers' own width
heuristic (`textUnits * 0.62 * fontSize`) to `sublabel` and `tag` and reports
overflow per node. It is not proposed for upstream, but it documents the
per-renderer geometry (font sizes and width sources) in one place and may be
useful when writing the tests.

Its heuristic is deliberately cruder than workflow's: it assumes a fixed font
size, because that is what the four unfixed renderers actually do. The upstream
fix should use the shrink-to-fit approach instead.
