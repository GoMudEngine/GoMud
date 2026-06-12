#!/usr/bin/env python3
"""Newbie-area manifest conformance check.

Asserts every authored Pothole Coulee room against a hardcoded copy of the
chunk-1 hub sub-spec (docs/superpowers/specs/2026-06-12-newbie-chunk1-hub-subspec.md
section 3): roomid/title/biome/coord/exits (exact set + targets), sanctuary
mutator, noun-count floor (>=2 hub, >=1 stub), single-token noun keys, and that
every noun key appears (case-insensitive) in the room's description body.

Grows with later chunks. Run from repo root: python tools/newbie_manifest_check.py
"""
import os
import sys
import yaml

ROOMS_DIR = os.path.join("_datafiles", "world", "dogmud", "rooms", "pothole_coulee")
MOBS_DIR = os.path.join("_datafiles", "world", "dogmud", "mobs", "pothole_coulee")

# Pothole Coulee NPC roster (Phase M). Per mob:
#   (expected_filename, name, host_room). All 8 are non-combatant, charm-immune,
#   maxwander 0, groups {humanoid, coulee_folk}, zone "Pothole Coulee".
# 9104 (Trader Onna) is additionally the shopkeeper and must carry a non-empty
# character.shop list with itemid entries.
NPC_MANIFEST = {
    9100: ("9100-cleric_hadwen.yaml", "Cleric Hadwen", 5200),
    9101: ("9101-innkeep_tally.yaml", "Innkeep Tally", 5205),
    9102: ("9102-sala_the_mender.yaml", "Sala the Mender", 5209),
    9103: ("9103-ledger_keeper_croup.yaml", "Ledger-Keeper Croup", 5208),
    9104: ("9104-trader_onna.yaml", "Trader Onna", 5207),
    9105: ("9105-granny_wicker.yaml", "Granny Wicker", 5210),
    9106: ("9106-crier_toke.yaml", "Crier Toke", 5203),
    9107: ("9107-warden_esk.yaml", "Warden Esk", 5215),
}
SHOPKEEPER_MOBID = 9104

# (title, biome, (x,y,z), {dir: target}, min_nouns). biome None = not asserted (stubs).
# Exit sets are the EXACT expected set incl. stub-attachment exits added to hosts.
MANIFEST = {
    5200: ("The Awakening Pool", "water", (45, 0, 0), {"west":5201,"east":5202,"south":5203,"north":5211}, 2),
    5201: ("West Shore Path", "shore", (44, 0, 0), {"east":5200,"west":5210,"south":5216,"north":5218}, 2),
    5202: ("East Shore Path", "shore", (46, 0, 0), {"west":5200,"east":5209,"south":5208,"north":5217}, 2),
    5203: ("Hub Square", "city", (45, 1, 0), {"north":5200,"south":5204,"west":5216,"east":5208}, 2),
    5204: ("Market Row", "city", (45, 2, 0), {"north":5203,"west":5205,"east":5207,"south":5215}, 2),
    5205: ("The Drowned Lantern", "house", (44, 2, 0), {"east":5204,"up":5206}, 2),
    5206: ("Lantern Sleeping Loft", "house", (44, 2, 1), {"down":5205}, 2),
    5207: ("Coulee Provisions", "house", (46, 2, 0), {"west":5204,"east":5222}, 2),
    5208: ("Strongbox House", "house", (46, 1, 0), {"west":5203,"north":5202}, 2),
    5209: ("The Mending Hut", "house", (47, 0, 0), {"west":5202,"east":5220,"north":5226}, 2),
    5210: ("Wickerwork Cottage", "house", (43, 0, 0), {"east":5201,"west":5221}, 2),
    5211: ("Basalt Stair", "cliffs", (45, -1, 0), {"south":5200,"north":5212}, 2),
    5212: ("School Shelf", "cliffs", (45, -2, 0), {"south":5211,"north":5213}, 2),
    5213: ("Chrysalis School Hall", "house", (45, -3, 0), {"south":5212,"west":5214}, 2),
    5214: ("Cleric's Study", "house", (44, -3, 0), {"east":5213}, 2),
    5215: ("The Threshold House", "house", (45, 3, 0), {"north":5204}, 2),
    5216: ("Stilt-House Walk", "shore", (44, 1, 0), {"north":5201,"east":5203,"west":5223}, 2),
    5217: ("North Shore Overlook", "cliffs", (46, -1, 0), {"south":5202,"north":5225}, 2),
    5218: ("Reed Jetty", "shore", (44, -1, 0), {"south":5201,"north":5224}, 2),
    # Stubs: return-only, biome not asserted, >=1 noun.
    5220: ("Dry Coulee Mouth", None, (48, 0, 0), {"west":5209}, 1),
    5221: ("Talus Gap", None, (42, 0, 0), {"east":5210}, 1),
    5222: ("Reedwash Mouth", None, (47, 2, 0), {"west":5207}, 1),
    5223: ("Scrub Draw", None, (43, 1, 0), {"east":5216}, 1),
    5224: ("Stargazer Cut", None, (44, -2, 0), {"south":5218}, 1),
    5225: ("Old Field Track", None, (46, -2, 0), {"south":5217}, 1),
    5226: ("Bluff Steps", None, (47, -1, 0), {"south":5209}, 1),
}


def check_room(rid, spec):
    title, biome, coord, exits, min_nouns = spec
    path = os.path.join(ROOMS_DIR, f"{rid}.yaml")
    fails = []
    if not os.path.exists(path):
        return [f"file missing: {path}"]
    with open(path, encoding="utf-8") as fh:
        r = yaml.safe_load(fh)
    if r.get("title") != title:
        fails.append(f"title {r.get('title')!r} != {title!r}")
    if biome is not None and r.get("biome") != biome:
        fails.append(f"biome {r.get('biome')!r} != {biome!r}")
    c = r.get("coord") or {}
    if (c.get("x"), c.get("y"), c.get("z")) != coord:
        fails.append(f"coord {(c.get('x'),c.get('y'),c.get('z'))} != {coord}")
    got = {d: (v or {}).get("roomid") for d, v in (r.get("exits") or {}).items()}
    if got != exits:
        fails.append(f"exits {got} != {exits}")
    muts = {m.get("mutatorid") for m in (r.get("mutators") or [])}
    if "sanctuary" not in muts:
        fails.append("missing sanctuary mutator")
    nouns = r.get("nouns") or {}
    if len(nouns) < min_nouns:
        fails.append(f"noun count {len(nouns)} < {min_nouns}")
    desc = (r.get("description") or "").lower()
    for key in nouns:
        if any(ch.isspace() for ch in key):
            fails.append(f"noun key {key!r} is not a single token")
        if key.lower() not in desc:
            fails.append(f"noun key {key!r} absent from description body")
    return fails


def check_npc(mid, spec):
    fname, name, host_room = spec
    path = os.path.join(MOBS_DIR, fname)
    fails = []
    if not os.path.exists(path):
        return [f"file missing: {path}"]
    with open(path, encoding="utf-8") as fh:
        m = yaml.safe_load(fh)
    if m.get("mobid") != mid:
        fails.append(f"mobid {m.get('mobid')!r} != {mid}")
    if m.get("zone") != "Pothole Coulee":
        fails.append(f"zone {m.get('zone')!r} != 'Pothole Coulee'")
    if m.get("non_combatant") is not True:
        fails.append(f"non_combatant {m.get('non_combatant')!r} != True")
    if m.get("charm_immune") is not True:
        fails.append(f"charm_immune {m.get('charm_immune')!r} != True")
    if m.get("maxwander") != 0:
        fails.append(f"maxwander {m.get('maxwander')!r} != 0")
    groups = set(m.get("groups") or [])
    for g in ("humanoid", "coulee_folk"):
        if g not in groups:
            fails.append(f"groups missing {g!r}")
    char = m.get("character") or {}
    if char.get("name") != name:
        fails.append(f"name {char.get('name')!r} != {name!r}")

    # Host room spawninfo must list this mobid.
    rpath = os.path.join(ROOMS_DIR, f"{host_room}.yaml")
    if not os.path.exists(rpath):
        fails.append(f"host room file missing: {rpath}")
    else:
        with open(rpath, encoding="utf-8") as fh:
            r = yaml.safe_load(fh)
        spawn_ids = {s.get("mobid") for s in (r.get("spawninfo") or [])}
        if mid not in spawn_ids:
            fails.append(f"host room {host_room} spawninfo {spawn_ids} missing mobid {mid}")

    # Shopkeeper must carry a non-empty shop with at least one itemid entry.
    if mid == SHOPKEEPER_MOBID:
        shop = char.get("shop") or []
        item_entries = [s for s in shop if isinstance(s, dict) and s.get("itemid") is not None]
        if not item_entries:
            fails.append(f"shop empty or has no itemid entries ({len(shop)} rows)")

    # No digits anywhere in the description body (player-facing immersion rule).
    desc = char.get("description") or ""
    if any(ch.isdigit() for ch in desc):
        bad = sorted({ch for ch in desc if ch.isdigit()})
        fails.append(f"description contains digit(s): {bad}")
    return fails


def main():
    print(f"{'ROOM':<6} {'RESULT':<6} DETAIL")
    print("-" * 70)
    total_fail = 0
    for rid in sorted(MANIFEST):
        fails = check_room(rid, MANIFEST[rid])
        if fails:
            total_fail += 1
            print(f"{rid:<6} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{rid:<6} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(MANIFEST)} rooms checked, {total_fail} FAIL")

    print()
    print(f"{'NPC':<6} {'RESULT':<6} DETAIL")
    print("-" * 70)
    npc_fail = 0
    for mid in sorted(NPC_MANIFEST):
        fails = check_npc(mid, NPC_MANIFEST[mid])
        if fails:
            npc_fail += 1
            print(f"{mid:<6} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{mid:<6} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(NPC_MANIFEST)} NPCs checked, {npc_fail} FAIL")

    return 1 if (total_fail or npc_fail) else 0


if __name__ == "__main__":
    sys.exit(main())
