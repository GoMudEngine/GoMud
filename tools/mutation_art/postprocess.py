"""Downscale generated 1024px mutation emblems to 256px PNGs.

Usage: python tools/mutation_art/postprocess.py <in_dir> [--out-dir DIR]
In-dir: raw 1024x1024 PNGs named <mutationid>.png from image-gen.
Out:    _datafiles/html/public/static/images/mutations/<mutationid>.png
"""
import argparse
import pathlib

from PIL import Image

DEFAULT_OUT = pathlib.Path("_datafiles/html/public/static/images/mutations")
SIZE = 256

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("in_dir", type=pathlib.Path)
    ap.add_argument("--out-dir", type=pathlib.Path, default=DEFAULT_OUT)
    args = ap.parse_args()
    args.out_dir.mkdir(parents=True, exist_ok=True)
    count = 0
    for src in sorted(args.in_dir.glob("*.png")):
        img = Image.open(src).convert("RGB")  # solid-tint bg: no alpha needed
        img = img.resize((SIZE, SIZE), Image.LANCZOS)
        img.save(args.out_dir / src.name, optimize=True)
        count += 1
    print(f"postprocessed {count} emblems -> {args.out_dir}")

if __name__ == "__main__":
    main()
