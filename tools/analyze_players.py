import glob, yaml

COMBAT = {"weapon-combat": "colossus", "unarmed-combat": "ravener", "ranged-combat": "stalker",
          "skullduggery": "stalker", "rhetoric": "zealot"}
CRAFT = ["alchemy", "blacksmithing", "tailoring", "cooking", "jewelcrafting", "enchanting", "salvage"]

for f in sorted(glob.glob("_archive/prod-users/users/*.yaml")):
    d = yaml.safe_load(open(f, encoding="utf-8"))
    ch = d.get("character", d)
    sk = ch.get("skills", {}) or {}
    comps = ch.get("companions", []) or []
    spellbook = ch.get("spellbook", {}) or {}
    wil = (((ch.get("stats") or {}).get("willpower") or {}).get("training")) or 0
    role = d.get("role", "")  # role is a TOP-LEVEL UserRecord field, not under character
    topcombat = max(((v, COMBAT[k]) for k, v in sk.items() if k in COMBAT), default=(0, ""))
    craftsum = sum(sk.get(c, 0) for c in CRAFT)
    import os
    print(f"{os.path.basename(f):9} {ch.get('name','?'):16} role={role:6} comps={len(comps)} "
          f"manif={sk.get('manifestation',0):3} spellbook={len(spellbook):2} wilTrain={wil:3} "
          f"topcombat={topcombat[0]:3}->{topcombat[1]:9} craftsum={craftsum}")
