"""
DOGMud Helpfile Coverage Tool

Cross-references registered commands and keywords against available help
template files to identify gaps in documentation coverage.

Usage:
    python tools/help_coverage.py
    just help-coverage
"""

import os
import re
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Paths (relative to project root)
# ---------------------------------------------------------------------------

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent

HELP_TEMPLATE_DIR = PROJECT_ROOT / "_datafiles" / "world" / "dogmud" / "templates" / "help"
KEYWORDS_FILE = PROJECT_ROOT / "_datafiles" / "world" / "dogmud" / "keywords.yaml"
USERCOMMANDS_FILE = PROJECT_ROOT / "internal" / "usercommands" / "usercommands.go"


def get_help_templates():
    """Return set of help topic names from template files on disk."""
    topics = set()
    if not HELP_TEMPLATE_DIR.exists():
        print(f"WARNING: Help template dir not found: {HELP_TEMPLATE_DIR}")
        return topics
    for f in HELP_TEMPLATE_DIR.iterdir():
        if f.is_file() and f.suffix == ".template":
            topics.add(f.stem)
    return topics


def get_keywords_topics():
    """Parse keywords.yaml to extract all registered help topics."""
    topics = set()
    if not KEYWORDS_FILE.exists():
        print(f"WARNING: Keywords file not found: {KEYWORDS_FILE}")
        return topics

    text = KEYWORDS_FILE.read_text(encoding="utf-8")

    # We're in the "help:" block. Extract all list items (- topic) under
    # help.command.*, help.skill.*, help.admin.*
    in_help = False
    for line in text.splitlines():
        stripped = line.strip()

        # Track when we enter/leave the help block
        if line.startswith("help:"):
            in_help = True
            continue
        if in_help and not line.startswith(" ") and not line.startswith("\t") and stripped:
            # We've left the help block (new top-level key)
            if not stripped.startswith("#"):
                in_help = False
                continue

        if in_help and stripped.startswith("- "):
            topic = stripped[2:].strip()
            if topic:
                topics.add(topic)

    return topics


def get_registered_commands():
    """Parse usercommands.go to extract all registered command names."""
    commands = {}
    if not USERCOMMANDS_FILE.exists():
        print(f"WARNING: usercommands.go not found: {USERCOMMANDS_FILE}")
        return commands

    text = USERCOMMANDS_FILE.read_text(encoding="utf-8")

    # Match lines like:  `commandname`: {FuncName, true, true, false},
    pattern = re.compile(r'`([^`]+)`\s*:\s*\{(\w+),\s*(true|false),\s*(true|false),\s*(true|false)\}')
    for m in pattern.finditer(text):
        cmd_name = m.group(1)
        is_admin = m.group(5) == "true"
        commands[cmd_name] = {"admin": is_admin}

    return commands


def get_help_aliases():
    """Parse keywords.yaml to extract help aliases (alias -> canonical topic)."""
    aliases = {}
    if not KEYWORDS_FILE.exists():
        return aliases

    text = KEYWORDS_FILE.read_text(encoding="utf-8")

    in_aliases = False
    for line in text.splitlines():
        stripped = line.strip()

        if line.startswith("help-aliases:"):
            in_aliases = True
            continue
        if in_aliases and not line.startswith(" ") and not line.startswith("\t") and stripped:
            if not stripped.startswith("#"):
                in_aliases = False
                continue

        if in_aliases and ":" in stripped and not stripped.startswith("#"):
            parts = stripped.split(":", 1)
            canonical = parts[0].strip().strip("'\"")
            alias_list_str = parts[1].strip()
            # Parse [alias1, alias2, ...]
            alias_matches = re.findall(r"['\"]?([^,\[\]'\"]+)['\"]?", alias_list_str)
            for alias in alias_matches:
                a = alias.strip()
                if a:
                    aliases[a] = canonical

    return aliases


def main():
    templates = get_help_templates()
    keyword_topics = get_keywords_topics()
    commands = get_registered_commands()
    aliases = get_help_aliases()

    # All known topics from keywords.yaml
    all_keyword_topics = keyword_topics.copy()

    # Commands that are internal/special and don't need help files
    internal_commands = {
        "start", "zombieact", "noop", "default", "gearup",
        "print", "printline", "save", "cancel", "target",
        "submit", "suicide", "experience",
    }

    # Mutation commands that are abilities, not general commands
    mutation_commands = {
        "blinding-flash", "blinding-spit", "healing-gel",
        "pacifism-aura", "sonic-shout", "toxic-bite",
    }

    print("=" * 70)
    print("  DOGMud Helpfile Coverage Report")
    print("=" * 70)
    print()

    # --- 1. Commands without help files ---
    missing = []
    for cmd, info in sorted(commands.items()):
        if cmd in internal_commands or cmd in mutation_commands:
            continue
        # Check if this command has a help template (directly or via alias)
        has_help = cmd in templates
        if not has_help and cmd in aliases and aliases[cmd] in templates:
            has_help = True
        if not has_help:
            label = "(admin)" if info["admin"] else ""
            missing.append((cmd, label))

    if missing:
        print(f"COMMANDS WITHOUT HELP FILES ({len(missing)}):")
        print("-" * 50)
        for cmd, label in missing:
            print(f"  {cmd:30s} {label}")
        print()
    else:
        print("All registered commands have help files!")
        print()

    # --- 2. Keyword topics without template files ---
    keyword_missing = []
    for topic in sorted(all_keyword_topics):
        if topic in templates:
            continue
        # Check if this topic is an alias that resolves to a template
        if topic in aliases and aliases[topic] in templates:
            continue
        keyword_missing.append(topic)

    if keyword_missing:
        print(f"KEYWORD TOPICS WITHOUT TEMPLATE FILES ({len(keyword_missing)}):")
        print("-" * 50)
        for topic in keyword_missing:
            print(f"  {topic}")
        print()

    # --- 3. Orphan template files (not in keywords or commands) ---
    all_known = set()
    all_known.update(all_keyword_topics)
    all_known.update(commands.keys())
    all_known.update(aliases.keys())
    # Also add canonical targets of aliases
    all_known.update(aliases.values())

    orphans = []
    for t in sorted(templates):
        if t not in all_known:
            orphans.append(t)

    if orphans:
        print(f"ORPHAN HELP FILES (not linked to any command/keyword) ({len(orphans)}):")
        print("-" * 50)
        for t in orphans:
            print(f"  {t}.template")
        print()

    # --- 4. Summary ---
    total_commands = len(commands) - len(internal_commands) - len(mutation_commands)
    covered_commands = total_commands - len(missing)
    coverage_pct = (covered_commands / total_commands * 100) if total_commands > 0 else 0

    print("=" * 70)
    print("  SUMMARY")
    print("=" * 70)
    print(f"  Help template files:         {len(templates)}")
    print(f"  Keyword topics registered:   {len(all_keyword_topics)}")
    print(f"  User commands registered:    {len(commands)}")
    print(f"  Internal/special commands:   {len(internal_commands) + len(mutation_commands)}")
    print(f"  Reportable commands:         {total_commands}")
    print(f"  Commands with help:          {covered_commands}")
    print(f"  Commands missing help:       {len(missing)}")
    print(f"  Keyword topics missing file: {len(keyword_missing)}")
    print(f"  Orphan help files:           {len(orphans)}")
    print(f"  Command help coverage:       {coverage_pct:.1f}%")
    print()


if __name__ == "__main__":
    main()
