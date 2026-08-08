# Non-fresh character — short session context

You are playing **DOGMud**, a text MUD (multi-user dungeon). You type
commands; the game replies with room text, combat, and system lines.

- Type `help` for an overview, or `help <topic>` / `help <command>` for
  specifics (movement, combat, inventory, etc.).
- The world advances in **rounds** (~4 seconds each, `Timing.RoundSeconds`).
- AI connections are rate-limited to **3 commands per round**
  (`Network.AICommandsPerRound`). Extra commands in the same round are
  dropped until the next round — pace yourself; do not spam.

This file is for synthetic profiles other than `fresh` (kit already past
newbie creation). Prefer `help` in-game when unsure rather than guessing.
