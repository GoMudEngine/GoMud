#!/usr/bin/env python3
"""
coord_inventory.py — Offline world-coordinate scanner for DOGMud.

PURPOSE
-------
The newbie-area rework (Pothole Coulee) mandates ZERO (x, y, z) coordinate
collisions, ever — within the new zone AND against every existing room in the
world (design spec tenet 8, docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md).

The engine already has the `cartcheck` admin command and a boot-time
`ValidateZoneConsistency` pass, BUT those only check coordinate consistency
*within a single zone* (each zone is crawled from its own root room, positions
derived from exit deltas, normalized to that root). They do NOT enforce a
shared cross-zone coordinate plane.

The cross-zone "world plane" the newbie-area spec reasons about lives in the
stored `coord:` field of each room YAML. (Note: the engine's Room struct does
NOT load `coord:` — it is an authoring-discipline / world-layout artifact that
authors maintain by hand so the zones tile a single consistent map.) This
scanner reads those stored `coord:` fields and checks GLOBAL uniqueness, which
is exactly the acceptance rule for every newbie-area chunk.

This is a filename/line parser — no YAML library required.

USAGE
-----
  python tools/coord_inventory.py
        Full report: totals, global + per-zone bounding boxes, any collisions.

  python tools/coord_inventory.py --zone stillwater
        Restrict the per-zone detail to one zone (global checks still run).

  python tools/coord_inventory.py --check-region X0 X1 Y0 Y1 Z0 Z1
        Verify a reserved box [X0..X1] x [Y0..Y1] x [Z0..Z1] (inclusive) is
        empty of all existing room coordinates. Exit code 0 if empty (clean),
        2 if any existing room falls inside it. This is how future chunks
        verify their new rooms land in the reserved Pothole Coulee budget
        without colliding with the existing world.

  python tools/coord_inventory.py --json
        Emit the scan as JSON for tooling.

EXIT CODES
----------
  0  clean (no global collisions; --check-region empty if requested)
  2  collisions found, OR --check-region found existing rooms in the box
"""

import argparse
import json
import os
import re
import sys
from collections import defaultdict

# Resolve the rooms dir relative to this file so it works from any CWD.
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, os.pardir))
ROOMS_DIR = os.path.join(REPO_ROOT, "_datafiles", "world", "dogmud", "rooms")

_ROOMID_RE = re.compile(r"^roomid:\s*(-?\d+)\s*$")
_ZONE_RE = re.compile(r"^zone:\s*(.+?)\s*$")
_COORD_HDR_RE = re.compile(r"^coord:\s*$")
_AXIS_RE = re.compile(r"^\s+([xyz]):\s*(-?\d+)\s*$")


class Room:
    __slots__ = ("roomid", "zone", "x", "y", "z", "path")

    def __init__(self, roomid, zone, x, y, z, path):
        self.roomid = roomid
        self.zone = zone
        self.x = x
        self.y = y
        self.z = z
        self.path = path

    @property
    def has_coord(self):
        return self.x is not None


def parse_room_file(path):
    """Lightweight parse: roomid, zone, and the coord: block (if present).

    The coord block is a fixed shape:
        coord:
          x: -11
          y: 8
          z: 0
    We only read axis lines immediately following a `coord:` header, so we
    don't accidentally pick up x/y/z keys nested elsewhere.
    """
    roomid = None
    zone = None
    x = y = z = None
    in_coord = False
    try:
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.rstrip("\n")
                if in_coord:
                    m = _AXIS_RE.match(line)
                    if m:
                        axis, val = m.group(1), int(m.group(2))
                        if axis == "x":
                            x = val
                        elif axis == "y":
                            y = val
                        elif axis == "z":
                            z = val
                        continue
                    else:
                        in_coord = False  # coord block ended
                if _COORD_HDR_RE.match(line):
                    in_coord = True
                    continue
                if roomid is None:
                    m = _ROOMID_RE.match(line)
                    if m:
                        roomid = int(m.group(1))
                        continue
                if zone is None:
                    m = _ZONE_RE.match(line)
                    if m:
                        zone = m.group(1).strip()
                        continue
    except OSError as e:
        print(f"WARN: could not read {path}: {e}", file=sys.stderr)
    if roomid is None:
        # Fall back to filename (e.g. 101.yaml -> 101)
        base = os.path.splitext(os.path.basename(path))[0]
        if base.isdigit():
            roomid = int(base)
    return Room(roomid, zone, x, y, z, path)


def scan(rooms_dir=ROOMS_DIR):
    rooms = []
    for zone_name in sorted(os.listdir(rooms_dir)):
        zone_dir = os.path.join(rooms_dir, zone_name)
        if not os.path.isdir(zone_dir):
            continue
        for fname in sorted(os.listdir(zone_dir)):
            if not fname.endswith(".yaml"):
                continue
            path = os.path.join(zone_dir, fname)
            room = parse_room_file(path)
            if room.zone is None:
                room.zone = zone_name  # folder fallback
            rooms.append(room)
    return rooms


def bounding_boxes(rooms):
    """Return {zone: (count_with_coord, minx, maxx, miny, maxy, minz, maxz)}."""
    boxes = {}
    by_zone = defaultdict(list)
    for r in rooms:
        if r.has_coord:
            by_zone[r.zone].append(r)
    for zone, rs in by_zone.items():
        xs = [r.x for r in rs]
        ys = [r.y for r in rs]
        zs = [r.z for r in rs]
        boxes[zone] = (
            len(rs),
            min(xs), max(xs),
            min(ys), max(ys),
            min(zs), max(zs),
        )
    return boxes


def find_collisions(rooms):
    """Return {(x,y,z): [room, ...]} for any tuple shared by >=2 rooms."""
    by_cell = defaultdict(list)
    for r in rooms:
        if r.has_coord:
            by_cell[(r.x, r.y, r.z)].append(r)
    return {cell: rs for cell, rs in by_cell.items() if len(rs) >= 2}


def rooms_in_region(rooms, x0, x1, y0, y1, z0, z1):
    """Existing rooms whose stored coord falls in the inclusive box."""
    lo_x, hi_x = min(x0, x1), max(x0, x1)
    lo_y, hi_y = min(y0, y1), max(y0, y1)
    lo_z, hi_z = min(z0, z1), max(z0, z1)
    hits = []
    for r in rooms:
        if not r.has_coord:
            continue
        if lo_x <= r.x <= hi_x and lo_y <= r.y <= hi_y and lo_z <= r.z <= hi_z:
            hits.append(r)
    return hits


def main(argv=None):
    ap = argparse.ArgumentParser(description="Offline world-coordinate scanner for DOGMud.")
    ap.add_argument("--zone", help="restrict per-zone detail to one zone")
    ap.add_argument("--json", action="store_true", help="emit JSON")
    ap.add_argument(
        "--check-region", nargs=6, type=int, metavar=("X0", "X1", "Y0", "Y1", "Z0", "Z1"),
        help="verify the inclusive box [X0..X1]x[Y0..Y1]x[Z0..Z1] is empty of existing rooms",
    )
    args = ap.parse_args(argv)

    rooms = scan()
    with_coord = [r for r in rooms if r.has_coord]
    without_coord = [r for r in rooms if not r.has_coord]
    boxes = bounding_boxes(rooms)
    collisions = find_collisions(rooms)
    unique_cells = len({(r.x, r.y, r.z) for r in with_coord})

    if args.json:
        out = {
            "total_room_files": len(rooms),
            "rooms_with_coord": len(with_coord),
            "rooms_without_coord": len(without_coord),
            "unique_coordinates": unique_cells,
            "collision_count": len(collisions),
            "zone_boxes": {
                z: {
                    "rooms": b[0],
                    "x": [b[1], b[2]], "y": [b[3], b[4]], "z": [b[5], b[6]],
                }
                for z, b in sorted(boxes.items())
            },
        }
        if with_coord:
            out["global_box"] = {
                "x": [min(r.x for r in with_coord), max(r.x for r in with_coord)],
                "y": [min(r.y for r in with_coord), max(r.y for r in with_coord)],
                "z": [min(r.z for r in with_coord), max(r.z for r in with_coord)],
            }
        print(json.dumps(out, indent=2))
    else:
        print("=== DOGMud World Coordinate Inventory ===")
        print(f"Total room files          : {len(rooms)}")
        print(f"Rooms with stored coord   : {len(with_coord)}")
        print(f"Rooms WITHOUT coord       : {len(without_coord)} "
              f"(maze/instance/root rooms — not on the world plane)")
        print(f"Unique (x,y,z) coordinates: {unique_cells}")
        print(f"Global collisions         : {len(collisions)}")
        if with_coord:
            gx = (min(r.x for r in with_coord), max(r.x for r in with_coord))
            gy = (min(r.y for r in with_coord), max(r.y for r in with_coord))
            gz = (min(r.z for r in with_coord), max(r.z for r in with_coord))
            print(f"Global bounding box       : "
                  f"x[{gx[0]}..{gx[1]}] y[{gy[0]}..{gy[1]}] z[{gz[0]}..{gz[1]}]")
        print()
        print("Per-zone bounding boxes (zones with stored coords):")
        zones = sorted(boxes.items())
        if args.zone:
            zones = [(z, b) for z, b in zones if z == args.zone]
        print(f"  {'zone':<26} {'rooms':>5}  {'x-range':>12}  {'y-range':>12}  {'z-range':>10}")
        for z, b in zones:
            cnt, minx, maxx, miny, maxy, minz, maxz = b
            print(f"  {z:<26} {cnt:>5}  "
                  f"{f'[{minx}..{maxx}]':>12}  "
                  f"{f'[{miny}..{maxy}]':>12}  "
                  f"{f'[{minz}..{maxz}]':>10}")
        if collisions:
            print()
            print("!!! COLLISIONS (same coord tuple shared by multiple rooms) !!!")
            for cell, rs in sorted(collisions.items()):
                ids = ", ".join(f"{r.zone}/{r.roomid}" for r in rs)
                print(f"  {cell}: {ids}")

    rc = 0
    if collisions:
        rc = 2

    if args.check_region:
        x0, x1, y0, y1, z0, z1 = args.check_region
        hits = rooms_in_region(rooms, x0, x1, y0, y1, z0, z1)
        print()
        print(f"--- Region check: x[{min(x0,x1)}..{max(x0,x1)}] "
              f"y[{min(y0,y1)}..{max(y0,y1)}] z[{min(z0,z1)}..{max(z0,z1)}] ---")
        if hits:
            print(f"NOT EMPTY: {len(hits)} existing room(s) inside the region:")
            for r in sorted(hits, key=lambda r: (r.zone, r.roomid)):
                print(f"  {r.zone}/{r.roomid} at ({r.x},{r.y},{r.z})")
            rc = 2
        else:
            print("EMPTY: no existing rooms occupy this region. Clean to reserve/build.")

    return rc


if __name__ == "__main__":
    sys.exit(main())
