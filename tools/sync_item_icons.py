#!/usr/bin/env python3
"""Sync GoMudAssetsPack PNG icons into the web-served static dir.

Copies every GoMudAssetsPack/**/*.png into
_datafiles/html/public/static/images/items/ as a flat, basename-keyed
file, downscaling anything larger than 64x64 to 64x64 (LANCZOS, alpha
preserved). Writes manifest.json (sorted list of served basenames).
Idempotent. First-copy-wins on basename collisions, with a warning.

Run: python tools/sync_item_icons.py
"""
import json
import os
import sys

try:
    from PIL import Image
except ImportError:
    sys.exit("Pillow required: pip install pillow")

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC = os.path.join(REPO, "GoMudAssetsPack")
DST = os.path.join(REPO, "_datafiles", "html", "public", "static", "images", "items")
SIZE = 64


def main():
    os.makedirs(DST, exist_ok=True)
    if not os.path.isdir(SRC):
        sys.exit(f"Source not found: {SRC}")
    served = {}        # basename -> source relpath (first wins)
    collisions = []
    copied = resized = 0

    for root, dirs, files in os.walk(SRC):
        dirs.sort()
        for fn in sorted(files):
            if not fn.lower().endswith(".png"):
                continue
            src = os.path.join(root, fn)
            rel = os.path.relpath(src, SRC)
            if fn in served:
                collisions.append((fn, served[fn], rel))
                continue
            served[fn] = rel
            im = Image.open(src).convert("RGBA")
            if im.size != (SIZE, SIZE):
                im = im.resize((SIZE, SIZE), Image.LANCZOS)
                resized += 1
            im.save(os.path.join(DST, fn))
            copied += 1

    manifest = sorted(served.keys())
    with open(os.path.join(DST, "manifest.json"), "w") as f:
        json.dump(manifest, f, indent=0)  # one entry per line — clean git diffs

    print(f"synced {copied} icons ({resized} resized) -> {DST}")
    if collisions:
        print(f"WARNING: {len(collisions)} basename collision(s) (first wins):")
        for fn, kept, dropped in collisions:
            print(f"  {fn}: kept {kept}, dropped {dropped}")
    print(f"manifest.json: {len(manifest)} entries")


if __name__ == "__main__":
    main()
