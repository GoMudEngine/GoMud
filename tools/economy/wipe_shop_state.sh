#!/usr/bin/env bash
# Wipe persisted shop economic state to force re-seed from mob templates.
#
# Use this during the vendor-types-economy-polish deploy to discard the
# legacy shop save files. After wipe + restart, every shop boots fresh
# with starting_gold per the new mob YAMLs (1000g specialists / 5000g
# generals) and a re-seeded Stock list.
#
# This is gitignored runtime state (.gitignore:4 — `_datafiles/**/shops`),
# so there's nothing to commit for the wipe itself. Run this manually
# on the prod droplet before restarting the server.
#
# Players' personal gold and inventories are untouched — only NPC shop
# state (gold drift, current stock levels, last-restock round) is reset.

set -euo pipefail

WORLD_DIR="${1:-_datafiles/world/dogmud}"
TARGET="${WORLD_DIR}/shops"

if [ ! -d "${TARGET}" ]; then
    echo "No shops directory at ${TARGET} — nothing to wipe."
    exit 0
fi

echo "Wiping ${TARGET}/ ..."
rm -rf "${TARGET}"
echo "Done. Restart the server; shops will re-seed from mob templates."
