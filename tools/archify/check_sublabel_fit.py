#!/usr/bin/env python3
"""Flag archify sublabels and tags that overflow their node box.

WHY THIS EXISTS
---------------
Archify's own validator checks a node's `label` against the box width and
errors if it is too wide. It does NOT check `sublabel` or `tag`, and it does
not wrap either of them: each renders as a single `<text>` element with
`text-anchor="middle"`, so an over-long one simply spills out over its
neighbours. That is why all six diagrams could pass 9/9 showcase validation
while visibly overlapping on screen.

Worse, every renderer's too-wide-label message advises the author to "move
detail to sublabel", which moves the text into the one field nothing measures.

This script applies the renderers' own width heuristic to the fields they
forgot. Run it after editing any spec and before delivering.

The upstream fix for this is written up at
docs/upstream/archify-sublabel-pr-brief.md, ready to hand to a clone of
tt-a1i/archify. This script is the local workaround, not the proposed fix.

The heuristic mirrors `textUnits(text) * 6.2` at font-size 10 in
renderers/shared/utils.mjs, scaled to each field's real font size:

    estimated px = units * 0.62 * fontSize

where a full-width (CJK) character counts as two units. Geometry read from
the renderers on 2026-08-04:

    architecture  sublabel font-size 9   box = component size[0]
    sequence      sublabel font-size 7   box = layout.participantW (86)
    dataflow      sublabel font-size 7   box = node.width
    lifecycle     sublabel font-size 7   box = state.width

Usage:
    python tools/archify/check_sublabel_fit.py [spec.json ...]

With no arguments it checks every spec in tools/archify/specs/.
Exits non-zero if anything overflows.
"""

import glob
import json
import os
import sys
import unicodedata

# Slack in px before a sublabel is reported. The renderers allow +6 on labels;
# sublabels sit inside the same box so the same tolerance applies.
SLACK_PX = 6.0
PX_PER_UNIT_AT_10 = 6.2

# diagram_type -> (collection key, how to get the box width, font sizes)
SEQUENCE_PARTICIPANT_W = 86.0

TYPES = {
    "architecture": ("components", "size0", {"sublabel": 9.0, "tag": 7.0}),
    "sequence": ("participants", "fixed86", {"sublabel": 7.0}),
    "dataflow": ("nodes", "width", {"sublabel": 7.0, "tag": 7.0}),
    "lifecycle": ("states", "width", {"sublabel": 7.0, "tag": 7.0}),
    "workflow": ("nodes", "width", {"sublabel": 7.0, "tag": 7.0}),
}


def text_units(s):
    """Mirror of textUnits() in renderers/shared/utils.mjs."""
    units = 0
    for ch in str(s or ""):
        units += 2 if unicodedata.east_asian_width(ch) in ("F", "W") else 1
    return units


def est_px(s, font_size):
    return text_units(s) * (PX_PER_UNIT_AT_10 / 10.0) * font_size


def box_width(item, mode):
    if mode == "fixed86":
        return SEQUENCE_PARTICIPANT_W
    if mode == "size0":
        size = item.get("size")
        # Architecture components may omit size; the renderer defaults it.
        return float(size[0]) if isinstance(size, list) and size else 160.0
    w = item.get("width")
    return float(w) if w else 126.0


def check(path):
    with open(path, encoding="utf-8") as fh:
        spec = json.load(fh)

    dtype = spec.get("diagram_type")
    if dtype not in TYPES:
        return [("?", "unknown diagram_type %r" % dtype, 0, 0)]

    key, width_mode, fields = TYPES[dtype]
    findings = []
    for item in spec.get(key, []) or []:
        width = box_width(item, width_mode)
        for field, font in fields.items():
            text = item.get(field)
            if not text:
                continue
            px = est_px(text, font)
            if px > width + SLACK_PX:
                findings.append((item.get("id", "?"), "%s: %r" % (field, text), px, width))
    return findings


def main(argv):
    paths = argv[1:]
    if not paths:
        here = os.path.dirname(os.path.abspath(__file__))
        paths = sorted(glob.glob(os.path.join(here, "specs", "*.json")))

    if not paths:
        print("no specs found", file=sys.stderr)
        return 2

    total = 0
    for path in paths:
        findings = check(path)
        name = os.path.basename(path)
        if not findings:
            print("OK   %s" % name)
            continue
        total += len(findings)
        print("FAIL %s  (%d overflowing)" % (name, len(findings)))
        for node_id, what, px, width in sorted(findings, key=lambda f: -f[2]):
            over = px - width
            print("       %-18s %-6s box=%dpx  est=%dpx  over by %dpx"
                  % (node_id, "", width, px, over))
            print("       %s%s" % (" " * 18, what))

    if total:
        print()
        print("%d field(s) overflow their box." % total)
        print("Archify does not wrap or measure these, so they render on top of")
        print("their neighbours. Shorten the text, or widen the node where the")
        print("diagram type allows it (architecture size, dataflow/lifecycle")
        print("width; sequence participants are fixed at 86px).")
        return 1

    print()
    print("All sublabels and tags fit their boxes.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
