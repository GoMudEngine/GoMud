#!/usr/bin/env python3
"""Flag existing AI/bot accounts so leaderboards exclude them.

Going forward, characters that enter the world on the AI port are auto-flagged
(internal/hooks/PlayerSpawn_HandleJoin.go). This one-time helper cleans up
EXISTING accounts that played as bots before that fix and never got the
persisted `isai: true` flag — the ones still polluting the leaderboard.

SAFE BY DEFAULT: dry-run only. It never writes without --apply, and it only
touches accounts whose username matches a known bot/test pattern AND that are
not already admin/AI-flagged. Review the dry-run, then re-run with --apply.

Usage:
  python tools/flag_bot_accounts.py [--users-dir DIR]            # dry-run (default)
  python tools/flag_bot_accounts.py --apply                     # flag pattern matches
  python tools/flag_bot_accounts.py --apply --only q4,tester7   # flag a reviewed list

Default users dir: _datafiles/world/dogmud/users (the real save location).
On prod, point --users-dir at the droplet's users directory.
"""
import argparse, glob, os, re, sys

BOT_PATTERNS = [
    r"quester\d*", r"tester\d*", r"smoketest\w*", r"polishtest\d*", r"p2smoke",
    r"explorer\d+", r"shopper\d+", r"newbie\d+", r"aitester", r"bugtest\w*",
    r"feeltest\w*", r"feeler\w*", r"feature.?test\w*", r"bot\w*", r"npc.?test\w*",
    r"verify\d*", r"tier[a-z]\d*",
]
BOT_RE = re.compile(r"^(?:%s)$" % "|".join(BOT_PATTERNS), re.IGNORECASE)


def field(text, key):
    m = re.search(r"(?m)^%s:\s*(.*)$" % re.escape(key), text)
    return m.group(1).strip() if m else None


def set_isai_true(path, text):
    """Return updated text with isai: true (add under username if absent)."""
    if re.search(r"(?m)^isai:", text):
        return re.sub(r"(?m)^isai:.*$", "isai: true", text)
    # insert after the username line (top-level field)
    return re.sub(r"(?m)^(username:.*)$", r"\1\nisai: true", text, count=1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--users-dir", default="_datafiles/world/dogmud/users")
    ap.add_argument("--apply", action="store_true", help="write the flag (default: dry-run)")
    ap.add_argument("--only", default="", help="comma list of usernames to flag (overrides patterns)")
    args = ap.parse_args()

    only = {u.strip().lower() for u in args.only.split(",") if u.strip()}
    flagged, matched, already, other = [], [], [], []

    for path in sorted(glob.glob(os.path.join(args.users_dir, "*.yaml"))):
        if os.path.basename(path) == "users.idx":
            continue
        with open(path, encoding="utf-8") as fh:
            text = fh.read()
        uname = field(text, "username") or os.path.basename(path)
        role = (field(text, "role") or "user").strip()
        isai = (field(text, "isai") or "").strip().lower()
        if role == "admin" or isai == "true":
            already.append(uname); continue
        is_bot = (uname.lower() in only) if only else bool(BOT_RE.match(uname))
        if not is_bot:
            other.append(uname); continue
        matched.append(uname)
        if args.apply:
            with open(path, "w", encoding="utf-8", newline="") as fh:
                fh.write(set_isai_true(path, text))
            flagged.append(uname)

    print(f"users dir: {args.users_dir}")
    print(f"already excluded (admin or isai:true): {len(already)}")
    print(f"non-bot unflagged players (LEFT ALONE): {len(other)} -> {sorted(other)}")
    print(f"bot/test accounts matched: {len(matched)} -> {sorted(matched)}")
    if args.apply:
        print(f"\nAPPLIED isai:true to {len(flagged)} accounts.")
    else:
        print("\nDRY-RUN. Re-run with --apply to flag the matched accounts, "
              "or --apply --only <names> for a reviewed subset.")


if __name__ == "__main__":
    main()
