#!/usr/bin/env bash
#
# Slow down archify's two autoplay features: the guided chapter story and the
# node-by-node journey walkthrough.
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
# note and look at the part of the diagram it just panned to. Reported by the
# user 2026-08-04 as "runs far too fast to follow". The journey walkthrough
# ships the same 1100ms dwell under a different constant and had the same
# problem.
#
# Tuning history: 1100 (stock) -> 5500, which the user then found a touch slow,
# -> 3575 (35 percent faster than 5500), the current value.
#
# Neither dwell is a per-diagram authored field: no schema exposes them and no
# CLI flag sets them. So the fix has to be applied to the installed skill, and
# every affected diagram re-delivered afterwards to pick up the new template.
#
# Because the skill is installed globally (outside this repo), reinstalling or
# updating archify silently reverts the change and the regression returns with
# no signal. Re-run this script after any archify update, then re-deliver the
# diagrams.
#
# Only the two dwell floors are touched. VIEW_INTERVAL_MS is deliberately left
# alone: it also feeds the progress-bar animation, and the story path passes the
# computed dwell through as that bar's duration, so raising only the floor keeps
# the bar in sync with the beat.
#
# THEME DEFAULT
# -------------
# This script also forces the viewer's default theme to dark. Stock archify
# falls back to the operating system's prefers-color-scheme, so the diagrams
# open light on a light-mode machine. The user wants dark by default because it
# pairs with the blueprint visual preset (2026-08-04).
#
# The precedence above the fallback is left intact: an explicit ?theme= URL
# parameter still wins, and a reader who toggles the theme still has their
# choice remembered in localStorage. Only the "no preference expressed"
# fallback changes, from following the OS to always dark.
#
# Usage:  bash tools/archify/patch-story-pacing.sh [--check]
#   --check   report status and exit non-zero if unpatched; change nothing

set -euo pipefail

TEMPLATE="${ARCHIFY_TEMPLATE:-$HOME/.agents/skills/archify/assets/template.html}"
WANT=3575
STOCK=1100
# Dwell values this project has set before, newest first. Prevents a spurious
# "archify changed its default" warning when re-tuning.
KNOWN_PRIOR="5500"

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

# The stock theme fallback follows the OS; we want dark.
#
# There are TWO of these, and patching only the first is a real bug: the early
# <head> bootstrap sets data-theme before first paint, then the viewer runtime's
# resolveInitial() recomputes it once the toolbar mounts. Miss the second and a
# light-mode machine gets a flash of dark that reverts to light.
THEME_OS_READ="window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';"

theme_os_reads() {
  grep -cF "$THEME_OS_READ" "$TEMPLATE" || true
}

theme_state() {
  case "$(theme_os_reads)" in
    0) echo patched ;;
    *) echo stock ;;
  esac
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
  case "$(theme_state)" in
    patched) echo "OK: default theme is dark (patched)." ;;
    stock)
      echo "UNPATCHED: default theme still follows the OS preference" \
           "($(theme_os_reads) site(s))." >&2
      [ "$rc" = 0 ] && rc=1
      ;;
  esac
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

  # Values this project has itself set in the past. Seeing one of these is
  # expected on a re-tune and is not worth warning about; seeing anything else
  # means archify changed its own default and the patch should be re-derived.
  case " $STOCK $KNOWN_PRIOR " in
    *" $current "*) ;;
    *)
      echo "NOTE: $name is ${current}ms, which is neither the stock ${STOCK}ms" >&2
      echo "nor a value this script has previously set. Patching anyway, but" >&2
      echo "check whether archify changed its default." >&2
      ;;
  esac

  sed -i "s/var ${name} = ${current};/var ${name} = ${WANT};/" "$TEMPLATE"

  verify="$(read_const "$name")"
  if [ "$verify" != "$WANT" ]; then
    echo "ERROR: patch of $name did not take; still ${verify}ms." >&2
    exit 4
  fi

  echo "Patched $name ${current}ms -> ${WANT}ms"
  changed=1
done

case "$(theme_state)" in
  patched)
    echo "Already patched: default theme is dark."
    ;;
  stock)
    before="$(theme_os_reads)"
    python - "$TEMPLATE" "$THEME_OS_READ" <<'PY'
import io, sys
path, old = sys.argv[1], sys.argv[2]
s = io.open(path, encoding='utf-8').read()
n = s.count(old)
if n == 0:
    sys.exit("theme fallback not found; archify's bootstrap has changed shape")
# Both call sites read `... ? 'light' : 'dark';` as the tail of either an
# assignment or a return, so replacing just the matchMedia expression with the
# literal keeps both statements syntactically intact.
io.open(path, 'w', encoding='utf-8', newline='').write(s.replace(old, "'dark';"))
print("replaced %d OS-preference read(s)" % n)
PY
    if [ "$(theme_state)" != "patched" ]; then
      echo "ERROR: theme patch did not take; $(theme_os_reads) OS read(s) remain." >&2
      exit 4
    fi
    echo "Patched default theme: OS preference -> always dark (${before} site(s))"
    changed=1
    ;;
esac

if [ "$changed" = 0 ]; then
  echo "Nothing to do."
  exit 0
fi

echo
echo "Now re-deliver every diagram so the new template is baked into the"
echo "artifacts -- the delivered HTML embeds the viewer, it does not link to it."
