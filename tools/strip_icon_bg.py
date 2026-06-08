#!/usr/bin/env python3
"""Strip baked light/card backgrounds from generated item icons.

gpt-image-2 sometimes ignores the transparent-background request and bakes
a light rounded-card behind the object. Each generated icon has a faint
dark outline around its silhouette, so we flood-fill from the image border
over connected light pixels and stop at that outline, then feather the
boundary. Operates in-place on the file paths given (or, with no args, on
GoMudAssetsPack/materials/*.png + the 5 new DOGMud-only-slot armor icons).

Run: python tools/strip_icon_bg.py [path.png ...]
"""
import os
import sys
from collections import deque

try:
    from PIL import Image
except ImportError:
    sys.exit("Pillow required: pip install pillow")

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PACK = os.path.join(REPO, "GoMudAssetsPack")
THRESH = 175      # luma below this is treated as object/outline (flood stops)
FEATHER_LUMA = 150  # boundary pixels lighter than this get alpha-softened
CARD_TOLERANCE = 32  # RGB distance from the sampled card color to key out


def _luma(r, g, b):
    return 0.299 * r + 0.587 * g + 0.114 * b


def _card_color(px, w, h):
    """Median color of the opaque-light border ring, or None if the border
    isn't predominantly a baked card (so we skip the chroma-key)."""
    rs, gs, bs = [], [], []
    ring = []
    for x in range(w):
        ring.append((x, 0))
        ring.append((x, h - 1))
    for y in range(h):
        ring.append((0, y))
        ring.append((w - 1, y))
    for x, y in ring:
        r, g, b, a = px[x, y]
        if a > 200 and _luma(r, g, b) > 170:
            rs.append(r)
            gs.append(g)
            bs.append(b)
    if len(rs) < w:        # border not mostly card -> no baked background
        return None
    rs.sort(); gs.sort(); bs.sort()
    mid = len(rs) // 2
    return (rs[mid], gs[mid], bs[mid])


def strip_bg(im):
    im = im.convert("RGBA")
    w, h = im.size
    px = im.load()
    bg = [[False] * w for _ in range(h)]
    # Pass 1 — chroma-key the baked card color everywhere (this is what
    # reaches ENCLOSED holes the edge flood-fill below can't, e.g. the
    # finger holes of knuckles or the gaps inside chain links). Object
    # highlights survive because they're brighter/different from the card.
    card = _card_color(px, w, h)
    if card:
        cr, cg, cb = card
        tol2 = CARD_TOLERANCE * CARD_TOLERANCE
        for y in range(h):
            for x in range(w):
                r, g, b, a = px[x, y]
                if a > 0 and (r - cr) ** 2 + (g - cg) ** 2 + (b - cb) ** 2 < tol2:
                    bg[y][x] = True
    # Pass 2 — edge flood-fill for any non-card light background.
    dq = deque()
    for x in range(w):
        dq.append((x, 0))
        dq.append((x, h - 1))
    for y in range(h):
        dq.append((0, y))
        dq.append((w - 1, y))
    while dq:
        x, y = dq.popleft()
        if x < 0 or y < 0 or x >= w or y >= h or bg[y][x]:
            continue
        r, g, b, a = px[x, y]
        if a > 0 and _luma(r, g, b) < THRESH:
            continue   # hit the object / its dark outline
        bg[y][x] = True
        dq.extend([(x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)])
    for y in range(h):
        for x in range(w):
            if bg[y][x]:
                px[x, y] = (0, 0, 0, 0)
    # Feather: soften light pixels left touching the transparent region.
    for y in range(h):
        for x in range(w):
            if bg[y][x]:
                continue
            r, g, b, a = px[x, y]
            if a == 0:
                continue
            touches = any(
                0 <= x + dx < w and 0 <= y + dy < h and bg[y + dy][x + dx]
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1))
            )
            if touches:
                l = _luma(r, g, b)
                if l > FEATHER_LUMA:
                    f = max(0.0, min(1.0, (255 - l) / 80.0))
                    px[x, y] = (r, g, b, int(a * f))
    return im


def default_targets():
    targets = []
    mats = os.path.join(PACK, "materials")
    if os.path.isdir(mats):
        for fn in sorted(os.listdir(mats)):
            if fn.lower().endswith(".png"):
                targets.append(os.path.join(mats, fn))
    for rel in ("armor/back/cloak.png", "armor/back/backpack.png",
                "armor/shoulders/pauldron.png", "armor/tail/tail_guard.png",
                "armor/wrist/bracer.png"):
        p = os.path.join(PACK, *rel.split("/"))
        if os.path.isfile(p):
            targets.append(p)
    return targets


def main(argv):
    targets = argv or default_targets()
    if not targets:
        sys.exit("no target icons found")
    for p in targets:
        strip_bg(Image.open(p)).save(p)
    print(f"stripped {len(targets)} icon background(s)")


if __name__ == "__main__":
    main(sys.argv[1:])
