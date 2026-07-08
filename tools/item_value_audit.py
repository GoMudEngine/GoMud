#!/usr/bin/env python3
"""Item value audit (Stage 4: shops-trade-affixed-gear catalog audit).

Computes a stat-implied gold value for each GEAR item using the affix
cost-point rubric (mitigation@5, stat@3, skill@12, damage_mult@8 per +0.05
over 1.0) x GoldPerAffixPoint, and reports the divergence from the authored
`value`. Also lists premium crafting materials/reagents (value-sorted) for
manual review, since those are market-priced, not stat-driven.

The rubric is a FLOOR/guide for ordinary gear. Pinnacle/legendary craftables
(e.g. the Blackrazor) intentionally sit far ABOVE their stat-implied value —
prestige/best-in-slot premium — so large positive divergence there is expected,
not a bug.

Usage: python tools/item_value_audit.py [--gold-per-point N] [--premium-min N]
"""
import argparse
import glob
import os

import yaml

SKILLS = {"weapon-combat", "unarmed-combat", "skullduggery",
          "spellcasting", "rhetoric", "manifestation"}
GEAR_DIRS = ["weapons-10000", "armor-20000"]
ITEMS_ROOT = "_datafiles/world/dogmud/items"


def affix_points(spec):
    """Total affix cost-points implied by an item's stat-bearing fields."""
    pts = 0
    pts += (spec.get("physical_mitigation", 0) or 0) * 5
    pts += (spec.get("magical_mitigation", 0) or 0) * 5
    pts += (spec.get("conviction_mitigation", 0) or 0) * 5
    dm = spec.get("damage_multiplier", 0) or 0
    if dm > 1.0:
        pts += round((dm - 1.0) / 0.05) * 8
    sdm = spec.get("spell_damage_multiplier", 0) or 0
    if sdm > 1.0:
        pts += round((sdm - 1.0) / 0.05) * 4
    for k, v in (spec.get("statmods") or {}).items():
        if not isinstance(v, int) or v <= 0:
            continue
        pts += v * (12 if k in SKILLS else 3)
    return pts


def load_specs(subdirs):
    out = []
    for sub in subdirs:
        for path in glob.glob(os.path.join(ITEMS_ROOT, sub, "**", "*.yaml"), recursive=True):
            try:
                with open(path, "r", encoding="utf-8") as f:
                    spec = yaml.safe_load(f)
            except Exception as e:
                print(f"  !! parse error {path}: {e}")
                continue
            if isinstance(spec, dict) and "itemid" in spec:
                spec["_path"] = path
                out.append(spec)
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--gold-per-point", type=float, default=3.0)
    ap.add_argument("--premium-min", type=int, default=200)
    args = ap.parse_args()
    gpp = args.gold_per_point

    gear = load_specs(GEAR_DIRS)
    rows = []
    for s in gear:
        implied = int(round(affix_points(s) * gpp))
        authored = s.get("value", 0) or 0
        # ratio: authored / implied (guard div-by-zero). <1 = under-valued.
        ratio = (authored / implied) if implied > 0 else None
        rows.append((s.get("name", "?"), authored, implied, ratio,
                     s.get("type", ""), os.path.basename(s["_path"])))

    print(f"=== GEAR value audit (GoldPerAffixPoint={gpp}) ===")
    print(f"{'item':32} {'authored':>8} {'implied':>8} {'ratio':>6}  type")
    print("-" * 78)
    # Sort by ratio ascending (most under-valued first); None (no stats) last.
    rows.sort(key=lambda r: (r[3] is None, r[3] if r[3] is not None else 0))
    for name, authored, implied, ratio, typ, _ in rows:
        rstr = f"{ratio:.2f}" if ratio is not None else "  -"
        flag = ""
        if ratio is not None and ratio < 0.6:
            flag = "  << UNDER"
        elif ratio is not None and ratio > 2.5:
            flag = "  >> OVER (pinnacle?)"
        print(f"{name[:32]:32} {authored:>8} {implied:>8} {rstr:>6}  {typ}{flag}")

    # Premium materials / reagents — market-priced, listed for manual review.
    mats = load_specs(["materials-40000", "consumables-30000"])
    premium = sorted([s for s in mats if (s.get("value", 0) or 0) >= args.premium_min],
                     key=lambda s: -(s.get("value", 0) or 0))
    print(f"\n=== PREMIUM materials/consumables (value >= {args.premium_min}) — manual review ===")
    for s in premium:
        print(f"{s.get('name','?')[:40]:40} {s.get('value',0):>8}   {os.path.basename(s['_path'])}")


if __name__ == "__main__":
    main()
