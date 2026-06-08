"""
DOGMud AI Player/Tester

Connects to the MUD on the AI port, logs in, and uses the Anthropic Claude API
to play the game autonomously. Designed for broad system coverage testing.

Prerequisites:
    pip install telnetlib3 anthropic

Usage:
    python tools/ai_player.py

    Environment variables:
        ANTHROPIC_API_KEY   required — your Anthropic API key
        MUD_HOST            default: dogmud.org
        MUD_PORT            default: 55555
        CLAUDE_MODEL        default: claude-sonnet-4-6
        AI_USERNAME         default: aitester
        AI_PASSWORD         default: testpass123

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
import traceback
from collections import deque
from datetime import datetime

import anthropic
import telnetlib3

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

MUD_HOST = os.environ.get("MUD_HOST", "dogmud.org")
MUD_PORT = int(os.environ.get("MUD_PORT", "55555"))
CLAUDE_MODEL = os.environ.get("CLAUDE_MODEL", "claude-sonnet-4-6")
AI_USERNAME = os.environ.get("AI_USERNAME", "aitester")
AI_PASSWORD = os.environ.get("AI_PASSWORD", "testpass123")

# Anthropic client (reads ANTHROPIC_API_KEY from env automatically)
try:
    claude_client = anthropic.Anthropic()
except anthropic.AuthenticationError:
    print("ERROR: ANTHROPIC_API_KEY environment variable is not set.")
    print("  Set it with: export ANTHROPIC_API_KEY='sk-ant-...'")
    sys.exit(1)

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

# Token budget: stop playing when reserve threshold is reached.
# TOKEN_BUDGET: total tokens the session is allowed to consume (input + output).
#   Default 250k — roughly 25% of a 1M-context session.
# BUDGET_RESERVE_PCT: stop at this fraction remaining (0.20 = stop at 80% used).
TOKEN_BUDGET = int(os.environ.get("TOKEN_BUDGET", "250000"))
BUDGET_RESERVE_PCT = float(os.environ.get("BUDGET_RESERVE_PCT", "0.20"))

# Where to write the end-of-session feedback report.
REPORT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)))

# ---------------------------------------------------------------------------
# System prompt
# ---------------------------------------------------------------------------

SYSTEM_PROMPT = """\
You are an AI play-tester in DOGMud, a text MUD set in a dark fantasy world reshaped by the
Chrysalis plague. Your job is to explore, interact with everything, and report genuine bugs.
Your secondary goal is to acquire and wear the best equipment you can find or craft.

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
Crafting:     forage, search, craft, help craft
Other:        look <sign>, bug <description>, suggest <description>

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
New characters start in Sanctum Basin with a guided quest chain (19 steps). Each step
requires you to PERFORM A SPECIFIC ACTION — it is not enough to just walk into the room.
Follow the NPC instructions and use the commands they tell you to use.

  1. Academy Hall (room 113) — enter to begin. Gives you the first quest flags.
  2. Market Street (room 108) — Merchant Adela teaches shopping:
     a. Type "list" to see her wares, then "buy <item>" to purchase something.
     b. Type "wield <weapon>" or "wear <armor>" to equip what you bought.
     Adela reacts to each action before advancing you.
  3. Training Yard (room 114) — Combat Trainer teaches fighting:
     a. Type "attack dummy" to fight the training dummy until it breaks.
  4. The Forge (room 109) — Korvath teaches crafting:
     a. Type "craft iron dagger" after he gives you materials.
  5. Alchemist's Workshop (room 111) — Yenna teaches alchemy:
     a. Type "craft healing poultice" after she gives you materials.
  6. West Meadow (room 106) — Wilderness Guide Fen teaches survival:
     a. Type "forage" when Fen tells you to.
     b. Type "track" when Fen tells you to.
  7. Observatory (room 116) — Elder Saris teaches spellcasting:
     a. Type "cast conviction-spike echo" to practice on the Chrysalis Echo.
  8. Cave system (rooms 118→119→120) — the final test. Fight through to the
     Aberrant Chrysalis boss in room 120. Defeat it.
  9. Basin Gate (room 102) — the Warden grants passage and completes the tutorial.

IMPORTANT: Each sub-step requires you to actually USE the command the NPC tells you.
If nothing happens, run "quests" to check your progress, then do what the quest
description says. The quest flags are sequential — you cannot skip ahead.

== POST-TUTORIAL: WORLD EXPLORATION ==
After completing the tutorial, the world opens up. Here are the zones and quest chains
you should explore, roughly in order of progression:

SANCTUM BASIN (post-tutorial):
  - Quest 2: The Warren Compact — talk to the tunnel shaman in the Labyrinth of Low
    Tunnels (underground, accessed from within Sanctum Basin). Bring 2 healing poultices
    to prove peaceful intent. NOTE: Completing this locks Quest 3.
  - Quest 3: The Scholar's Collection — talk to the Chrysalis Scholar near the cave
    entrance. She needs specimens from the tunnels: a bone totem and a spore sac.
    NOTE: Completing this locks Quest 2.

DUSTWALK ROAD (south from Basin Gate):
  - Quest 4: The Warden's Report — Road Warden Tessara at Warden's Post suspects
    bandits. Travel south, find their hideout, recover evidence (bandit's purse).

WATCHERS CROSSING (crossroads settlement):
  - Innkeeper Tolva, Merchant Brecca, Toll Collector Harn — talk to all NPCs.
  - Quest 5: The Innkeeper's Complaint
  - Quest 6: The Collector's Burden

THORNWALL CITY (major city, many NPCs and shops):
  - Quest 7: The Fallow Field
  - Quest 8: The City Watch's Missing Person
  - Quest 9: The Temple's Tithe Audit
  - Quest 10: The Drowning Post's Debt — talk to tavern keeper Marek about his
    troubles. Take his protection notice to Guard Captain Velk at the barracks.
  - Explore shops, buy better gear, sell loot

IRONWIND STEPPE (wilderness zone, dangerous):
  - This is the most dangerous zone with 40+ mob types. Bring good gear and potions.
  - Mobs include wolves, boars, raptors, goblins, and worse. Fight strategically.
  - Quest 11: The Windwarden's Dilemma — talk to Hermit Kael in the Sheltered
    Ridge Alcove (mid-steppe). He sends you to investigate a conflict:
    a. Visit Windwarden Sylara near the stone circle (southeast, goblin territory).
       Ask her about the conflict. She asks you to retrieve her stolen totem.
    b. Visit Geomancer Rhett at the Ridge Approach (northwest, rocky area).
       Ask him about windstone. He asks you to collect crystal samples.
    c. Go to the Kill Ground (wolf territory, east) — type "get totem" to pick up
       the carved wolf totem from the bones. Bring it to Sylara.
    d. Go to the Rocky Shelf (ridge, quartz veins) — type "get crystal" to chip off
       a windstone sample. Bring it to Rhett.
    e. Return to Kael. He asks you to CHOOSE A SIDE:
       - Say "sylara" to Kael → Quest 12: The Warden's Covenant (Path A)
         Go to Sylara, say "ready" for the ritual, then "accept" to complete it.
         Reward: Summon Steppe Spirit spell + spirit fetishes
       - Say "rhett" to Kael → Quest 13: The Prospector's Gambit (Path B)
         Go to Rhett, say "ready" for extraction, then "steady" to complete it.
         Reward: Windstone Aegis (powerful shield)
    YOUR CHOICE IS PERMANENT — the other path is locked out forever.
  - After Path A: cast "summon-steppe-spirit" to summon a spirit wolf companion.
    Ask Sylara for "fetish" to get more components when you run out.
  - After Path B: equip the Windstone Aegis for strong defensive stats.

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

== SURVIVAL & COMBAT ==
BEFORE every fight:
- Check "status" — know your HP, stamina, and conviction before engaging.
- If HP is below half, do NOT start a fight. Wait for regen, eat food, or drink a potion.
- If stamina is low, rest before fighting. Low stamina reduces your attack effectiveness.

DURING combat:
- Use "attack <name>" to engage, then the fight auto-attacks each round.
- Mix in special moves EVERY fight — rotate between "bash", "trip", "kick", "grapple".
  Each uses a different skill. Variety trains all your combat skills.
- Cast spells during combat too: "cast conviction-spike <target>" deals conviction damage.
  Rotate through ALL your known spells — check "spells" to see what you have learned.
- New spells are discovered through casting. The more you cast, the more you learn.

AFTER combat:
- Check "status" immediately. If HP is low, wait several rounds to regenerate before
  moving on. Do NOT rush into the next fight wounded.
- If you have healing items (poultices, food), use them: "drink poultice", "eat <food>".
- Check "conditions" to see active buffs/debuffs. Wait out negative conditions.

FLEEING:
- If HP drops dangerously low during a fight, type "flee" immediately.
- After fleeing, heal up before re-engaging. Dying loses progress.

DARK ROOMS:
- Some rooms are dark and you cannot see. Cast "cast chrysalis-glow" to create light.
  This is essential for cave exploration and underground areas. Always cast it when
  you enter a dark room.

GEAR OPTIMIZATION:
- Check "inventory" regularly. Look at items with "look <item>" to see their stats.
- Equip the best weapon you find: "wield <weapon>". Compare damage values.
- Wear the best armor: "wear <armor>". Check mitigation values (physical, magical).
- Equip a shield in your offhand if you find one: "wear <shield>".
- Sell junk at shops for gold, then buy upgrades. Always be improving your loadout.
- Try different weapon types — swords, maces, daggers, staves all train different skills.

== TESTING STRATEGY ==
1. FOLLOW THE TUTORIAL FIRST. Complete the quest chain before exploring freely.
2. INTERACT WITH EVERYTHING: Talk to every NPC, look at every noun in room descriptions,
   read signs, try every shop. Look at items you pick up to read their descriptions.
3. TRY COMBAT: Fight mobs using different tactics each fight — mix melee, spells, and
   special moves (bash/trip/kick/grapple). Variety trains different skills.
4. PURSUE QUESTS: After the tutorial, pick up quests in each zone. Use "ask <npc> quest"
   or "ask <npc> task" to discover available quests from NPCs.
5. READ HELPFILES: "help" lists topics. "help <topic>" gives details. When stuck or
   unsure about a system, read the relevant helpfile before filing a bug.
6. EXERCISE SYSTEMS: Try crafting (forage, then craft), use shops (list, buy, sell),
   try all spell types, test different weapon types.
7. MONITOR STATE: Check "status", "quests", "conditions", "skills" regularly.
8. GEAR UP: Buy or craft better equipment as you progress. Sell loot for gold.
   Always wear the best gear available. Compare items before equipping.
9. EXPLORE EVERY ZONE: Move through the world systematically — Basin → Dustwalk Road
   → Watchers Crossing → Thornwall City → Ironwind Steppe.
10. STAY ALIVE: Prioritize survival. Heal between fights. Flee when wounded. Dying
    wastes time. A cautious tester covers more ground than a dead one.

== WHEN YOU ARE STUCK ==
If a command fails or nothing happens, do NOT repeat it. Instead, try these in order:
1. Type "look" to re-read the room — you may have missed an exit or NPC name.
2. Type "quests" to check what you should be doing next.
3. Type "inventory" to check what you are carrying.
4. Type "status" to check your health and resources.
5. Type "help <topic>" to learn about a command or system you are unsure about.
6. Try a completely different approach — move to another room, talk to a different NPC,
   or work on a different quest.
7. If truly stuck in an area with no obvious exits, type "map" to see the zone layout.

== DIALOGUE SYSTEM ==
NPCs have two interaction modes:
- "talk <npc>" — starts a structured dialogue tree with hints about what to say next.
- "ask <npc> <topic>" — free-form topic queries. Try keywords like: quest, task, help,
  trouble, work, rumor, danger, history, crystal, totem, etc.
When an NPC's dialogue gives hints (e.g. "Ask about the conflict"), use those keywords
in your next "ask" command. Many quest steps advance through dialogue — keep talking.

== BUG AND SUGGESTION REPORTING ==
Use "bug" for things that are BROKEN — errors, crashes, missing exits, garbled output,
items that don't work, quest steps that won't advance, typos in descriptions.

  FORMAT: bug [Room/Area]: Tried [command]. Expected [what should happen]. Got [what
  actually happened]. [Any extra context.]

  EXAMPLES:
    bug Sheltered Ridge Alcove: Tried "ask kael quest". Expected dialogue about the
    conflict. Got "not recognized" error. Kael is visible in the room.

    bug Kill Ground: Tried "get totem". Expected to pick up the carved wolf totem.
    Got no response at all. I have quest 11-sylara active.

    bug Market Street: Description says "A bustling market steet" — typo, should be
    "street".

    bug Training Yard: Attacked the dummy but combat never started. Status shows I am
    not in combat. Dummy is visible in the room.

Use "suggest" for OPINIONS and IDEAS — improvements, missing flavor, confusing design.

  FORMAT: suggest [Area]: [Your idea or observation.]

  EXAMPLES:
    suggest Dustwalk Road: The road feels empty between waypoints. More ambient
    descriptions or wandering NPCs would help.

    suggest Ironwind Steppe: It would be nice if there was a way to track quest NPCs
    or get directions to them from other NPCs.

DO NOT bug: targeting errors (try "look" first — you probably used the wrong name),
losing fights (that is normal), not understanding a command (use "help <topic>" first),
or anything you have not tried at least twice with different approaches.

== WORLD CONTEXT ==
Stats (Strength, Dexterity, Perception, Vitality, Willpower, Charisma) are centered at 100.
Combat uses stamina and conviction. Skills and stats improve through use, not leveling.
Spells use "folds" — mental bifurcation of belief. Spells are discovered through practice,
not taught by trainers. The Chrysalis plague causes mutations through sustained combat.
Three moons influence the world.

== PERSONALITY ==
You are a careful, curious, and thorough play-tester. Your priorities in order:
1. SURVIVE — never rush into danger. Heal first, fight second.
2. PROGRESS QUESTS — always be working toward the next quest objective.
3. GEAR UP — constantly improve your equipment loadout.
4. EXPLORE — visit every room, read every description, talk to every NPC.
5. TEST SYSTEMS — try every command, spell, and combat move at least once.
6. REPORT — file clear, detailed bugs for anything broken. File suggestions for
   improvements. Report typos when you spot them.

You read room descriptions carefully and use exact NPC names from the "Also here:" line.
You follow quest hints and dialogue suggestions. When stuck, you check "quests", "look",
and "help" before trying random things. You NEVER repeat a command that failed — you try
a different approach. You keep yourself alive, healed, and well-equipped at all times.
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


async def query_claude(history: list[dict], system: str = None,
                       max_tokens: int = 60) -> tuple[str, int, int]:
    """Send conversation history to Claude and get a command back.

    Returns (text, input_tokens, output_tokens).
    """
    if system is None:
        system = SYSTEM_PROMPT
    try:
        # Run the synchronous Anthropic call in a thread to avoid blocking
        loop = asyncio.get_event_loop()
        response = await loop.run_in_executor(None, lambda: claude_client.messages.create(
            model=CLAUDE_MODEL,
            max_tokens=max_tokens,
            temperature=0.7,
            system=system,
            messages=history[-MAX_HISTORY:],
        ))
        in_tok = response.usage.input_tokens
        out_tok = response.usage.output_tokens
        return response.content[0].text, in_tok, out_tok
    except anthropic.APITimeoutError:
        print(f"  [{timestamp()}] CLAUDE timeout")
        return "look", 0, 0
    except anthropic.APIError as e:
        print(f"  [{timestamp()}] CLAUDE API error: {e}")
        return "look", 0, 0
    except Exception as e:
        print(f"  [{timestamp()}] CLAUDE error: {e}")
        return "look", 0, 0


# ---------------------------------------------------------------------------
# Prompt-driven I/O helpers
# ---------------------------------------------------------------------------

async def read_until(reader, marker: str, timeout: float = 10.0) -> str:
    """Read from the MUD until `marker` appears in the accumulated text, or timeout."""
    chunks = []
    deadline = asyncio.get_event_loop().time() + timeout
    while True:
        remaining = deadline - asyncio.get_event_loop().time()
        if remaining <= 0:
            break
        try:
            data = await asyncio.wait_for(reader.read(8192), timeout=min(remaining, 2.0))
            if not data:
                break
            chunks.append(data)
            combined = "".join(chunks)
            if marker in combined:
                # Drain any trailing data that arrives immediately after the marker
                await asyncio.sleep(0.2)
                try:
                    extra = await asyncio.wait_for(reader.read(8192), timeout=0.3)
                    if extra:
                        chunks.append(extra)
                except asyncio.TimeoutError:
                    pass
                break
        except asyncio.TimeoutError:
            # No data within this read window; check if we already have marker
            combined = "".join(chunks)
            if marker in combined:
                break
            # If no marker yet but still within overall deadline, keep reading
            continue
    return "".join(chunks)


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


# ---------------------------------------------------------------------------
# Connection and login
# ---------------------------------------------------------------------------

async def connect(reader, writer):
    """Read and discard the splash screen + initial username prompt.

    Returns the text of the splash (for logging).  The reader is left
    positioned right after the 'Username (or "new"):' prompt, ready
    for the caller to send a username or "new".
    """
    splash = await read_until(reader, ":", timeout=15.0)
    splash = clean_text(splash)
    if splash:
        print(f"[{timestamp()}] SPLASH:\n{splash[:500]}\n")
    return splash


async def send_and_wait(writer, reader, text: str, label: str,
                        marker: str = ":", timeout: float = 10.0) -> str:
    """Send a line to the MUD and read until the next prompt marker appears."""
    writer.write(text + "\r\n")
    resp = await read_until(reader, marker, timeout=timeout)
    resp = clean_text(resp)
    if resp:
        print(f"[{timestamp()}] {label}:\n{resp[:400]}\n")
    return resp


async def attempt_login():
    """Try to log in with existing credentials.

    Returns (reader, writer, success). On failure the connection is dead
    (server disconnects on bad login), so caller must reconnect.
    """
    print(f"[{timestamp()}] Attempting login as {AI_USERNAME}...")
    reader, writer = await telnetlib3.open_connection(
        MUD_HOST, MUD_PORT,
        encoding='utf-8',
        term='dumb',
        cols=120, rows=50,
    )

    # Read splash + username prompt (ends with ":")
    await connect(reader, writer)

    # Send username — expect password prompt (also ends with ":")
    resp = await send_and_wait(writer, reader, AI_USERNAME, "LOGIN_USER",
                               marker=":", timeout=8.0)

    # Send password — the response is either the game world (success)
    # or an error message (failure + disconnect).  Game prompts don't
    # have a single reliable marker so we just read until a pause.
    writer.write(AI_PASSWORD + "\r\n")
    await asyncio.sleep(2)
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
    reader, writer = await telnetlib3.open_connection(
        MUD_HOST, MUD_PORT,
        encoding='utf-8',
        term='dumb',
        cols=120, rows=50,
    )

    # Read splash + username prompt
    await connect(reader, writer)

    # Step 1: send "new" — expect "Choose your username:" prompt
    await send_and_wait(writer, reader, "new", "NEW_ACCOUNT",
                        marker=":", timeout=8.0)

    # Step 2: choose username — expect "Choose your password:" prompt
    await send_and_wait(writer, reader, AI_USERNAME, "SET_USERNAME",
                        marker=":", timeout=8.0)

    # Step 3: choose password — expect "Repeat your password:" prompt
    await send_and_wait(writer, reader, AI_PASSWORD, "SET_PASSWORD",
                        marker=":", timeout=8.0)

    # Step 4: repeat password — expect "Email Address" or "screen reader" prompt
    await send_and_wait(writer, reader, AI_PASSWORD, "VERIFY_PASSWORD",
                        marker=":", timeout=8.0)

    # Step 5: email (optional — send empty) — expect "screen reader?" prompt
    await send_and_wait(writer, reader, "", "EMAIL",
                        marker=":", timeout=8.0)

    # Step 6: screen reader prompt — expect "create a new user" confirm prompt
    await send_and_wait(writer, reader, "n", "SCREEN_READER",
                        marker=":", timeout=8.0)

    # Step 7: confirm creation — after this the server logs us in.
    # The response is the game world, not a prompt, so read until pause.
    writer.write("y\r\n")
    await asyncio.sleep(3)
    resp = clean_text(await read_until_pause(reader, 3.0))
    if resp:
        print(f"[{timestamp()}] CONFIRM_CREATE:\n{resp[:500]}\n")

    print(f"[{timestamp()}] Account creation complete.")
    return reader, writer


# ---------------------------------------------------------------------------
# End-of-session report
# ---------------------------------------------------------------------------

REPORT_PROMPT = """\
You are reviewing your play-testing session of DOGMud. You will receive the
conversation history from your session (MUD output interleaved with the
commands you sent). Write a structured feedback report in Markdown.

Include these sections:

## Session Summary
Brief overview: how far you got, what zones you visited, quests attempted.

## Bugs Found
List every bug you reported via the "bug" command, plus any you noticed but
didn't report. Include the room/area, what you tried, and what went wrong.
If none, say "None observed."

## Suggestions
List every suggestion you filed via "suggest", plus any additional ideas.

## Quest Progress
Which quests you started, completed, or got stuck on. For stuck quests,
describe exactly where you got stuck and what you tried.

## Combat & Balance Notes
Observations about fight difficulty, gear progression, skill usage, any
fights that felt too easy or too hard.

## Systems Tested
Which game systems you exercised (crafting, shopping, dialogue, spells,
combat moves, foraging, etc.) and how well they worked.

## Session Stats
Commands sent, bugs reported, suggestions filed, tokens consumed.

Be specific and honest. This report is for the game developer.
"""


async def generate_report(history: list[dict], commands_sent: int,
                          bugs_reported: int, suggestions_sent: int,
                          tokens_used: int, stop_reason: str) -> tuple[str, int]:
    """Generate a feedback report from the session history.

    Returns (report_text, tokens_consumed_by_report).
    """
    # Build a condensed transcript for the report prompt
    transcript_lines = []
    for msg in history:
        prefix = "MUD>" if msg["role"] == "user" else "CMD>"
        # Truncate long MUD outputs for the report context
        content = msg["content"][:500]
        transcript_lines.append(f"{prefix} {content}")
    transcript = "\n".join(transcript_lines[-80:])  # last 80 exchanges

    stats = (
        f"\n\nSession stats: {commands_sent} commands sent, "
        f"{bugs_reported} bugs reported, {suggestions_sent} suggestions filed, "
        f"{tokens_used:,} tokens consumed. Stop reason: {stop_reason}."
    )

    report_messages = [{"role": "user", "content": transcript + stats}]

    text, in_tok, out_tok = await query_claude(
        report_messages, system=REPORT_PROMPT, max_tokens=4096
    )

    # Prepend a header
    header = (
        f"# DOGMud AI Playtest Report\n"
        f"**Date:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
        f"**Model:** {CLAUDE_MODEL}\n"
        f"**Account:** {AI_USERNAME}\n"
        f"**Stop reason:** {stop_reason}\n\n"
    )

    return header + text, in_tok + out_tok


# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------

async def main():
    print(f"[{timestamp()}] DOGMud AI Player starting")
    print(f"  Host: {MUD_HOST}:{MUD_PORT}")
    print(f"  Model: {CLAUDE_MODEL}")
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
    budget_limit = int(TOKEN_BUDGET * (1.0 - BUDGET_RESERVE_PCT))
    print(f"  Token budget: {TOKEN_BUDGET:,} total, stop at {budget_limit:,} "
          f"({BUDGET_RESERVE_PCT:.0%} reserved for report)")
    print("=" * 70)

    # --- Game loop ---
    history: list[dict] = []
    recent_commands: deque = deque(maxlen=LOOP_WINDOW)
    commands_sent = 0
    bugs_reported = 0
    suggestions_sent = 0
    tokens_used = 0
    stop_reason = "unknown"

    # First action: look around
    writer.write("look\r\n")
    commands_sent += 1
    await asyncio.sleep(COMMAND_INTERVAL)

    while True:
        try:
            # --- Budget check ---
            if tokens_used >= budget_limit:
                stop_reason = "budget"
                print(f"\n[{timestamp()}] TOKEN BUDGET reached "
                      f"({tokens_used:,}/{budget_limit:,}). Stopping game loop.")
                break

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

            # Trim history — ensure alternating user/assistant pattern for Claude
            if len(history) > MAX_HISTORY + 10:
                history = history[-MAX_HISTORY:]

            # Merge consecutive user messages (Claude requires alternating roles)
            merged_history = []
            for msg in history:
                if merged_history and merged_history[-1]["role"] == msg["role"]:
                    merged_history[-1]["content"] += "\n\n" + msg["content"]
                else:
                    merged_history.append(dict(msg))

            # Get command from LLM
            raw_response, in_tok, out_tok = await query_claude(merged_history)
            tokens_used += in_tok + out_tok
            command = sanitize_command(raw_response)

            # Track reporting commands
            if command.lower().startswith("bug "):
                bugs_reported += 1
            elif command.lower().startswith("suggest "):
                suggestions_sent += 1

            commands_sent += 1
            recent_commands.append(command.lower())
            history.append({"role": "assistant", "content": command})

            pct_used = tokens_used / TOKEN_BUDGET * 100
            print(f"[{timestamp()}] AI COMMAND #{commands_sent}: {command}")
            print(f"  (bugs: {bugs_reported}, suggestions: {suggestions_sent}, "
                  f"tokens: {tokens_used:,}/{TOKEN_BUDGET:,} [{pct_used:.1f}%])")

            # Send command
            writer.write(command + "\r\n")

            # Pace ourselves
            await asyncio.sleep(COMMAND_INTERVAL)

        except KeyboardInterrupt:
            stop_reason = "interrupted"
            print(f"\n[{timestamp()}] Interrupted by user.")
            break
        except Exception as e:
            print(f"\n[{timestamp()}] ERROR: {e}")
            traceback.print_exc()
            # Try to recover
            await asyncio.sleep(5)
            try:
                writer.write("look\r\n")
            except:
                stop_reason = "disconnect"
                print(f"[{timestamp()}] Connection lost.")
                break

    # --- End-of-session report ---
    print(f"\n[{timestamp()}] Generating feedback report "
          f"(tokens remaining: {TOKEN_BUDGET - tokens_used:,})...")

    report = await generate_report(history, commands_sent, bugs_reported,
                                   suggestions_sent, tokens_used, stop_reason)
    tokens_used += report[1]

    report_name = f"ai_playtest_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    report_path = os.path.join(REPORT_DIR, report_name)
    with open(report_path, "w", encoding="utf-8") as f:
        f.write(report[0])
    print(f"[{timestamp()}] Report written to: {report_path}")

    # Log out
    try:
        writer.write("quit\r\n")
        await asyncio.sleep(1)
    except Exception:
        pass

    print(f"\n[{timestamp()}] Session complete.")
    print(f"  Commands sent: {commands_sent}")
    print(f"  Bugs reported: {bugs_reported}")
    print(f"  Suggestions:   {suggestions_sent}")
    print(f"  Tokens used:   {tokens_used:,}/{TOKEN_BUDGET:,}")
    print(f"  Stop reason:   {stop_reason}")
    print(f"  Report:        {report_path}")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nShutdown.")
