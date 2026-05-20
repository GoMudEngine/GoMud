#!/usr/bin/env python3
"""Bulk-migrate SendTextLegacy / SendTextVisualLegacy callsites in usercommands/
to the categorized messaging API.

Conservative default mappings:
  *.SendTextLegacy(...)         -> *.SendText(messaging.CategorySystem, ...)
  *.SendTextVisualLegacy(...)   -> *.SendTextVisual(messaging.CategoryMobEmote, ...)

Adds the `messaging` import if any replacement happened in a file. Idempotent —
re-running on a partially-migrated file only fixes the remaining Legacy calls.

Usage:
  python tools/migrate_messaging.py <category_override> file1 [file2 ...]

  category_override: name of the messaging.CategoryX to use for SendTextLegacy
                     (default if not specified). Example: CategoryError.
                     Pass `default` to use CategorySystem / CategoryMobEmote.
"""
import re
import sys
from pathlib import Path

# Regex matches `<receiver>.SendTextLegacy(`. The receiver is one identifier
# chunk (letters/digits/_). The Legacy method may end with `Legacy(` exactly.
TEXT_RE = re.compile(r"(\w+)\.SendTextLegacy\(")
VISUAL_RE = re.compile(r"(\w+)\.SendTextVisualLegacy\(")

IMPORT_PATH = '"github.com/GoMudEngine/GoMud/internal/messaging"'
IMPORT_BLOCK_RE = re.compile(r"^(import \(\n)(.*?)(\n\))", re.DOTALL | re.MULTILINE)


def migrate_text(content: str, category: str) -> tuple[str, int]:
    """Replace .SendTextLegacy( with .SendText(messaging.<category>, ."""
    def repl(m: re.Match) -> str:
        return f"{m.group(1)}.SendText(messaging.{category}, "
    new_content, n = TEXT_RE.subn(repl, content)
    return new_content, n


def migrate_visual(content: str, category: str) -> tuple[str, int]:
    """Replace .SendTextVisualLegacy( with .SendTextVisual(messaging.<category>, ."""
    def repl(m: re.Match) -> str:
        return f"{m.group(1)}.SendTextVisual(messaging.{category}, "
    new_content, n = VISUAL_RE.subn(repl, content)
    return new_content, n


def add_messaging_import(content: str) -> str:
    """Add the messaging package import to a Go file's import block.

    Inserts into the GoMud-internal group (last block of imports, after a
    blank line if present). Falls back to appending to the import block.
    """
    if IMPORT_PATH in content:
        return content
    m = IMPORT_BLOCK_RE.search(content)
    if not m:
        return content
    body = m.group(2)
    lines = body.split("\n")
    new_line = f"\t{IMPORT_PATH}"

    # Find the GoMud-internal group: the contiguous block of "github.com/GoMudEngine/GoMud"
    # imports at the end of the block. Insert alphabetically within it.
    gomud_prefix = '"github.com/GoMudEngine/GoMud/internal/'
    # Find indices of gomud-internal imports
    gomud_indices = [
        i for i, ln in enumerate(lines)
        if gomud_prefix in ln
    ]
    if gomud_indices:
        # Insert alphabetically within the gomud group
        insert_idx = None
        for i in gomud_indices:
            stripped = lines[i].strip()
            if stripped > IMPORT_PATH:
                insert_idx = i
                break
        if insert_idx is None:
            insert_idx = gomud_indices[-1] + 1
        lines.insert(insert_idx, new_line)
    else:
        # Append at end (before closing paren). Find last non-empty line.
        insert_idx = len(lines)
        for i in range(len(lines) - 1, -1, -1):
            if lines[i].strip():
                insert_idx = i + 1
                break
        lines.insert(insert_idx, new_line)
    new_body = "\n".join(lines)
    return content[:m.start(2)] + new_body + content[m.end(2):]


def migrate_file(path: Path, text_cat: str, visual_cat: str) -> tuple[int, int]:
    content = path.read_text(encoding="utf-8")
    new_content, n_text = migrate_text(content, text_cat)
    new_content, n_visual = migrate_visual(new_content, visual_cat)
    if n_text == 0 and n_visual == 0:
        return 0, 0
    new_content = add_messaging_import(new_content)
    path.write_text(new_content, encoding="utf-8")
    return n_text, n_visual


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    text_cat = sys.argv[1]
    visual_cat = sys.argv[2]
    if text_cat == "default":
        text_cat = "CategorySystem"
    if visual_cat == "default":
        visual_cat = "CategoryMobEmote"
    total_t, total_v = 0, 0
    for arg in sys.argv[3:]:
        p = Path(arg)
        if not p.exists():
            print(f"skip (missing): {p}")
            continue
        n_t, n_v = migrate_file(p, text_cat, visual_cat)
        if n_t or n_v:
            print(f"{p}: text={n_t} visual={n_v}")
            total_t += n_t
            total_v += n_v
    print(f"\nTOTAL: text={total_t} visual={total_v}")


if __name__ == "__main__":
    main()
