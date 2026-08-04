#!/usr/bin/env bash
#
# Slow down archify's guided-story autoplay.
#
# WHY THIS EXISTS
# ---------------
# The diagrams under _datafiles/html/public/diagrams/ each carry a guided
# "story" of up to five chapters. Archify's stock viewer advances those beats
# from:
#
#     storyBeatDwell(total) = max(STORY_FOLLOW_MIN_DWELL_MS, VIEW_INTERVAL_MS / total)
#
# With the shipped defaults (1100 and 3200) the floor always wins for a
# three-or-more chapter story, so every beat gets 1.1 seconds and a five-beat
# story is over in five and a half. That is far too fast to read the beat's
# note and look at the part of the diagram it just panned to — reported by the
# user 2026-08-04 as "runs far too fast to follow".
#
# The dwell is baked into the skill's viewer template. It is NOT a per-diagram
# authored field: no schema exposes it, and no CLI flag sets it. So the fix has
# to be applied to the installed skill, and every affected diagram re-delivered
# afterwards to pick up the new template.
#
# Because the skill is installed globally (outside this repo), reinstalling or
# updating archify silently reverts the change and the regression returns with
# no signal. Re-run this script after any archify update, then re-deliver the
# diagrams.
#
# Only STORY_FOLLOW_MIN_DWELL_MS is touched. VIEW_INTERVAL_MS is deliberately
# left alone: it also feeds the progress-bar animation, and the story path
# passes the computed dwell through as that bar's duration, so raising only the
# floor keeps the bar in sync with the beat.
#
# Usage:  bash tools/archify/patch-story-pacing.sh [--check]
#   --check   report status and exit non-zero if unpatched; change nothing

set -euo pipefail

TEMPLATE="${ARCHIFY_TEMPLATE:-$HOME/.agents/skills/archify/assets/template.html}"
WANT=5500
STOCK=1100

# Both of archify's autoplay features ship the same too-fast 1100ms dwell:
#   STORY_FOLLOW_MIN_DWELL_MS - guided chapter story ("play story" / P)
#   JOURNEY_DWELL_MS          - node-by-node journey walkthrough
CONSTANTS="STORY_FOLLOW_MIN_DWELL_MS JOURNEY_DWELL_MS"

if [ ! -f "$TEMPLATE" ]; then
  echo "ERROR: archify template not found at: $TEMPLATE" >&2
  echo "Set ARCHIFY_TEMPLATE if the skill lives elsewhere." >&2
  exit 2
fi

read_const() {
  grep -oE "var $1 = [0-9]+" "$TEMPLATE" | grep -oE '[0-9]+$' || true
}

# --check: report only, non-zero if anything is unpatched.
if [ "${1:-}" = "--check" ]; then
  rc=0
  for name in $CONSTANTS; do
    current="$(read_const "$name")"
    if [ -z "$current" ]; then
      echo "MISSING: $name not found in $TEMPLATE" >&2
      rc=3
    elif [ "$current" = "$WANT" ]; then
      echo "OK: $name is ${current}ms (patched)."
    else
      echo "UNPATCHED: $name is ${current}ms, expected ${WANT}ms." >&2
      [ "$rc" = 0 ] && rc=1
    fi
  done
  if [ "$rc" != 0 ]; then
    echo "Run: bash tools/archify/patch-story-pacing.sh" >&2
    echo "Then re-deliver every diagram so the new template is baked in." >&2
  fi
  exit "$rc"
fi

changed=0
for name in $CONSTANTS; do
  current="$(read_const "$name")"

  if [ -z "$current" ]; then
    echo "ERROR: $name not found in $TEMPLATE" >&2
    echo "Archify's viewer template has changed shape. Re-derive this patch by" >&2
    echo "hand before trusting autoplay pacing again." >&2
    exit 3
  fi

  if [ "$current" = "$WANT" ]; then
    echo "Already patched: $name is ${current}ms."
    continue
  fi

  if [ "$current" != "$STOCK" ]; then
    echo "NOTE: $name is ${current}ms, neither the stock ${STOCK}ms nor the" >&2
    echo "wanted ${WANT}ms. Patching anyway, but check whether archify changed" >&2
    echo "its default." >&2
  fi

  sed -i "s/var ${name} = ${current};/var ${name} = ${WANT};/" "$TEMPLATE"

  verify="$(read_const "$name")"
  if [ "$verify" != "$WANT" ]; then
    echo "ERROR: patch of $name did not take; still ${verify}ms." >&2
    exit 4
  fi

  echo "Patched $name ${current}ms -> ${WANT}ms"
  changed=1
done

if [ "$changed" = 0 ]; then
  echo "Nothing to do."
  exit 0
fi

echo
echo "Now re-deliver every diagram so the new template is baked into the"
echo "artifacts -- the delivered HTML embeds the viewer, it does not link to it."
