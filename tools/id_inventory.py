#!/usr/bin/env python3
"""
DOGMud ID Inventory Tool

Walks the world data directory, parses IDs from filenames, and reports
per-zone ranges, gaps, and next-free IDs across rooms / mobs / items /
spells / buffs / quests / dialogue / behaviors.

Run before creating any new YAML to avoid ID collisions, and before
dispatching parallel content-creation subagents to allocate
non-overlapping ID blocks.

Usage:
    python tools/id_inventory.py                   # all types, all zones
    python tools/id_inventory.py --type rooms      # one type
    python tools/id_inventory.py --zone stillwater # one zone (any type)
    python tools/id_inventory.py --alloc rooms 20  # reserve next 20 free
                                                   # IDs (for parallel
                                                   # subagent dispatch)

The script does NOT read YAML contents — it only parses filenames, which
encode the entity ID as a leading integer (e.g. `4105.yaml`,
`120-banker.yaml`, `30036-healing_salve.yaml`).
"""

import argparse
import re
import sys
from collections import defaultdict
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
DATA_ROOT = PROJECT_ROOT / "_datafiles" / "world" / "dogmud"

# (subdir-relative-to-DATA_ROOT, has-zone-folder)
# When has_zone is False, all entries roll up under "(global)".
# Note: spells use STRING IDs (e.g. "blood-boil.yaml") and are excluded —
# numeric collision detection doesn't apply to them.
TYPES = {
    "rooms":     ("rooms",     True),
    "mobs":      ("mobs",      True),
    "items":     ("items",     True),   # "zone" folder = category-NNNN
    "behaviors": ("behaviors", True),
    "buffs":     ("buffs",     False),
    "quests":    ("quests",    False),
    "dialogue":  ("dialogue",  False),
}

FILENAME_RE = re.compile(r"^(\d+)(?:-.*)?\.yaml$")


def collect(type_name):
    """Return {zone: sorted list of ids}."""
    subdir, has_zone = TYPES[type_name]
    root = DATA_ROOT / subdir
    out = defaultdict(set)
    if not root.exists():
        return out
    for path in root.rglob("*.yaml"):
        m = FILENAME_RE.match(path.name)
        if not m:
            continue
        id_num = int(m.group(1))
        zone = path.parent.name if has_zone else "(global)"
        out[zone].add(id_num)
    return {z: sorted(ids) for z, ids in out.items()}


def find_gaps(ids):
    """Return list of (start, end) tuples for gaps in a sorted id list."""
    gaps = []
    for a, b in zip(ids, ids[1:]):
        if b > a + 1:
            gaps.append((a + 1, b - 1))
    return gaps


def fmt_gaps(gaps, max_show=4):
    if not gaps:
        return "-"
    out = []
    for a, b in gaps[:max_show]:
        out.append(str(a) if a == b else f"{a}-{b}")
    s = ", ".join(out)
    if len(gaps) > max_show:
        s += f", +{len(gaps) - max_show} more"
    return s


def report(type_name, zone_filter=None):
    data = collect(type_name)
    if not data:
        return
    zones = sorted(data.keys())
    if zone_filter:
        zones = [z for z in zones if z == zone_filter]
    if not zones:
        return

    all_ids = sorted({i for ids in data.values() for i in ids})
    global_max = all_ids[-1] if all_ids else 0

    print(f"\n=== {type_name.upper()} ===")
    print(f"  {'zone':<26} {'count':>6}  {'range':<14}  {'gaps':<26}  next-in-zone")
    print(f"  {'-' * 26} {'-' * 6}  {'-' * 14}  {'-' * 26}  {'-' * 12}")
    for zone in zones:
        ids = data[zone]
        if not ids:
            continue
        lo, hi = ids[0], ids[-1]
        gaps = find_gaps(ids)
        # Smallest gap entry, else hi+1
        next_in_zone = gaps[0][0] if gaps else hi + 1
        print(
            f"  {zone:<26} {len(ids):>6}  {f'{lo}-{hi}':<14}  "
            f"{fmt_gaps(gaps):<26}  {next_in_zone}"
        )
    if not zone_filter:
        print(f"  global next free (>= max+1): {global_max + 1}")


def alloc(type_name, count):
    """Print a contiguous free ID block of `count` past the global max.

    This block is *advisory* — the caller is responsible for embedding
    it in a subagent's dispatch prompt. The script does not write any
    state. If two parents call `--alloc` simultaneously they'll get the
    same range; this is a single-developer tool.
    """
    data = collect(type_name)
    used = sorted({i for ids in data.values() for i in ids})
    start = (used[-1] + 1) if used else 1
    end = start + count - 1
    print(f"{type_name}: reserve IDs {start}-{end} ({count} IDs)")
    print(f"  past current global max ({used[-1] if used else 0})")
    print(f"  embed in subagent prompt: 'use {type_name} IDs in {start}-{end}'")


def main():
    parser = argparse.ArgumentParser(
        description="DOGMud ID Inventory — list IDs by zone, find gaps,"
                    " reserve blocks for parallel dispatch.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--type",
        choices=list(TYPES.keys()) + ["all"],
        default="all",
        help="entity type (default: all)",
    )
    parser.add_argument("--zone", help="filter to one zone")
    parser.add_argument(
        "--alloc",
        nargs=2,
        metavar=("TYPE", "COUNT"),
        help="reserve next COUNT free IDs of TYPE (for parallel dispatch)",
    )
    args = parser.parse_args()

    if args.alloc:
        type_name, count_str = args.alloc
        if type_name not in TYPES:
            sys.exit(f"unknown type: {type_name} (choose from {list(TYPES.keys())})")
        try:
            count = int(count_str)
        except ValueError:
            sys.exit(f"count must be a positive integer: {count_str}")
        if count < 1:
            sys.exit("count must be >= 1")
        alloc(type_name, count)
        return

    types = list(TYPES.keys()) if args.type == "all" else [args.type]
    for t in types:
        report(t, args.zone)


if __name__ == "__main__":
    main()
