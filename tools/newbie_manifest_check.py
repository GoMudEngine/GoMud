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
    # 5220 is the Spoke A (Martial) mouth: the hub stub plus the eastward
    # attachment exit into the spoke's Drill Yard (5227), added in chunk 2.
    5220: ("Dry Coulee Mouth", None, (48, 0, 0), {"west":5209, "east":5227}, 1),
    5221: ("Talus Gap", None, (42, 0, 0), {"east":5210, "west":5244}, 1),
    5222: ("Reedwash Mouth", None, (47, 2, 0), {"west":5207}, 1),
    5223: ("Scrub Draw", None, (43, 1, 0), {"east":5216}, 1),
    5224: ("Stargazer Cut", None, (44, -2, 0), {"south":5218}, 1),
    5225: ("Old Field Track", None, (46, -2, 0), {"south":5217}, 1),
    5226: ("Bluff Steps", None, (47, -1, 0), {"south":5209}, 1),
}


# ---------------------------------------------------------------------------
# Spoke A (Martial) — Pothole Coulee rooms 5227-5243.
# Three concentric rings climbing the wash to the boss tower:
#   inner (sanctuary):   5227-5231
#   middle (NO sanct.):  5232-5237
#   outer  (NO sanct.):  5238-5243
# Each entry: (rid, title, sanctuary_expected). biome only checked non-empty +
# in the engine's known set; title/description only checked non-empty.
SPOKE_A_ROOMS = [
    (5227, "The Drill Yard", True),
    (5228, "Weapon Rack Lean-to", True),
    (5229, "Sparring Circle", True),
    (5230, "Yard Overlook", True),
    (5231, "The Last Safe Step", True),
    (5232, "Lower Wash", False),
    (5233, "Gravel Bend", False),
    (5234, "Squatter's Hollow", False),
    (5235, "Cracked Cistern", False),
    (5236, "Upper Wash", False),
    (5237, "Tower Approach", False),
    (5238, "Broken Gate", False),
    (5239, "Collapsed Barracks", False),
    (5240, "Tower Base", False),
    (5241, "Tower Stair", False),
    (5242, "The Watch Room", False),
    (5243, "Tower Top", False),
]

# Reciprocal exit pairs (a, dir_a_to_b, b). The reverse exit on b must point
# back at a in the opposite direction. dir_pairs below define the opposites.
# Most edges are cardinal; the 5235<->5236 edge is intentionally diagonal
# (NE/SW) because the all-cardinal loop was geometrically unclosable, and the
# tower climb 5240->5241->5242 is vertical (up/down).
SPOKE_A_EXIT_PAIRS = [
    (5220, "east", 5227),
    (5227, "east", 5228),
    (5227, "north", 5229),
    (5229, "east", 5230),
    (5230, "east", 5231),
    (5231, "east", 5232),
    (5232, "east", 5233),
    (5232, "south", 5234),
    (5234, "east", 5235),
    (5235, "northeast", 5236),   # diagonal-by-design edge
    (5233, "east", 5236),
    (5236, "east", 5237),
    (5237, "east", 5238),
    (5238, "east", 5239),
    (5238, "north", 5240),
    (5240, "up", 5241),          # vertical climb
    (5241, "up", 5242),          # vertical climb
    (5242, "east", 5243),
]

OPPOSITE_DIR = {
    "north": "south", "south": "north",
    "east": "west", "west": "east",
    "northeast": "southwest", "southwest": "northeast",
    "northwest": "southeast", "southeast": "northwest",
    "up": "down", "down": "up",
}

# ---------------------------------------------------------------------------
# Spoke B (Forge) — Pothole Coulee rooms 5244-5261. Climbs WEST from hub stub
# 5221. Three rings: inner smithy (sanctuary, 5244-5248), middle talus slope
# (NO sanct., 5249-5254), outer mine shaft (NO sanct., descends z-1/z-2,
# 5255-5261). Same (rid, title, sanctuary_expected) shape as Spoke A.
SPOKE_B_ROOMS = [
    (5244, "Forge Path", True),
    (5245, "The Coulee Smithy", True),
    (5246, "Ore Stall", True),
    (5247, "Quench Shed", True),
    (5248, "The Last Worked Stone", True),
    (5249, "Lower Talus", False),
    (5250, "Scree Field", False),
    (5251, "Collapsed Cut", False),
    (5252, "Ore Pocket", False),
    (5253, "Upper Talus", False),
    (5254, "Mine Mouth", False),
    (5255, "Mine Head", False),
    (5256, "Timbered Drift", False),
    (5257, "Flooded Sump", False),
    (5258, "Lower Drift", False),
    (5259, "The Stone Gallery", False),
    (5260, "The Den", False),
    (5261, "Deep Vein", False),
]

# Reciprocal exit edges (each listed once; the checker verifies both
# directions). The mine descent 5254->5255 and 5256->5258 is vertical
# (down/up); everything else is cardinal (the layout is all-cartesian).
SPOKE_B_EXIT_PAIRS = [
    (5221, "west", 5244),   # hub stub attachment
    (5244, "west", 5245),
    (5245, "west", 5246),
    (5245, "north", 5247),
    (5247, "west", 5248),
    (5248, "west", 5249),
    (5249, "west", 5250),
    (5249, "south", 5251),
    (5250, "west", 5253),
    (5250, "south", 5252),
    (5251, "west", 5252),
    (5253, "west", 5254),
    (5254, "down", 5255),   # descent into the mine
    (5255, "west", 5256),
    (5256, "west", 5257),
    (5256, "down", 5258),   # descent to lower workings
    (5258, "west", 5259),
    (5259, "west", 5260),
    (5260, "west", 5261),
]

# Spoke B (Forge) mobs 9116-9123 (Phase M). Same shape as SPOKE_A_MOBS:
#   mobid: (filename, name, [host_rooms], hostile, behavior_archetype, statpool)
# 9116 Smith Rusk is the shopkeeper/crafter (craft_support blacksmithing).
# The 6 foes are cave creatures (NOT humanoid -> no Opened requirement).
SPOKE_B_MOBS = {
    9116: ("9116-smith_rusk.yaml",            "Smith Rusk",           [5245],             False, "noncombat_questgiver", 44),
    9117: ("9117-survivor_ovell.yaml",        "Survivor Ovell",       [5251],             False, "noncombat_questgiver", 44),
    9118: ("9118-scree_scavenger.yaml",       "Scree Scavenger",      [5249, 5250, 5253], True,  "generic_fighter",      18),
    9119: ("9119-talus_lurker.yaml",          "Talus Lurker",         [5252, 5253],       True,  "generic_fighter",      40),
    9120: ("9120-mine_crawler.yaml",          "Mine Crawler",         [5255, 5256],       True,  "generic_fighter",      60),
    9121: ("9121-tunnel_brute.yaml",          "Tunnel Brute",         [5256, 5257],       True,  "generic_fighter",      90),
    9122: ("9122-stone_crusted_lurker.yaml",  "Stone-Crusted Lurker", [5258],             True,  "generic_fighter",     130),
    9123: ("9123-stone_blooded_beast.yaml",   "Stone-Blooded Beast",  [5260],             True,  "tank_taunter",        200),
}

# ---------------------------------------------------------------------------
# Spoke A (Martial) — mobs 9108-9115 (Phase M). Per mob:
#   mobid: (filename, name, [host_rooms], hostile, behavior_archetype, statpool)
# Every mob must exist with the declared fields AND be listed in the
# spawninfo of EACH of its declared host rooms. Descriptions carry no digits
# (player-facing immersion rule). The two questgivers (Vorn, Garve) are
# non_combatant; the dummy is combat_passive (attackable) but NOT hostile;
# the rest are hostile fighters scaling inner->outer ring.
SPOKE_A_MOBS = {
    9108: ("9108-drillmaster_vorn.yaml",      "Drillmaster Vorn",    [5227],             False, "noncombat_questgiver", 44),
    9109: ("9109-training_dummy.yaml",        "Training Dummy",      [5227, 5228],       False, "combat_passive",        3),
    9110: ("9110-bandit_scout.yaml",          "Bandit Scout",        [5232, 5233, 5236, 5238], True, "generic_fighter",  18),
    9111: ("9111-bandit_squatter.yaml",       "Bandit Squatter",     [5235, 5236],       True,  "generic_fighter",      40),
    9112: ("9112-caravan_guard_garve.yaml",   "Caravan Guard Garve", [5234],             False, "noncombat_questgiver", 44),
    9113: ("9113-bandit_bruiser.yaml",        "Bandit Bruiser",      [5238, 5239],       True,  "generic_fighter",      90),
    9114: ("9114-bandit_lieutenant.yaml",     "Bandit Lieutenant",   [5239],             True,  "leader",              130),
    9115: ("9115-bandit_captain.yaml",        "Bandit Captain",      [5242],             True,  "tank_taunter",        200),
}


def check_spoke_a_mob(mid, spec):
    fname, name, host_rooms, hostile, archetype, statpool = spec
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
    if m.get("behavior_archetype") != archetype:
        fails.append(f"behavior_archetype {m.get('behavior_archetype')!r} != {archetype!r}")
    if bool(m.get("hostile")) != hostile:
        fails.append(f"hostile {m.get('hostile')!r} != {hostile}")
    if m.get("statpool") != statpool:
        fails.append(f"statpool {m.get('statpool')!r} != {statpool}")
    char = m.get("character") or {}
    if char.get("name") != name:
        fails.append(f"name {char.get('name')!r} != {name!r}")
    # No digits in the description body (immersion rule).
    desc = char.get("description") or ""
    if any(ch.isdigit() for ch in desc):
        bad = sorted({ch for ch in desc if ch.isdigit()})
        fails.append(f"description contains digit(s): {bad}")
    # Must be spawned by EACH declared host room.
    for hr in host_rooms:
        r, rpath = _load_room(hr)
        if r is None:
            fails.append(f"host room file missing: {rpath}")
            continue
        spawn_ids = {s.get("mobid") for s in (r.get("spawninfo") or [])}
        if mid not in spawn_ids:
            fails.append(f"host room {hr} spawninfo {spawn_ids} missing mobid {mid}")
    return fails

# Engine biome registry (internal/rooms biomes). Spoke A only uses a subset,
# but assert against the full known set so a typo'd biome fails.
KNOWN_BIOMES = {
    "water", "shore", "city", "house", "cliffs", "fort", "cave", "forest",
    "swamp", "mountains", "desert", "plains", "underground", "road",
}


def _load_room(rid):
    path = os.path.join(ROOMS_DIR, f"{rid}.yaml")
    if not os.path.exists(path):
        return None, path
    with open(path, encoding="utf-8") as fh:
        return yaml.safe_load(fh), path


def check_spoke_a_room(rid, title, sanctuary_expected):
    r, path = _load_room(rid)
    fails = []
    if r is None:
        return [f"file missing: {path}"]
    if r.get("roomid") != rid:
        fails.append(f"roomid {r.get('roomid')!r} != {rid}")
    if r.get("zone") != "Pothole Coulee":
        fails.append(f"zone {r.get('zone')!r} != 'Pothole Coulee'")
    if r.get("title") != title:
        fails.append(f"title {r.get('title')!r} != {title!r}")
    if not (r.get("description") or "").strip():
        fails.append("empty/missing description")
    biome = r.get("biome")
    if not biome:
        fails.append("empty/missing biome")
    elif biome not in KNOWN_BIOMES:
        fails.append(f"biome {biome!r} not a known biome")
    muts = {m.get("mutatorid") for m in (r.get("mutators") or [])}
    has_sanct = "sanctuary" in muts
    if sanctuary_expected and not has_sanct:
        fails.append("sanctuary mutator expected but ABSENT")
    if not sanctuary_expected and has_sanct:
        fails.append("sanctuary mutator present but should be ABSENT")
    # Noun discoverability: every noun key is a single token AND appears
    # (case-insensitive) somewhere in the description body — same rule the
    # ch.1 hub check enforces.
    desc = (r.get("description") or "").lower()
    for key in (r.get("nouns") or {}):
        if any(ch.isspace() for ch in key):
            fails.append(f"noun key {key!r} is not a single token")
        if key.lower() not in desc:
            fails.append(f"noun key {key!r} absent from description body")
    return fails


def check_spoke_a_exit_pair(a, dir_ab, b):
    """Confirm a has exit dir_ab->b AND b has the reverse exit back to a."""
    fails = []
    ra, pa = _load_room(a)
    rb, pb = _load_room(b)
    if ra is None:
        return [f"file missing: {pa}"]
    if rb is None:
        return [f"file missing: {pb}"]
    rev = OPPOSITE_DIR[dir_ab]
    a_exits = {d: (v or {}).get("roomid") for d, v in (ra.get("exits") or {}).items()}
    b_exits = {d: (v or {}).get("roomid") for d, v in (rb.get("exits") or {}).items()}
    if a_exits.get(dir_ab) != b:
        fails.append(f"{a}.{dir_ab} -> {a_exits.get(dir_ab)!r} (expected {b})")
    if b_exits.get(rev) != a:
        fails.append(f"{b}.{rev} -> {b_exits.get(rev)!r} (expected {a}; non-reciprocal)")
    return fails


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

    # --- Spoke A (Martial) rooms -------------------------------------------
    print()
    print(f"{'SPOKE-A':<8} {'RESULT':<6} DETAIL")
    print("-" * 70)
    spoke_room_fail = 0
    for rid, title, sanct in SPOKE_A_ROOMS:
        fails = check_spoke_a_room(rid, title, sanct)
        if fails:
            spoke_room_fail += 1
            print(f"{rid:<8} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{rid:<8} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_A_ROOMS)} Spoke A rooms checked, {spoke_room_fail} FAIL")

    print()
    print(f"{'EXITPAIR':<14} {'RESULT':<6} DETAIL")
    print("-" * 70)
    pair_fail = 0
    for a, dir_ab, b in SPOKE_A_EXIT_PAIRS:
        fails = check_spoke_a_exit_pair(a, dir_ab, b)
        label = f"{a}-{dir_ab[:2]}-{b}"
        if fails:
            pair_fail += 1
            print(f"{label:<14} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{label:<14} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_A_EXIT_PAIRS)} Spoke A exit pairs checked, {pair_fail} FAIL")

    # --- Spoke A (Martial) mobs ---------------------------------------------
    print()
    print(f"{'SPOKE-A MOB':<14} {'RESULT':<6} DETAIL")
    print("-" * 70)
    spoke_mob_fail = 0
    for mid in sorted(SPOKE_A_MOBS):
        fails = check_spoke_a_mob(mid, SPOKE_A_MOBS[mid])
        if fails:
            spoke_mob_fail += 1
            print(f"{mid:<14} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{mid:<14} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_A_MOBS)} Spoke A mobs checked, {spoke_mob_fail} FAIL")

    # --- Spoke B (Forge) rooms ----------------------------------------------
    print()
    print(f"{'SPOKE-B':<8} {'RESULT':<6} DETAIL")
    print("-" * 70)
    spoke_b_room_fail = 0
    for rid, title, sanct in SPOKE_B_ROOMS:
        fails = check_spoke_a_room(rid, title, sanct)  # generic room check
        if fails:
            spoke_b_room_fail += 1
            print(f"{rid:<8} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{rid:<8} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_B_ROOMS)} Spoke B rooms checked, {spoke_b_room_fail} FAIL")

    print()
    print(f"{'EXITPAIR-B':<14} {'RESULT':<6} DETAIL")
    print("-" * 70)
    pair_b_fail = 0
    for a, dir_ab, b in SPOKE_B_EXIT_PAIRS:
        fails = check_spoke_a_exit_pair(a, dir_ab, b)  # generic exit-pair check
        label = f"{a}-{dir_ab[:2]}-{b}"
        if fails:
            pair_b_fail += 1
            print(f"{label:<14} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{label:<14} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_B_EXIT_PAIRS)} Spoke B exit pairs checked, {pair_b_fail} FAIL")

    # --- Spoke B (Forge) mobs -----------------------------------------------
    print()
    print(f"{'SPOKE-B MOB':<14} {'RESULT':<6} DETAIL")
    print("-" * 70)
    spoke_b_mob_fail = 0
    for mid in sorted(SPOKE_B_MOBS):
        fails = check_spoke_a_mob(mid, SPOKE_B_MOBS[mid])  # generic mob check
        if fails:
            spoke_b_mob_fail += 1
            print(f"{mid:<14} {'FAIL':<6} {'; '.join(fails)}")
        else:
            print(f"{mid:<14} {'PASS':<6}")
    print("-" * 70)
    print(f"{len(SPOKE_B_MOBS)} Spoke B mobs checked, {spoke_b_mob_fail} FAIL")

    return 1 if (total_fail or npc_fail or spoke_room_fail or pair_fail
                 or spoke_mob_fail or spoke_b_room_fail or pair_b_fail
                 or spoke_b_mob_fail) else 0


if __name__ == "__main__":
    sys.exit(main())
