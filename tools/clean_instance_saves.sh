#!/bin/bash
# clean_instance_saves.sh
# Run after killing the server to delete stale instance saves that would
# override freshly edited room templates.
#
# Usage: bash tools/clean_instance_saves.sh [zone_folder ...]
#   No args  = clean ALL instance saves across every zone
#   With args = clean only the named zone folders
#
# Examples:
#   bash tools/clean_instance_saves.sh
#   bash tools/clean_instance_saves.sh labyrinth_of_low_tunnels sanctum_basin

INSTANCES_DIR="_datafiles/world/dogmud/rooms.instances"

# Find the repo root (script may be called from anywhere)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TARGET="$REPO_ROOT/$INSTANCES_DIR"

if [ ! -d "$TARGET" ]; then
    echo "Instance saves directory not found: $TARGET"
    exit 1
fi

deleted=0

if [ $# -eq 0 ]; then
    # Clean all zones
    echo "Cleaning ALL instance saves in $INSTANCES_DIR/"
    for f in "$TARGET"/*/*.yaml; do
        [ -f "$f" ] || continue
        echo "  Deleting: ${f#$REPO_ROOT/}"
        rm "$f"
        ((deleted++))
    done
else
    # Clean only named zones
    for zone in "$@"; do
        zone_dir="$TARGET/$zone"
        if [ ! -d "$zone_dir" ]; then
            echo "Zone folder not found: $INSTANCES_DIR/$zone/ (skipping)"
            continue
        fi
        echo "Cleaning instance saves for zone: $zone"
        for f in "$zone_dir"/*.yaml; do
            [ -f "$f" ] || continue
            echo "  Deleting: ${f#$REPO_ROOT/}"
            rm "$f"
            ((deleted++))
        done
    done
fi

if [ $deleted -eq 0 ]; then
    echo "No instance saves found. Nothing to clean."
else
    echo "Deleted $deleted instance save(s). Safe to restart the server."
fi
