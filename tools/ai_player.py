"""
DOGMud AI Player/Tester

Connects to the MUD on the AI port, logs in, and uses a local Ollama model
to play the game autonomously. Designed for broad system coverage testing.

Prerequisites:
    pip install telnetlib3 aiohttp

Usage:
    python tools/ai_player.py

    Environment variables (all optional):
        MUD_HOST        default: localhost
        MUD_PORT        default: 55555
        OLLAMA_URL      default: http://localhost:11434/api/chat
        OLLAMA_MODEL    default: gemma3:4b
        AI_USERNAME     default: aitester
        AI_PASSWORD     default: testpass123

First run:
    1. Start the MUD server with AIPort enabled
    2. Run this script — it will auto-create the account if it doesn't exist
    3. After first login, flag the account from an admin session on the human
       port:  ai-flag aitester
"""

import asyncio
import os
import re
import sys
import time
import traceback
from collections import deque
from datetime import datetime

import aiohttp
import telnetlib3

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

MUD_HOST = os.environ.get("MUD_HOST", "localhost")
MUD_PORT = int(os.environ.get("MUD_PORT", "55555"))
OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434/api/chat")
OLLAMA_MODEL = os.environ.get("OLLAMA_MODEL", "gemma3:12b")
AI_USERNAME = os.environ.get("AI_USERNAME", "aitester")
AI_PASSWORD = os.environ.get("AI_PASSWORD", "testpass123")

# How long to wait between commands (seconds). The server enforces 2 cmds/round
# (4s rounds), so 4s is the sweet spot. Going faster just gets throttled.
COMMAND_INTERVAL = 4.5

# Max messages kept in the LLM conversation history (older ones are pruned).
MAX_HISTORY = 30

# Loop detection: if the same command appears this many times in the last N
# commands, inject a "you are looping" nudge.
LOOP_WINDOW = 10
LOOP_THRESHOLD = 3

# How often (in commands) to inject a periodic status/quest check reminder.
PERIODIC_CHECK_INTERVAL = 25

# ---------------------------------------------------------------------------
# System prompt
# ---------------------------------------------------------------------------

SYSTEM_PROMPT = """\
You are an AI play-tester in DOGMud, a text MUD set in a dark fantasy world reshaped by the
Chrysalis plague. Your job is to explore, interact with everything, and report genuine bugs.

== OUTPUT FORMAT ==
Respond with EXACTLY ONE MUD command per message. No commentary, no quotes, no explanations.
Just the raw command text. If you want to speak in-game, use "say".

== ESSENTIAL COMMANDS ==
Movement:     north, south, east, west, up, down  (and ne, nw, se, sw)
Look:         look, look <thing>, look <direction>
Interaction:  talk <npc>, ask <npc> <topic>, say <message>
Combat:       attack <target>, cast <spell>, flee, bash, trip, kick, grapple
Items:        get <item>, drop <item>, inventory, equip <item>, remove <item>
              use <item>, eat <item>, drink <item>
Shops:        list, buy <item>, sell <item>
Info:         status, skills, spells, quests, conditions, cooldowns, map, help <topic>
Crafting:     forage, search, craft
Other:        read <sign>, bug <description>, suggest <description>

== NPC TARGETING — CRITICAL ==
You MUST use the NPC's exact name keyword from the room description. Never guess.

  Room says "Also here: Grukk" → use "grukk" (talk grukk, attack grukk, look grukk)
  Room says "Also here: a cave bat" → use "bat" (attack bat)
  Room says "Also here: Elder Saris" → use "saris" (talk saris, ask saris moons)

WRONG: talk blacksmith, talk shopkeeper, talk elder, attack monster, attack creature
RIGHT: talk korvath, talk brecca, talk saris, attack bat, attack goblin

If you get "not recognized" or "couldn't find" — you used the wrong name. Type "look" to
re-read the room and find the correct keyword. Do NOT file a bug for targeting errors.

== TUTORIAL QUEST CHAIN ==
New characters start in Sanctum Basin with a guided quest chain. Each step happens in a
specific room and is triggered automatically when you enter while holding the right quest
flag. Follow this sequence:

  1. Starting area (room 113) — enter to begin. Gives you the first quest flags.
  2. Market Street (room 108) — introduces shopping. Walk there after step 1.
  3. Training Ground (room 114) — introduces combat. Fight the training dummy until it
     "shatters." Use "attack dummy" to fight. After it breaks, you advance.
  4. Smithy (room 109) — introduces crafting. Watch the scripted event.
  5. Workshop (room 111) — introduces alchemy. Watch the scripted event.
  6. Ranger's Ledge (room 106) — introduces wilderness skills. Enter to advance.
  7. Observatory (room 116) — Elder Saris explains spellcasting. Enter to advance.
  8. Cave system (rooms 118→119→120) — the final test. Fight through bats and goblin
     guards to reach the Aberrant Chrysalis boss in room 120. Defeat it. The caves are
     lit by bioluminescent lichen — you can see without any light spell.
  9. South Gate (room 102) — talk to the Warden to complete the tutorial.

IMPORTANT: Quest steps trigger on room entry. If nothing happens when you enter a room,
you probably skipped a step. Run "quests" to check your progress, then go back to the
room for the step you are actually on. The quest flags are sequential — you cannot skip
ahead.

== HOW TO NAVIGATE ==
Rooms show exits in their description and in a compass in the prompt. Read the exits
carefully. If you see "Exits: north, east" then you can go north or east. Do not try
directions that are not listed.

When exploring Sanctum Basin, the general layout is:
- The starting area (113) is central
- Market and shops are nearby to the east/south
- Training ground is to the east
- The observatory is to the northwest (up on the plateau)
- The cave entrance is north of the main area
- The south gate is at room 102

== COMBAT TIPS ==
- Check "status" before fights to know your HP
- Your starting spell is Conviction Spike: "cast mm" to use it in combat
- Use "attack <name>" to engage a mob, then the fight proceeds automatically
- You can use "bash", "trip", "kick", "grapple" as special moves during combat
- If low on health, "flee" to escape, then eat food or wait to regenerate
- New spells are discovered through casting — the more you cast, the more you learn

== TESTING STRATEGY ==
1. FOLLOW THE TUTORIAL FIRST. Complete the quest chain before exploring freely.
2. INTERACT WITH EVERYTHING: Talk to NPCs, look at items, read signs, try shops.
3. TRY COMBAT: Fight mobs using different tactics — melee, spells, special moves.
4. READ HELPFILES: "help" lists topics. "help <topic>" gives details.
5. EXERCISE SYSTEMS: Try crafting (forage, then craft), use shops (list, buy, sell).
6. MONITOR STATE: Check "status", "quests", "conditions" regularly.
7. DO NOT LOOP: If something isn't working after 2-3 tries, try something completely
   different. Move to a new room, try a different command, or check "quests" to
   reorient yourself.

== WHEN TO USE bug vs suggest ==
"bug" = something is BROKEN (exit doesn't work, crash, garbled output, stuck with no exit):
  FORMAT: bug In [room name]: [what happened] vs [what should have happened]

"suggest" = an OPINION or IDEA (thin description, confusing quest, missing feature):
  FORMAT: suggest [area]: [your idea]

DO NOT bug: targeting errors (your mistake), losing fights (normal), not understanding
a command (use "help").

== WORLD CONTEXT ==
Stats (Strength, Dexterity, Perception, Vitality, Willpower, Charisma) are centered at 100.
Combat uses stamina and conviction. Skills and stats improve through use, not leveling.
Spells use "folds" — mental bifurcation of belief. Spells are discovered through practice,
not taught by trainers. The Chrysalis plague causes mutations through sustained combat.
Three moons influence the world.

== PERSONALITY ==
You are curious and methodical. You read room descriptions carefully, use exact NPC names,
and follow quest directions. When stuck, you check "quests" and "look" before trying random
things. You do not repeat failed commands.
"""

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def timestamp():
    return datetime.now().strftime("%H:%M:%S")


def clean_text(text: str) -> str:
    """Strip residual ANSI escapes, telnet noise, and excessive whitespace."""
    # Strip ANSI escape codes (should already be stripped by AI port, but just in case)
    text = re.sub(r'\x1b\[[0-9;]*[a-zA-Z]', '', text)
    # Strip carriage returns
    text = text.replace('\r', '')
    # Collapse multiple blank lines
    text = re.sub(r'\n{3,}', '\n\n', text)
    return text.strip()


def sanitize_command(raw: str) -> str:
    """Extract a single valid MUD command from the LLM response."""
    # Take only the first line
    line = raw.strip().split('\n')[0].strip()
    # Remove surrounding quotes
    line = line.strip('"').strip("'").strip('`')
    # Remove any leading ">" or "Command:" prefixes the LLM might add
    line = re.sub(r'^(>|Command:\s*)', '', line).strip()
    # Safety: cap length
    if len(line) > 200:
        line = line[:200]
    # If empty, fallback
    if not line:
        line = "look"
    return line


def detect_loop(recent_commands: deque) -> str | None:
    """Check if the AI is stuck in a loop. Returns a nudge message or None."""
    if len(recent_commands) < LOOP_WINDOW:
        return None

    # Count command frequencies in the window
    from collections import Counter
    counts = Counter(recent_commands)
    most_common_cmd, most_common_count = counts.most_common(1)[0]

    if most_common_count >= LOOP_THRESHOLD:
        # Check if the last few commands are all the same
        last_3 = list(recent_commands)[-3:]
        if len(set(last_3)) == 1:
            return (
                f"[SYSTEM: You have sent '{last_3[0]}' three times in a row. STOP. "
                f"Try something completely different. Check 'quests' to see what you "
                f"should be doing, or move to a different room, or try 'look' to "
                f"re-read your surroundings.]"
            )

        # More subtle loop: same 2-3 commands alternating
        if most_common_count >= LOOP_THRESHOLD + 2:
            return (
                f"[SYSTEM: You seem stuck in a loop — you have sent '{most_common_cmd}' "
                f"{most_common_count} times in your last {LOOP_WINDOW} commands. Try a "
                f"completely different approach. Move to a new room, check 'quests', or "
                f"try 'help' to find new commands to try.]"
            )

    return None


async def query_ollama(history: list[dict], session: aiohttp.ClientSession) -> str:
    """Send conversation history to Ollama and get a command back."""
    payload = {
        "model": OLLAMA_MODEL,
        "messages": [{"role": "system", "content": SYSTEM_PROMPT}] + history[-MAX_HISTORY:],
        "stream": False,
        "options": {
            "temperature": 0.7,
            "num_predict": 40,  # commands are short
        }
    }
    try:
        async with session.post(OLLAMA_URL, json=payload, timeout=aiohttp.ClientTimeout(total=30)) as resp:
            if resp.status != 200:
                body = await resp.text()
                print(f"  [{timestamp()}] OLLAMA HTTP {resp.status}: {body[:200]}")
                return "look"
            data = await resp.json()
            return data["message"]["content"]
    except asyncio.TimeoutError:
        print(f"  [{timestamp()}] OLLAMA timeout")
        return "look"
    except Exception as e:
        print(f"  [{timestamp()}] OLLAMA error: {e}")
        return "look"


# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------

async def read_until_pause(reader, timeout: float = 2.0) -> str:
    """Read from the MUD until there's a pause in output."""
    chunks = []
    while True:
        try:
            data = await asyncio.wait_for(reader.read(8192), timeout=timeout)
            if not data:
                break
            chunks.append(data)
            # Brief pause to let more data arrive if the server is still sending
            await asyncio.sleep(0.1)
        except asyncio.TimeoutError:
            break
    return "".join(chunks)


async def connect():
    """Open a telnet connection to the MUD AI port. Returns (reader, writer)."""
    reader, writer = await telnetlib3.open_connection(
        MUD_HOST, MUD_PORT,
        encoding='utf-8',
        term='dumb',
        cols=120, rows=50,
    )
    # Read and discard splash screen
    await asyncio.sleep(2)
    splash = await read_until_pause(reader, 3.0)
    splash = clean_text(splash)
    if splash:
        print(f"[{timestamp()}] SPLASH:\n{splash[:500]}\n")
    return reader, writer


async def attempt_login():
    """Try to log in with existing credentials.

    Returns (reader, writer, success). On failure the connection is dead
    (server disconnects on bad login), so caller must reconnect.
    """
    print(f"[{timestamp()}] Attempting login as {AI_USERNAME}...")
    reader, writer = await connect()

    # Send username
    writer.write(AI_USERNAME + "\r\n")
    await asyncio.sleep(1.5)
    resp = clean_text(await read_until_pause(reader, 2.0))
    if resp:
        print(f"[{timestamp()}] LOGIN:\n{resp[:300]}\n")

    # Send password
    writer.write(AI_PASSWORD + "\r\n")
    await asyncio.sleep(3)
    resp = clean_text(await read_until_pause(reader, 3.0))
    if resp:
        print(f"[{timestamp()}] LOGIN RESPONSE:\n{resp[:500]}\n")

    # Check for failure indicators — server sends these then disconnects
    resp_lower = resp.lower()
    if any(kw in resp_lower for kw in ("incorrect", "invalid", "nope", "bye")):
        print(f"[{timestamp()}] Login failed (account may not exist).")
        try:
            writer.close()
        except Exception:
            pass
        return None, None, False

    # If we got no response at all the connection probably died
    if not resp:
        print(f"[{timestamp()}] No response after password — connection lost.")
        try:
            writer.close()
        except Exception:
            pass
        return None, None, False

    return reader, writer, True


async def create_account():
    """Go through the new-account creation flow.

    Returns (reader, writer) logged in and ready to play.
    """
    print(f"[{timestamp()}] Creating new account: {AI_USERNAME}")
    reader, writer = await connect()

    async def send_and_read(text, label, pause=1.5, read_timeout=2.0):
        writer.write(text + "\r\n")
        await asyncio.sleep(pause)
        resp = clean_text(await read_until_pause(reader, read_timeout))
        if resp:
            print(f"[{timestamp()}] {label}:\n{resp[:300]}\n")
        return resp

    # Step 1: send "new" to start account creation
    await send_and_read("new", "NEW_ACCOUNT")

    # Step 2: choose username
    await send_and_read(AI_USERNAME, "SET_USERNAME")

    # Step 3: choose password
    await send_and_read(AI_PASSWORD, "SET_PASSWORD")

    # Step 4: repeat password
    await send_and_read(AI_PASSWORD, "VERIFY_PASSWORD")

    # Step 5: email (optional — send empty)
    await send_and_read("", "EMAIL")

    # Step 6: screen reader prompt
    await send_and_read("n", "SCREEN_READER")

    # Step 7: confirm creation
    resp = await send_and_read("y", "CONFIRM_CREATE", pause=3, read_timeout=3.0)

    print(f"[{timestamp()}] Account creation complete.")
    return reader, writer


async def main():
    print(f"[{timestamp()}] DOGMud AI Player starting")
    print(f"  Host: {MUD_HOST}:{MUD_PORT}")
    print(f"  Model: {OLLAMA_MODEL}")
    print(f"  Account: {AI_USERNAME}")
    print()

    # --- Login / account creation ---
    try:
        reader, writer, logged_in = await attempt_login()
    except Exception as e:
        print(f"  Connection failed: {e}")
        print("  Is the MUD server running with AIPort enabled?")
        sys.exit(1)

    if not logged_in:
        print(f"[{timestamp()}] Account not found — creating from scratch...")
        try:
            reader, writer = await create_account()
        except Exception as e:
            print(f"  Account creation failed: {e}")
            traceback.print_exc()
            sys.exit(1)

    print(f"[{timestamp()}] Login complete. Starting game loop.\n")
    print("=" * 70)

    # --- Game loop ---
    history: list[dict] = []
    recent_commands: deque = deque(maxlen=LOOP_WINDOW)
    commands_sent = 0
    bugs_reported = 0
    suggestions_sent = 0

    async with aiohttp.ClientSession() as session:
        # First action: look around
        writer.write("look\r\n")
        commands_sent += 1
        await asyncio.sleep(COMMAND_INTERVAL)

        while True:
            try:
                # Read whatever the MUD sent
                text = clean_text(await read_until_pause(reader, 2.5))

                if not text:
                    # Nothing received — poke with look
                    text = "(no output received)"

                # Log MUD output (truncated for console)
                display = text[:600]
                if len(text) > 600:
                    display += f"\n  ... ({len(text)} chars total)"
                print(f"\n[{timestamp()}] MUD OUTPUT:\n{display}\n")

                # Feed to LLM
                history.append({"role": "user", "content": text[:3000]})

                # Loop detection: inject a nudge if the AI is stuck
                loop_nudge = detect_loop(recent_commands)
                if loop_nudge:
                    print(f"  [{timestamp()}] LOOP DETECTED — injecting nudge")
                    history.append({"role": "user", "content": loop_nudge})

                # Periodic orientation: every N commands, remind AI to check quests
                if commands_sent > 0 and commands_sent % PERIODIC_CHECK_INTERVAL == 0:
                    periodic_msg = (
                        "[SYSTEM: Periodic check — run 'quests' to review your progress, "
                        "or 'status' to check your health and resources. If you have been "
                        "in the same area for a while, consider exploring a new direction.]"
                    )
                    history.append({"role": "user", "content": periodic_msg})
                    print(f"  [{timestamp()}] PERIODIC CHECK injected (cmd #{commands_sent})")

                # Trim history
                if len(history) > MAX_HISTORY + 10:
                    history = history[-MAX_HISTORY:]

                # Get command from LLM
                raw_response = await query_ollama(history, session)
                command = sanitize_command(raw_response)

                # Track reporting commands
                if command.lower().startswith("bug "):
                    bugs_reported += 1
                elif command.lower().startswith("suggest "):
                    suggestions_sent += 1

                commands_sent += 1
                recent_commands.append(command.lower())
                history.append({"role": "assistant", "content": command})

                print(f"[{timestamp()}] AI COMMAND #{commands_sent}: {command}")
                print(f"  (bugs: {bugs_reported}, suggestions: {suggestions_sent})")

                # Send command
                writer.write(command + "\r\n")

                # Pace ourselves
                await asyncio.sleep(COMMAND_INTERVAL)

            except KeyboardInterrupt:
                print(f"\n[{timestamp()}] Interrupted by user. Logging out...")
                writer.write("quit\r\n")
                await asyncio.sleep(1)
                break
            except Exception as e:
                print(f"\n[{timestamp()}] ERROR: {e}")
                traceback.print_exc()
                # Try to recover
                await asyncio.sleep(5)
                try:
                    writer.write("look\r\n")
                except:
                    print(f"[{timestamp()}] Connection lost. Exiting.")
                    break

    print(f"\n[{timestamp()}] Session complete.")
    print(f"  Commands sent: {commands_sent}")
    print(f"  Bugs reported: {bugs_reported}")
    print(f"  Suggestions: {suggestions_sent}")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nShutdown.")
