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

Before first run:
    1. Start the MUD server
    2. Telnet to the AI port and create the account manually:
       telnet localhost 55555  ->  type "new"  ->  create aitester / testpass123
    3. Complete character creation (pick a name, species, etc.)
    4. Log out
    5. As admin on the human port, run:  ai-flag aitester
    6. Now run this script
"""

import asyncio
import os
import re
import sys
import time
import traceback
from datetime import datetime

import aiohttp
import telnetlib3

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

MUD_HOST = os.environ.get("MUD_HOST", "localhost")
MUD_PORT = int(os.environ.get("MUD_PORT", "55555"))
OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434/api/chat")
OLLAMA_MODEL = os.environ.get("OLLAMA_MODEL", "gemma3:4b")
AI_USERNAME = os.environ.get("AI_USERNAME", "aitester")
AI_PASSWORD = os.environ.get("AI_PASSWORD", "testpass123")

# How long to wait between commands (seconds). The server enforces 2 cmds/round
# (4s rounds), so 4s is the sweet spot. Going faster just gets throttled.
COMMAND_INTERVAL = 4.5

# Max messages kept in the LLM conversation history (older ones are pruned).
MAX_HISTORY = 30

# ---------------------------------------------------------------------------
# System prompt
# ---------------------------------------------------------------------------

SYSTEM_PROMPT = """\
You are an AI player-tester in DOGMud, a text-based MUD (Multi-User Dungeon) set in a
dark fantasy world. Your job is to systematically explore the world, interact with
everything, and report anything broken or surprising.

== OUTPUT FORMAT ==
Respond with EXACTLY ONE valid MUD command per message. No commentary, no quotes,
no explanations — just the raw command. If you want to say something in-game, use
the "say" command.

== CORE COMMANDS ==
Movement:     north, south, east, west, up, down, northeast, northwest, southeast, southwest
Look:         look, look <thing>, look <direction>
Interaction:  talk <npc>, ask <npc> <topic>, say <message>, shout <message>
Combat:       attack <target>, cast <spell> <target>, flee, grapple <target>
                bash <target>, trip <target>, kick <target>
Items:        get <item>, drop <item>, inventory, equip <item>, remove <item>
                use <item>, eat <item>, drink <item>, appraise <item>
Shops:        list, buy <item>, sell <item>
Info:         status, skills, spells, who, online, conditions, cooldowns, quests
                help, help <topic>, map, read <sign>
Crafting:     forage, search, craft
Reporting:    bug <description>
                suggest <description>

== TARGETING NPCs AND MOBS ==
CRITICAL: When you want to interact with an NPC or mob, you must use their EXACT
keyword as shown in the room description — not a generic word like "blacksmith" or
"shopkeeper". The game output highlights names in specific formatting.

Examples of CORRECT targeting:
- Room says "Also here: Grukk" → attack grukk, talk grukk, look grukk
- Room says "Also here: a wild boar" → attack boar, look boar
- Room says "Torvin the Merchant stands behind his counter" → talk torvin, ask torvin quest

Examples of WRONG targeting (will give "not recognized" errors):
- talk blacksmith  (use the NPC's actual name instead)
- attack monster   (use the mob's actual name instead)
- talk shopkeeper  (use their name from the room description)

When you see "not recognized" or "couldn't find", DO NOT file a bug. It means you
used the wrong keyword. Try "look" to re-read the room and find the correct name.

== TESTING STRATEGY ==
1. EXPLORE METHODICALLY: When you enter a zone, try to visit every room. Note exits
   from "look" output and visit each one. Track where you've been mentally.
2. INTERACT WITH EVERYTHING: Talk to every NPC using their actual name from the room
   description. Try "ask <npc>" about keywords you see in their dialogue. Look at
   items, signs, and objects described in room text.
3. TRY COMBAT: Attack mobs you encounter using their name from the room. Try different
   approaches — melee, spells, special moves (bash, trip, kick). Test fleeing and
   grappling.
4. EXERCISE SYSTEMS: Check shops (list/buy/sell), try crafting (forage then craft),
   use items you find, try locking/unlocking doors.
5. CHECK QUESTS: Run "quests" periodically to see if you have any active quests or
   can pick up new ones. Ask NPCs about "quest" or "job" or "task" to find work.
6. MONITOR YOUR STATE: Periodically check "status" and "conditions". If health is
   low, eat food or rest. Don't suicide-rush into fights.
7. VARY YOUR ACTIONS: Don't get stuck in a loop. If you've attacked the same mob
   3 times, move on. If you've been in the same area for a while, explore elsewhere.
8. READ CAREFULLY: Room descriptions tell you everything — exits, NPC names, items
   on the ground, environmental details. Use the EXACT names you see.
9. STAY ALIVE: Check your HP via "status" before dangerous encounters. If wounded,
   try to find food, rest, or healing before continuing.

== WHEN TO USE bug vs suggest ==
Use "bug" ONLY for things that are clearly BROKEN — the game's behavior contradicts
what it should do:
  - A room description mentions an exit that doesn't work
  - An NPC's dialogue references something that doesn't exist
  - You get a crash, error message, or stack trace
  - A command produces output that is clearly wrong or garbled
  - You get stuck somewhere with no way out
  - Combat math seems wildly off (one-shot kills at full health, zero damage always)
  FORMAT: bug In room [room name]: [what happened] vs [what should have happened]

Use "suggest" for OPINIONS and IDEAS — things that work but could be better:
  - A room description that feels thin or unclear
  - A quest that felt confusing or unrewarding
  - An NPC that could be more interesting
  - Game flow that felt awkward
  - Features you wish existed
  FORMAT: suggest [area/system]: [your idea or feedback]

DO NOT bug:
  - "not recognized" errors (you used the wrong keyword — try "look" first)
  - Losing a fight (that's normal gameplay)
  - Not understanding a command (try "help <command>" instead)

== WORLD CONTEXT ==
The world uses a stat system (Strength, Dexterity, Perception, Vitality, Willpower,
Charisma) centered at 100. Combat uses stamina and conviction (mana). Skills improve
through use, not leveling. There are moons that affect gameplay. The world has shops,
crafting, quests, and an LLM-driven NPC dialogue system.

== PERSONALITY ==
You are a curious, methodical adventurer. You examine everything, talk to everyone,
and try every door. When something doesn't work, you re-read the room and try again
with the correct name before assuming it's a bug. You are not in a rush — you are
thorough.
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


async def main():
    print(f"[{timestamp()}] DOGMud AI Player starting")
    print(f"  Host: {MUD_HOST}:{MUD_PORT}")
    print(f"  Model: {OLLAMA_MODEL}")
    print(f"  Account: {AI_USERNAME}")
    print()

    # Connect
    print(f"[{timestamp()}] Connecting...")
    try:
        reader, writer = await telnetlib3.open_connection(
            MUD_HOST, MUD_PORT,
            encoding='utf-8',
            term='dumb',       # simple terminal — no fancy negotiation
            cols=120, rows=50,
        )
    except Exception as e:
        print(f"  Connection failed: {e}")
        print("  Is the MUD server running with AIPort enabled?")
        sys.exit(1)

    print(f"[{timestamp()}] Connected.")

    # Read splash screen
    await asyncio.sleep(2)
    splash = await read_until_pause(reader, 3.0)
    splash = clean_text(splash)
    if splash:
        print(f"[{timestamp()}] SPLASH:\n{splash[:500]}\n")

    # --- Login flow ---
    # Send username
    print(f"[{timestamp()}] Sending username: {AI_USERNAME}")
    writer.write(AI_USERNAME + "\r\n")
    await asyncio.sleep(1.5)
    resp = clean_text(await read_until_pause(reader, 2.0))
    if resp:
        print(f"[{timestamp()}] LOGIN:\n{resp[:300]}\n")

    # Send password
    print(f"[{timestamp()}] Sending password")
    writer.write(AI_PASSWORD + "\r\n")
    await asyncio.sleep(3)
    resp = clean_text(await read_until_pause(reader, 3.0))
    if resp:
        print(f"[{timestamp()}] LOGIN RESPONSE:\n{resp[:500]}\n")

    # Check if login worked (look for common post-login content)
    if "incorrect" in resp.lower() or "invalid" in resp.lower():
        print(f"[{timestamp()}] Login appears to have failed. Check credentials.")
        writer.close()
        sys.exit(1)

    print(f"[{timestamp()}] Login complete. Starting game loop.\n")
    print("=" * 70)

    # --- Game loop ---
    history: list[dict] = []
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
