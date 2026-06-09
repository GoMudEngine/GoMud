# DOGMud Patch Notes

## 2026-06-08 — Bug fixes: duplicate warcry/rally conditions + CI data race

- **Warcry and Rally no longer appear twice** in the conditions card and the
  `conditions` command. Each effect existed as both a buff (bookkeeping) and a
  combat condition (mechanics) with the same name; the display now shows it
  once, via a new `condition-mirror` buff flag the buff-display loops skip.
- **Fixed a data race in the config system.** Every config getter validated the
  config (a one-time mutation) while holding only a read lock, so concurrent
  first-use callers could mutate shared state simultaneously — a real
  server-startup race the CI `-race` detector flagged. Validation now runs once
  under a write lock (`ensureConfigValidated`, double-checked), and
  `AddOverlayOverrides` takes the write lock it always needed.
- **Quieted a spurious caravan log error.** Pickup-only vendor stops no longer
  call `SaveThroughput` (which logged a harmless "no cached entry" error every
  time); throughput is only persisted when a delivery actually changes it.

## 2026-06-08 — Web client: item icons in the equipment & inventory card

The equipment/inventory card now shows a small painted **icon** for each item
instead of a plain type glyph — drawn from a community icon pack plus a set of
custom-generated icons filling the gaps (staffs, knuckles, hook-spears,
component bags, and more). Icons pick the best match by equipment slot first,
then material, with a clean SVG fallback when no art exists. The grid is denser
and reflows to fit narrow panels.

## 2026-06-08 — Web client: readability fixes

- **Companion & party names in the Vitals card are legible again.** Their names
  were nearly invisible (light text on the parchment field); they're now dark,
  high-contrast ink, while their bars stay gently dimmed to set them apart from
  your own.
- **The Triggers card no longer cramps when actions are queued.** Adding a queue
  bar used to squeeze the trigger rows; the list now scrolls instead, so the
  rows keep their normal height.
- **The graphical map controls stay out of the way** on small map panels —
  they shrink and fade rather than overlapping the map itself.

## 2026-06-08 — Under the hood: security & stability hardening

Several defensive fixes hand-ported from upstream GoMud: admin-only gating on
the web basic-auth and internal system commands, safer nil-handling when moving
between or saving rooms, and a forced password change for any legacy
plaintext-stored password on next login. No visible change in normal play.

## 2026-06-08 — Fix: trigger action queue now paces on the cooldown

The "Queue this action" trigger option could fire everything at once instead of
releasing one ability per cooldown — especially outside of combat. The web client
now tracks the shared special-move cooldown reliably (the server reports it every
round) and releases exactly one queued action each time the cooldown frees.

## 2026-06-08 — Fix: web-client ticks now fire reliably

Ticks (the automation panel's interval timers) could silently stop firing while
you were actively playing — especially with triggers going off. Their countdowns
were being reset every time your character changed. Ticks now keep running on
schedule regardless of other activity.

## 2026-06-08 — Web client: your buffs are back in the Status panel

The **Status & Conditions** panel now lists your active buffs — Illumination,
Iron Will, shields, and the rest — as teal "status" chips, right alongside your
combat conditions. Stealth effects (Hidden, Empathic Shroud) stay hidden, as
they should.

- The panel now fills in **the moment you log in**, instead of sitting blank
  until something next changed.
- Your **aliases and triggers** also load reliably at login now — they no longer
  occasionally come up empty in the automation panel.

## 2026-06-08 — Web client: vitals show your locked pools

- Each HP/SP/CP bar now fences off the **reserved** (locked-away) portion of the
  pool as a cross-hatched segment, so a pool that's half-reserved no longer looks
  full.
- The bar's **color now tracks your usable pool** — the part you can actually
  spend — instead of the whole pool.
- The in-terminal **prompt bars** were brought in line with the same look: the
  reserved blocks sit to the right in neutral grey, and the colors match the
  vitals window.

## 2026-06-08 — Web client look & feel

- **One console header.** The brand bar and the page-nav bar are now a single
  rounded console plate — a gold wordmark on the left, page chips on the right —
  with a matching footer card. That frees up a row of height for the game cards
  in the middle.
- **The full name.** The site now shows **Delusions of Grandeur** throughout —
  the header, the page titles, and the connect button.
- **No doubled map.** In the web client (which already has the graphical map
  panel), the small ASCII minimap next to room descriptions is now hidden, giving
  room text the full width. Telnet and Mudlet keep their minimap.
- **Readability & focus.** The starfield on the connect splash is visible again;
  the game feed has more line spacing so fast text doesn't blur together; and the
  command bar grabs focus as soon as you type — clicking the map or inventory
  never leaves you unable to move.
- The **known-spells** list (`spells`) was narrowed to fit a standard 80-column
  screen.

## 2026-06-08 — Stealth no longer announces itself

You're no longer told when you're hidden or when you stop being hidden — if you
can't know who spotted you, you can't know you've been spotted. The `conditions`
command no longer lists stealth effects (Hidden, Empathic Shroud), and Empathic
Shroud no longer tells you when it lapses. Others in the room still see you slip
into and out of the shadows.

## 2026-06-08 — Web client automation panel: action queue

Triggers can now feed a shared **action queue** that respects your ability
cooldown — so you can set up a rotation that auto-executes.

- Your special abilities (kick, trip, bash, grapple, taunt, rally, warcry, and
  casting any spell) all share one cooldown. A trigger set to **"Queue this
  action"** drops its command into a queue instead of firing it right away; the
  queue then runs **one ability each time the cooldown frees**, in order.
- Use it to react to several things at once — e.g. a few buffs drop together and
  their re-cast triggers all queue up, then fire one after another as fast as the
  cooldown allows. **Queue at front** lets an urgent action (an emergency heal)
  jump the line.
- The queued triggers are highlighted and float to the top of the Triggers tab
  with their place in line; a **Clear** button empties the queue, and dying (or
  reloading) clears it too.
- **Rally** and **warcry** now announce when they wear off, so a trigger can
  catch it and re-cast.
- See `help triggers` for the details and worked examples.

## 2026-06-07 — Web client automation panel: Triggers

The automation panel's **Triggers** tab is live — the panel's four tabs (Macros,
Aliases, Ticks, Triggers) are now complete.

- **Triggers** watch your game text and run command(s) when a pattern matches.
  Use `*` wildcards to match anything; each `*` is captured as `$1`, `$2`, … for
  use in the commands (e.g. pattern `* tells you '*'` → `reply got it, $1`).
- **Optional if/else condition.** A trigger can branch on one condition:
  your HP/SP/CP **percent** (measured against your *usable* pool), an active
  **status/condition**, a **captured** value, your current **target**, or
  whether your shared **ability cooldown** is ready. Leave a branch blank to do
  nothing.
- **Point-and-click builder** in the Triggers & Timers panel: + New, the if/else
  builder, enable/disable toggle, left-click to test-fire, right-click to
  Edit / Duplicate / Remove. Triggers run in the web client; definitions save to
  your account. A short cooldown prevents a matching line from spamming.
- See `help triggers` (and `help ticks`).

## 2026-06-07 — Web client automation panel: Ticks

The automation panel's **Ticks** tab is now live.

- **Ticks** are timers you set up that automatically run one or more commands
  every few seconds while you're playing in the web client — e.g. "sip a
  healing potion every 30 seconds."
- Manage them in the **Triggers & Timers** panel: **+ New** to add one (name,
  the command(s), and the interval in seconds), an on/off toggle to pause it,
  left-click to run it right now, right-click to Edit / Duplicate / Remove.
- Separate multiple commands with `;`. Minimum interval is 1 second.
- Tick definitions save to your account, but they only *run* in the web client.
- Under the hood: the web client now talks back to the server over GMCP for
  point-and-click editing (a fix also lets the web client send GMCP at all).

## 2026-06-07 — Web client automation panel (Macros & Aliases)

The reserved right-column panel is now a working **Automation panel**.

- **Macros & Aliases tabs.** Your macros and custom aliases appear as compact
  sticker-chips — each shows just its shortcut (an alias like `war=warcry`
  reads `war`; macros read `F1`, `F2`, …), with the full command on hover. They
  populate on login and update instantly when you change one.
- **Click to use, right-click to manage.** Left-click a chip to fire it
  (a macro runs `=N`, an alias runs its expansion). Right-click for
  Edit / Duplicate / Remove, or hit **+ New** to add one — all from the panel,
  no typing required.
- **Ticks & Triggers tabs** are present but show "coming soon" — those land in
  the next phases.
- Macros and aliases still work exactly as before from any client; the panel is
  just a point-and-click front end for them.

## 2026-06-07 — Web client terminal theming

The web client's game terminal now matches the antique tooled-leather UI.

- **Leather color palette.** The terminal's background, text, and the 16 ANSI
  colors were retuned to the warm parchment-and-leather theme — no more cold
  black rectangle clashing with the brass-framed panels. Game colors (room
  titles, exits, combat, gold, prompt) stay clearly readable and distinct.
- **IBM Plex Mono.** The terminal switched from Courier to a self-hosted IBM
  Plex Mono — a warmer, more legible monospace that pairs with the serif chrome
  while keeping the fixed-width grid (room text, prompt bars, and the ASCII map
  stay aligned).
- **Brighter blue.** Blue was lifted so the connect splash and in-game exits
  read clearly on the warm background.
- **Card frames.** The vellum cards (session/reconnect strip, Vitals, Status &
  Conditions, command input) gained a visible thin red frame; they previously
  read as borderless against the dark page.

## 2026-06-07 — Web client inventory & equipment panel

The web client gained a graphical, interactive inventory panel.

- **Tabbed Inventory panel.** A new panel (in the left column) shows your
  gear across four tabs — **Equipped**, **Bandolier**, **Components**, and
  **Backpack** — each as a grid of item tiles with a type icon.
- **Equipped tab.** Lists every slot you're wearing in the same order as the
  `eq` command, including mutation slots (extra arms/wrists, tail) when you
  have them. Only filled slots are shown.
- **Charge meters & stacks.** Items with limited charges show a small vertical
  meter; identical items in your bag/bandolier/component bag collapse into one
  tile with an `xN` count.
- **Right-click to act.** Right-click any item for a context menu — Look,
  Identify, Equip/Remove, Drink/Eat, Drop — which runs the command on that
  exact item (no more grabbing the wrong duplicate). The result appears in
  the game feed.
- **Auto-sizing.** The panel grows to fit a long equipment list (e.g. an
  extra-arms character), borrowing space from the map.
- **Status panel freshness.** The Status & Conditions panel now updates
  immediately when a condition wears off, instead of lagging.
- **Under the hood.** Fixed a couple of latent bugs where gear in the newer
  equipment slots (shoulders, back, wrists, extra arms, tail, component bag,
  second ring) was not counted for worn buffs or gear value.

## 2026-06-06 — In-game web client dashboard

The web client's play screen was rebuilt from a bare terminal with floating
windows into a docked, resizable dashboard.

- **Docked panel layout.** Map, Vitals, Scene (placeholder), the game feed
  + command bar, Comms, Status & Conditions, and Triggers (placeholder) now
  sit in a tidy three-column layout instead of floating windows you had to
  arrange yourself.
- **Resize, rearrange, collapse, pop out.** Drag the dividers between
  columns to resize them; drag a panel's title bar onto another to swap
  their spots; collapse a panel to just its header; or pop any panel out
  into a floating window and dock it back. Your arrangement is remembered
  across reloads, with a "Reset layout" button to start fresh.
- **Fits any screen.** The layout reflows as the window shrinks — three
  columns on desktop, a tabbed side rail on smaller windows, and a bottom
  tab-bar with a slide-up drawer on phones — so the client stays usable on
  any device.
- **Tabbed Comms with its own input.** Say, Whisper, Party, and Broadcasts
  each get a tab, and a chat box sends to the active channel without leaving
  the panel.
- **Status & Conditions panel.** Shows your character and any active
  conditions as at-a-glance chips.
- **Auto-sized game text.** The feed font scales so a full 80 columns
  always fit, keeping room descriptions and the inline map aligned.
- **Themed to match.** The whole client wears the leather/brass look with
  parchment accent cards and a re-themed connect button.

## 2026-06-06 — Warm-dark web site theme

The public web pages were rethemed from the old arcade-pixel look to a
warm-dark cartographer style that matches the leather map.

- **New site theme.** The nav, header, footer, and content panels now use a
  warm-dark palette with brass buttons and a serif typeface, replacing the
  indigo `Press Start 2P` pixel styling.
- **Landing hero.** The home page leads with the MUD name, a tagline, a
  prominent "Play in Browser" button, the telnet connection info, and a live
  count of adventurers online.
- **Themed info pages.** The Who's Online and configuration tables and the
  404 page inherit the new look.

## 2026-06-06 — Tooled-leather web mapper with connection types and party markers

The web map got a full visual overhaul alongside new data from the server.

- **Antique tooled-leather style.** The map now renders on a fixed
  leather-textured SVG surface (the frame) with a nested pannable world
  view. The aesthetic is hand-tooled parchment rather than a flat grid,
  giving the map a world-artifact feel.
- **Per-exit connection styling.** Every connection line is styled by
  exit type: biome-aware roads, trails, and waterways; locked doors;
  secret passages (dimmed/dashed); one-way arrows; gate/barrier exits;
  cross-zone boundary stubs labeled with the destination zone; and
  fog-of-war stubs that hint an unexplored passage continues beyond the
  known edge.
- **Stairs ticks.** Vertical exits (up/down) draw a faint ▲ or ▼ tick
  on the room node — unchanged from the previous pass, but now correctly
  styled against the leather background.
- **Party-member markers.** The server now includes a `party` field in
  the `Zone.Map` GMCP payload (a list of room IDs occupied by party
  members). The client renders small figures on those nodes so you can
  see where your group is on the map at a glance.
- **Raised current-room node.** Your current room is rendered with a
  drop-shadow lift, making it immediately distinguishable from adjacent
  rooms.
- **Server-side snapshot additions.** `SnapshotExit` gained `Locked`,
  `Secret`, `OneWay`, `Gate`, `Stub`, and `ToZone` fields. Unvisited
  and cross-zone exits are now emitted as stub entries (previously
  dropped) so the client can render passage hints without revealing
  unvisited room details. `nodeExit.Gate` is derived from the exit's
  `ExitMessage` field.

## 2026-06-06 — Cartesian map consistency engine

Under-the-hood infrastructure that keeps the world map geometrically
coherent as new zones and rooms are added.

- **Startup consistency pass.** When the server boots, every zone's room
  grid is checked for coordinate collisions (two rooms on the same cell),
  non-reciprocal exits, and exits whose compass direction contradicts the
  actual coordinate delta between the rooms. A long-connector soft warning
  fires when a multi-cell passage's straight span clips another room's
  cell. Behavior is controlled by the new `GamePlay.MapConsistencyEnforce`
  config knob (`off` | `warn` (default) | `panic`); the default logs
  findings without interrupting startup.
- **`cartcheck [zone]` admin command.** Runs the same check on demand and
  prints findings to the admin's console, optionally scoped to a single
  zone. Useful for triaging new content before promoting to production.
- **`oneway: true` exit flag.** Marks an intentional one-way passage (a
  trapdoor, a one-way drop, etc.). Suppresses the reciprocity check for
  that exit while keeping it in the collision scan.
- **`non_cartesian: true` zone flag.** Set in a zone's `zone-config.yaml`
  to declare intentionally toroidal or maze geometry. Skips the three hard
  checks for that zone; the web mapper will render its wrap exits as edge
  stubs rather than misplaced connectors.
- **Phased rollout.** Shipped at `warn`. Triage findings with `cartcheck`
  after each content push, then flip to `panic` once the world is clean.

## 2026-06-06 — Hybrid web mapper with fog of war

The web client now shows an explored-zone map that fills in as you move.

- **Fog of war.** Only rooms you have visited appear on the map. Each
  move marks the destination room in your character's per-zone visited
  list; the map updates immediately on arrival.
- **Hybrid node style.** Each room is a small biome-tinted circle with
  a faint glyph overlay. Biome tints distinguish city, forest, cave,
  water, marsh, and other terrain types at a glance.
- **Wrap edge-stubs.** Toroidal or maze exits (zones declared
  `non_cartesian`) render as teal stubs with outward chevrons rather
  than dangling connectors, keeping the map readable.
- **Up/down ticks.** Vertical exits draw a faint ▲ or ▼ on the room
  node so multi-level areas are visible without cluttering the 2-D grid.
- **Long-exit connectors.** Multi-cell spanning exits render as
  proportionally longer amber lines.
- **Fit, center, and zoom controls.** Overlay buttons let you fit the
  whole explored zone into view, re-center on your current room, or
  step the zoom in and out.

## 2026-06-05 — A world that takes sides

The living-world pass reaches across the whole map. Groups of people (and
creatures) now have their own loyalties, and the back roads and outlying
settlements have started to breathe the way Stillwater and Thornwall do.

- **Friends and enemies between groups.** The town guards, road wardens, caravan
  trains, the merchants' concord, and the village folk now stand together — wrong
  one of them and word travels to all of them. Bandits and the steppe goblins
  stand on the other side of that line. You can earn or lose standing with each,
  and it colors how they treat you.
- **The Warren, reconsidered.** The tunnel-folk of the Warren are no longer
  treated as outlaws. They are an insular people who keep to themselves and are
  more often looked down on than feared — mistrustful of surface-dwellers on
  sight, but not your enemy unless you make them one. Their regard is something
  you earn.
- **Two more places that live.** The farming village of Ashwick and the
  waystation at Watcher's Crossing now keep daily hours, know one another, trade
  short conversations, and gossip their local troubles — the herbalist and the
  wary newcomer she's taken in, the innkeeper and the toll-keeper and their
  good-natured feud over the bridge fee.
- **The wild looks after its own.** Wolf packs, the goblin tribe, and the Warren
  now treat their members as kin — harm one and the others remember it. Out in
  the wilds, a steppe hermit and the wandering foragers will share what they've
  seen if you stop to listen: the goblins growing bolder, the steppe drying, the
  timber wolves ranging closer to the paths.
- **Word on the road.** Travelers, peddlers, the road warden, and the caravan
  master all have something to say about the bandits working the trade roads —
  and the caravans now cross under hired guard, running the gauntlet between
  towns.
- **Crimes get the right witnesses.** Rob or attack someone in town and the
  guards now enforce the law — warn you, move to arrest — instead of chasing a
  personal grudge, while ordinary townsfolk react with alarm and raise the cry
  rather than forming a vigilante mob. Actual fighters who see it will still come
  after you themselves.
- **Shops keep their shelves stocked.** General stores in the trade towns no
  longer run dry — their everyday basics now restock on a steady supply cadence,
  joining the cooks and enchanters whose supply chains were sorted out earlier.
- **Stability.** Fixed a rare crash that could take the server down if two people
  happened to connect at the exact same instant.

## 2026-06-03 — Private cells, taunts that stick, and combat polish

- **Your own holding cell.** Getting arrested no longer drops you into a shared
  room. Each prisoner is now taken to a private cell that only they can enter —
  no cellmates, no one wandering in, no guard following you through the door. You
  walk free by paying your fine or serving your time, and the cell is dismantled
  when you go. Your sentence runs even while you are logged off, so if you
  disconnect mid-sentence you return to a fresh cell with your time still ticking
  down — or, if your term elapsed while you were away, you simply come back free.
- **Taunts actually stick.** A companion or a taunting fighter now holds an
  enemy's attention for a few rounds after a successful taunt, instead of the
  enemy snapping straight back to whoever last hit it. Your golem can finally
  stand in front and take the blows the way a tank should. (Tuning knob:
  `TauntHoldRounds`, default 4.)
- **No more "%s" in the brawl.** Golems and other natural bashers were printing a
  raw placeholder where their attack name should be — they now "slam" with a
  crushing slam, while shield-bearers land a proper shield bash.

## 2026-06-03 — Stillwater comes to life

The lakeside town of Stillwater is the first place where the townsfolk truly
behave like a community rather than fixtures:

- **They keep hours.** The innkeeper and her barmaid open the Pike & Lantern at
  dawn and work it into the night before retiring to the loft; the smith stokes
  his forge by day and nurses a quiet tankard in the evening; the priest holds
  morning and evening rites; the dock master walks the promenade at dusk; the
  apothecary, the miller, and the healer each keep their own working day. Visit
  at different hours and you'll find different people doing different things —
  and the sleepers really are asleep.
- **They know each other.** The town now has a web of ties — kin, employers and
  the people who work for them, an old fisherman and the tradesmen he taught, a
  pair of healers who compare notes, neighbors who look in on one another. Idle
  townsfolk strike up short conversations drawn from those relationships, so the
  square and the tavern feel lived-in.
- **They remember the town's troubles.** The fishermen and the constable speak
  of the failing catch and the things coming out of the caves; the healer and
  the priest carry the older grief of a man the lake took years ago. Gossips —
  the barmaid, the old net-mender, the cottager on the lane, the beggar in the
  square, and a passing pilgrim — carry that news around.

This is a behind-the-scenes pass: no new quests, no map changes. It's the first
full application of the NPC-aliveness systems to a single zone, and a template
for bringing the rest of the world to life.

## 2026-06-03 — Thornwall City wakes up too

The same treatment now reaches the trade city of Thornwall — the second town
brought to life with the NPC-aliveness systems:

- **The market keeps hours.** The market merchant, food vendor, apothecary,
  jeweler, and weaver each work their stalls through the day and close up at
  night; the guard captain walks his rounds. Come back at different hours and
  the market wears a different face.
- **They're connected.** Traders, kin, and rivals across the city now share a
  web of relationships, and idle neighbors trade short remarks drawn from those
  ties — the market square and side streets feel populated rather than staffed.
- **They talk about the city's troubles.** Word of Thornwall's public woes
  passes mouth to mouth, carried by the city's gossips. (Nothing that spoils a
  quest — just the open talk of the streets.)

Like the Stillwater pass, this is content-only: no new quests, no map changes.

## 2026-06-02 — The economy feeds itself (supply chains + market participation)

Shopkeepers no longer run dry. Several merchant types — cooks and enchanters
in particular — had no reliable way to replenish what they sold, so their
shelves stayed bare once the opening stock was gone. The world now supplies
them:

- **Cooks are real cooks.** The tavern cook, the food vendor, and the camp
  cook now prepare their dishes from ingredients, and the meat that feeds them
  comes from the world — foragers haul it in and corpse salvage yields raw
  meat and wild hare meat. Cooked meals restock as they are made.
- **Enchanters draw on alchemy.** Spoiled and salvaged potions break down into
  enchanting materials — cheap potions into binding paste, rarer and
  harder-to-brew potions into higher-tier chrysalis materials. Decayed stock
  from the town apothecaries flows into a shared reserve the enchanter draws
  from, so chrysalis supplies trickle back over time.
- **NPCs buy and sell.** Merchants and the townsfolk around them now take part
  in their own economy — selling surplus, with foragers topping up whichever
  vendors are emptiest from their storage caches.

Alongside the economy work:

- **Bosses wear their loot.** The elemental oasis champions were silently
  dropping nothing because their equipment slots were disabled — the queen
  never wore her crown. Their slots are open now, so their signature gear
  actually drops.
- **Combat in the dark stays dark.** Blinded, or in an unlit room without
  special sight, your prompt no longer reveals the name, health, or stance of
  your foe — information you should not have. (Matches the combat text, which
  already hid this.)
- **Faster growth where it lagged.** Dexterity, vitality, and charisma now
  improve a little more quickly through use.
- **Cleaner web login.** Web-client players no longer see a stray "Mudletmap
  not recognized" line at login.
- **Jails are secure.** Holding-cell doors are now locked — only an arrest
  puts you inside, and serving your time or paying your fine releases you just
  as before. No more wandering townsfolk turning up in the cells.

(Internally: 5.4 NPC market participation — `actions.Sell` lift + overstock
decay + globalized forager-chest backfill; cooking and enchanting supply
chains via crafter conversion, corpse/potion salvage, and a shared
enchanting-mat reserve; elemental species `DisabledSlots` cleared;
fight-prompt visibility gating via `messaging.CanSeeClearly`; per-stat
progression multiplier tuning; holding-cell exit locks.)

## 2026-06-01 — NPCs keep what they earn (goal-progress persistence)

NPC goal pursuit now *sticks*. The engine quietly unloads idle, empty rooms
to save memory and rebuilds them from templates when someone returns — and
until now that wiped whatever a mob had worked toward. A thief who sold
stolen loot to build up gold, or an NPC who saved up and bought better gear,
reverted to its starting state the next time its corner of the map respawned.

Now a mob's **gold, equipment, and in-flight goal plans persist** across that
performance despawn, so an NPC that works toward something keeps the result —
the thief stays richer, the upgrader stays better-armed. Death and
admin-despawn still reset a mob, so a foe you cut down comes back fresh; only
the routine memory-saving despawn preserves progress. (Old save data migrates
transparently — nothing to wipe.)

Two supporting fixes ride along:

- **Named NPCs now actually pursue their goals.** Townspeople and threats that
  run their own behavior scripts previously never reached the strategic-goal
  planner — only generic archetype mobs did — so much of the "NPCs pursue
  their own goals" feature was silently inert for the exact named characters
  it was meant for. They now run the planner like everyone else.
- **No more duplicate guards.** A scheduled guard whose post differs from its
  spawn point could appear twice ("ghost" guards). Fixed — a scheduled mob is
  now listed only in the room it is actually standing in.

(Internally: `MobInstanceData` gains gold/equipment/plan-state with
presence-pointer semantics + save-at-despawn in `removeRoomFromMemory`; the
idle handler dispatches the goal planner for per-mob-tree NPCs; and
`removeRoomFromMemory`/`Prepare` no longer double-list schedule-relocated
spawns. Aliveness 5.3 + goal-pipeline & ghost-guard fixes.)

## 2026-05-30 — Balance & quality-of-life tuning

- **Magic damage up ~30%.** `SpellDamageScale` 1.2 → 1.56. To keep area-effect
  spells from running away with the buff, every AoE damage spell's per-spell
  `damage_multiplier` was cut 20% (conviction-barrage, hemorrhagic-burst,
  hemorrhagic-wave, sparks, veil-rend) — net effect: single-target spells
  ~+30%, AoE only ~+4%.
- **Bigger crafting storage.** Component pouches now hold **3×** their old
  capacity (component pouch 60, artisan's satchel 120, master's component case
  225) and bandoliers **2×** (leather 12, reinforced 24).
- **Two crafting staples are now forageable.** Previously vendor-only,
  **pine pitch** now forages in forest biomes (e.g. the Fernway pines) and
  **hive fragments** forage in cave biomes (e.g. the Ironwind Steppe caves), so
  they no longer depend solely on shop stock. (A broader audit of every
  crafting material's obtain path is queued.)

## 2026-05-30 — Bounty hunters (5.2)

The law no longer waits for you to wander back into town. Rack up a serious
enough bounty — the kind a single heinous crime earns — and a **bounty hunter**
takes the contract. Word reaches you that you're being hunted; then a tough,
well-geared tracker sets out from the wronged town and closes in, room by room,
wherever you run. You can't simply hide.

Your ways out are real choices: turn yourself in and face **justice** (pay the
fine or serve the time — which clears the bounty and calls the hunter off, so
sprinting to a cell to get arrested is a legitimate, if humbling, escape), or
stand and **fight** the hunter. Beating one buys a reprieve, not absolution —
the bounty stands, and another will come. Falling to one settles the debt: your
record with that town is wiped clean, the hard way. A hunter will never strike
you down while you sit in a holding cell.

The hunter scales to how wanted (and how dangerous) you are, and wears gear to
match — gear that, on rare occasions, a victor might claim from the body.

And it cuts both ways: notable threats now carry **standing bounties** you can
collect. Check a bounty board (or `bounty list`) — the **Chrysalis Phantom**,
the bandit **Soren**, and others have prices on their heads, payable to whoever
brings them down.

This is NPC-vs-player pursuit, not PvP — no player ever hunts another player.
(Internally: `internal/bountyhunter` dispatch + a `hunt_bounty_target`
goal/planner + `bounty_hunter` archetype; reuses the 1.5 bounty, 2.8 tracking,
4.x goal, and 5.1 justice substrate.)

## 2026-05-30 — Town justice comes to Stillwater (5.1 rollout)

The watch now reaches a second town. **Constable Drunn** of Stillwater is no
longer just a quest-giver — he's a combat-capable lawman who enforces the law
the way Thornwall's guards do: warn, arrest, and (if you resist) fight. Wanted
characters caught in Stillwater are hauled to a new **holding cell** beneath the
lakeside constabulary, and — fixing a rollout gotcha — prisoners are now released
back into the *town that jailed them* rather than always to Thornwall. Arrest
messages now name the watch properly ("the Stillwater Constabulary") instead of
an internal label.

The town itself feels more aware: Stillwater's townsfolk, its wilderness
foragers, and the Ketil trading family are now recognized **citizens**, so
crimes committed in front of them are witnessed and reported — even out in the
marsh. (Citizens watch and report; only guards enforce.)

**Doing right by a town now pays off.** Completing civic quests in Thornwall and
Stillwater raises your **standing** with the relevant guards or townsfolk —
seven quests across both towns now grant reputation on completion, whether you
finish them through dialogue or by the deed itself.

(Internally: holding-cell and release rooms moved onto faction definitions
(`holding_cell_room` / `release_room`), boot-validated against real rooms, so
future towns are pure data — no code change. New `stillwater_guards` +
`stillwater_citizens` factions. Quest reputation is a new `QuestReward` field
applied in the completion handler. Stale guard warn-stamps are now swept
in-tick. Closes the two Town Justice 5.1 followups.)

## 2026-05-29 — Town justice 5.1c (arrest, jail & fines)

The watch no longer just kills you. Guards now **arrest** wanted characters
instead of cutting them down on sight. How that plays out is up to you: a new
**arrest policy** (`set arrest surrender|resist`, surrender by default) decides
whether you go quietly or fight.

**Surrender** and a guard hauls you off to a holding cell beneath the Thornwall
guard barracks to serve a sentence. You can wait it out, or buy your freedom
early — the cell door shows a **fine** that shrinks the longer you sit (`fine`
to check it, `payfine` to pay, drawn from your pocket first and your bank if you
come up short). Either way, answering for your crime **clears your record** with
that town: the crimes are resolved, any bounty the town posted on you is lifted,
and your standing with the guards *and* their citizenry resets to a wary
neutral. Wrong them again and the cycle starts over.

**Resist** and the guards fight you for real — and, thanks to the guard-combat
work shipping alongside this, that is now a genuine threat. While jailed you're
locked in: you can't walk out, flee, or recall away, and the guards leave you be
in your cell rather than pile on.

Help is in-game: `help justice`, `help arrest`, `help fine`. This is the third
slice of town justice (after guard enforcement and auto-bounties); paying a fine
or serving time now doubles as the way to clear your name. (Internally: an
`Arrest` severity rung in the `internal/justice` verdict; `internal/justice/
arrest.go` for the jail lifecycle; a Jailed buff with `no-go` + `no-aggro-target`;
knobs `ArrestResistGraceRounds`, `JusticeFineDecayPerRound`, `JusticeArrestRepReset`.)

## 2026-05-29 — Combat-capable town guards

Thornwall's guards can finally fight. They were built as non-combatant
quest-and-flavor NPCs, which meant the watch could threaten but never actually
trade blows — attack a guard and the fight would simply stall. The three active
Thornwall enforcers (the city guard, the gate guard, and Guard Captain Velk) are
now proper combatants: they hold the line, call for backup, and hit hard enough
to make crime in town a real risk. Velk keeps his quest-giver duties on top of a
captain's fighting tree. This is the groundwork that makes resisting arrest (and
the watch in general) mean something.

## 2026-05-29 — Town justice 5.1b (crime → auto-bounty)

Towns now put a price on serious offenders' heads. When a character commits an
identified **murder** of a townsperson, or their standing with a town faction
sinks to **hostile**, that faction automatically posts a kill-bounty on them —
scaled to the offender's power and the severity of what they did. The bounty
rides the existing bounty system, so guards treat a marked character as wanted
and other hunters can collect.

Those bounties resolve when the marked character dies: a town guard who lands the
kill funds the watch (the guard pockets the reward to spend on better gear down
the line), a third party who turns them in is paid out and earns faction favor,
and otherwise the bounty simply lapses. A character can't collect a bounty on
their own head. (Internally: `justice.MaybeDeclareBounty` fired from the crime
sites; a `PlayerDeath_BountyResolve` hook for attribution; knobs
`JusticeBountyExpiryRounds`, `JusticeBountyMurderMult`, `JusticeBountyRepMultMax`.)

## 2026-05-29 — Town justice 5.1a (guard enforcement)

The city watch now actually watches. Thornwall guards recognize **wanted**
characters — anyone with bad standing with the guards (or their allied
citizenry), an open faction bounty on their head, or a fresh unresolved crime
against a townsperson — and respond. A merely-disliked offender gets a verbal
warning to move along; ignore it and linger, or show up genuinely hostile or
with a price on your head, and the guards draw steel. Because a crime against a
citizen is recorded the moment it happens, a guard on the beat will turn on a
mugger within moments of the act.

Friendly, in-good-standing characters are treated as citizens and left in
peace, as before. This is the first slice of the larger town-justice system;
auto-bounties for repeat offenders, arrest, and ways to clear your name are
coming in later slices. (Internally: a new `internal/justice` "wanted" verdict
over the existing faction/crime/bounty data, fired by a per-round guard tick;
knobs `GuardWarnGraceRounds` and `JusticeCrimeLookbackRounds`.)

## 2026-05-29 — Mob aliveness 4.6 (goal satisfaction & pruning)

NPCs no longer hoard dead ambitions. The strategic goal layer (chunks
4.1–4.5) gained a cleanup sweep: goals that have been **achieved** (a
revenge target died, a wealth target was reached) or have **expired** are
retired, and goals that have gone permanently **unreachable** — a guardian
whose ward is long dead, a grudge against someone never seen again — are
abandoned once they've sat dormant past a threshold. This keeps NPC
decision-making sharp and stops mobs from fixating on impossible aims.

Internally: a throttled, per-mob prune sweep runs from the goals tick and
keys abandonment off how long a goal's relevance has been zero (tracked per
goal). Two config knobs tune it — `GoalPruneIntervalRounds` (50, sweep
cadence) and `GoalAbandonDormantRounds` (600, dormancy threshold).

## 2026-05-29 — Planar Oasis: all bosses + Elemental Princess

The Planar Oasis instance is bigger and meaner. The procedurally-generated
cube grew from 4×4×4 (64 rooms) to 5×5×5 (125 rooms), and instead of one
randomly-chosen boss per run, **all** of the elemental royalty now appear —
each wandering its own corner of the cube.

A fourth boss joins the king, queen, and prince: the **Elemental Princess**,
a water-form skirmisher who fights with a set of unarmed claws. She drops two
instance-scaled pieces — the **drowned claws** (a clawed weapon that trains
unarmed-combat) and a **tidal torc** for the neck. Like all instance gear,
both scale with the gold paid to open the instance.

## 2026-05-29 — Playtest bug fixes (sell feedback, companion gear, death buffs)

Three fixes from playtesting:

- **Selling to crafter shops no longer fails silently.** Merchant
  refusals ("I'm not interested in that.", "I can't afford that right
  now.", "I'm afraid I don't buy those.") and offer quotes were delivered
  through the merchant's asynchronous command pipeline (`mob.Command`),
  which on busy shopkeepers — scheduled townsfolk and autonomous crafters
  like blacksmith Kerra and apothecary Voss — could defer the line for
  many turns, long enough that the sale appeared to do nothing. The
  refusal path was the lone async outlier; every other sell outcome
  already reported synchronously. Refusals now speak immediately (`sell`
  and `offer` commands).

- **Companions can now wear (and show) a full set of equipment slots.**
  Summoned companions — flesh golem, the elementals (water/earth/air/
  fire/magma), skeleton, zombie, wraith, spectre, vampire, the steppe
  spirit wolf (canine), and the hive swarm (insectoid) — previously had
  most equipment slots disabled at the species level, so gear handed to
  them had nowhere to go and the `look` panel hid the slots. Those 13
  species now enable all standard slots. NOTE: canine and insectoid are
  shared with wild mobs, so wild wolves/dogs/insects now display an empty
  equipment panel when looked at (cosmetic only; they carry no gear). If
  that's undesirable, the spirit wolf / hive swarm can be split into
  dedicated companion species.

- **Potion (and other flagless) buffs now clear on death.** `Buffs.HasFlag`
  skipped any buff that declared no flags, so `CancelBuffsWithFlag(buffs.All)`
  in the death cascade left flagless buffs — pure regen potions like
  Healing Salve (buff 54) and the rest of buffs 54–60 — active across
  death/respawn. `HasFlag` now matches flagless buffs under the `All`
  wildcard.

Known unrelated: enchanting/cooking/general-store restocking under the
living-economy system needs a considered fix (tracked separately, not in
this batch).

## 2026-05-27 — Mob aliveness 4.5 (reactive goal generation)

NPCs now react to world events. Kill an NPC's friend (per chunk 1.6
relationship edges) and that NPC gets a revenge goal targeting you.
Steal from a mob and both the victim and room witnesses seed revenge
goals. Attack a non-hostile mob and the same revenge cascade fires —
plus, if you helped a mob in a fight by attacking another mob it was
fighting, the helped mob warms to you (positive opinion bump). Give
a mob an item it keeps and its opinion of you bumps up by a value-
tiered amount with a per-pair cooldown. Mobs killing other mobs now
bump faction-kill counters that satisfy the chunk-4.3 revenge-faction
predicate.

10 rules in the new internal/seeders/ package; 7 are live and 3 ship
as documented stubs awaiting upstream event-shape extensions (faction-
rep counter, quest-completion opinion, mastery-milestone). Three new
event types added: PlayerAttackedMob (shared by rules 6 and 9 —
multi-consumer payoff of the unified events architecture),
GiftOffered, GiftAccepted. The witness-of-theft and craft-materials
rules use direct-invocation rather than event subscription because
their triggers are action handlers / planner states without clean
world-event analogs.

Permanent-stuck-goal detection (now relevant since 4.5 actively seeds
goals into the world) is deferred to 4.6's prune sweep. Cross-type
goal conflicts still deferred per the 4.3 decision; seeders pre-check
obvious contradictions before seeding.

A documented follow-up: rule 5 (witness-of-theft) currently seeds
revenge into all room witnesses; in a future content pass (after 5.1
Town Justice), guards and civilians will split — some report the
crime instead of pursuing revenge personally.

No schema change. Player-facing impact: NPCs react meaningfully to
your actions in ways the earlier chunks only set up the substrate
for.

---

## 2026-05-27 — Mob aliveness 4.4 (strategic → tactical translation)

NPCs now actually pursue the goals 4.2 selects from 4.3's catalog.
Combat-capable mobs flee when HP drops below the survival threshold.
Thieves wander to vendors and sell loot when seeded with wealth-gold.
Named NPCs path to and attack revenge targets, defend protection
targets, walk among faction members for befriend / protection / revenge
goals. Foragers move toward unvisited zones for visit-zone goals.
Crafters seek stations to produce known recipes.

13 hand-authored per-type planners in the new internal/planners/
package, dispatched via one new behavior-tree action `try_goal_planner`
inserted into 18 non-boss archetype trees at the priority position
each archetype's author chose. Planners are pure Go functions returning
one command + status per tick; intermediate state lives in
mob.Character.MiscData under a `plan:<goal_type>:` key prefix that's
wiped automatically on goal switch.

Boot wiring registers planners.ClearPlanState as a callback into
goals.Recompute (mirrors 4.2's SetWeightsLookup and 4.3's
SetArchetypeDefaultsLookup patterns). Bridges the goals → planners
direction without an import cycle.

Permanent-stuck-goal detection (planner-perpetually-fails) is deferred
to 4.6's pruning sweep; reactive seeding of coexisting goals (e.g.,
craft-item triggers a wealth-item for missing materials) is deferred
to 4.5. Cross-zone pursuit, plan visualization admin command, and
schedule-aware planning are all out of 4.4 scope.

No schema change. Player-facing impact: noticeable NPC liveness.

---

## 2026-05-27 — Mob aliveness 4.3 (goal types catalog)

13 concrete goal types now register with the strategic-layer
substrate: survival, wealth-gold, wealth-item, craft-item,
revenge-mob, revenge-faction, protection-mob, protection-faction,
befriend, befriend-faction, mastery-skill, mastery-equip,
visit-zone. Each has a Predicate (when satisfied), ContextScore
(relevance multiplier), and — where multi-instance makes sense —
an AllowMultiple flag plus DedupKey func so the same mob can
hold goals against multiple targets without collapsing.

Engine deltas: declarative ParamSchema validation at Add time
(rejects malformed goals); AllowMultiple + DedupKey for
multi-instance types; archetype lazy-seed sentinel so default
goals seed once per mob template on first access and survive
admin Clear.

Sparse archetype defaults: every combat-capable archetype defaults
to a survival goal (kicks in when HP drops to ~25%); thieves and
shopkeepers add a generic wealth-gold goal. Mob-specific param
goals (revenge targets, befriend targets) arrive via 4.5 reactive
event hooks.

Substrate-only — chosen goals aren't wired into behavior-tree
execution yet (chunk 4.4). Observable change: `goal current <mob>`
now returns a real current goal for most loaded mobs; the
`goals.switch` debug log fires when survival kicks in during combat.
No player-facing change.

Note: the existing MobIdle gossiper system
(`buildGossipLine` in `NewRound_HandleIdleMobs.go`) is intentionally
untouched. A goal-driven directed-gossip mechanism belongs in a
future gossip-system refinement chunk, not 4.3.

---

## 2026-05-27 — Mob aliveness 4.2 (goal selection)

Adds the strategic-layer selection function over chunk 4.1's goal
substrate. NPCs now pick one current goal from their goal list per
round, weighted by priority, per-archetype multipliers, and an
optional per-type context-score hook. Hysteresis (margin + min-hold)
prevents goal-thrash.

Substrate-only — the chosen goal isn't wired into behavior-tree
execution yet (chunk 4.4's job). Two new admin subcommands surface
the selection state for inspection: `goal current <mob>` and
`goal scores <mob>`. A `goals.switch` debug log line fires per
strategic switch.

Config knobs in `Balance` (defaults conservative): `GoalSelectSwitchMargin`
(5.0), `GoalSelectMinHoldRounds` (100), `GoalSelectTickEnabled` (true).
Archetype YAML can now optionally carry a `goal_weights:` map (4.3
will start using it).

No player-facing change.

---

## 2026-05-26 — Mob aliveness 4.1 (goal representation substrate)

**No change to NPC behavior yet — this is foundations work.** Chunk 4.1
lays the substrate for the strategic layer: a typed goal store that lets
NPCs hold persistent, prioritized goals (revenge, wealth, protection,
survival, etc.) with satisfaction predicates and expiry. The store
persists per-NPC on disk, is concurrent-safe, and resolves conflicts by
priority when the same target receives two goals of the same type.

**Why it matters.** The current behavior tree answers "what do I do
*right now*?" A goal store answers "what do I *want*?" Without it,
NPCs react but never *pursue*. Chunk 4.2 (goal selection) will wire the
store into the behavior tick so NPCs actually act on what they want —
that's when the observable change lands. Chunks 4.3–4.6 build out the
goal-type catalog, strategic-to-tactical translation, reactive goal
generation (NPCs form grudges from world events), and automatic
satisfaction pruning.

**Admin tooling.** `goal list <mob>`, `goal show <mob> <id>`,
`goal add <mob> <type> ...`, `goal remove <mob> <id>`, and
`goal clear <mob>` are available for inspection and testing during
development. Visible only to admins; players have no new UI.

---

## 2026-05-26 — Aliveness 3.8 hotfix (caravan + forager + dashboard)

**Lars stops at every vendor now.** A one-line bug in
`StartOneshotPatrol` initialized the dwell counter to 0, so when
Lars arrived at his first vendor the executor immediately advanced
him to the next waypoint without dwelling or firing the
`caravan_vendor` arrival event. Same bug also made Tova appear to
"vanish" from her first vendor — she'd arrive and walk off without
selling. Fixed.

**Lars actually trades materials with every vendor.** Pre-fix the
cross-city pickup filter only accepted items whose bucket was in
the zone-source list. Iron ingot, steel ingot, coal dust — all
standard crafting materials — were skipped because their bucket is
`base` or `overlap`. Expanded the filter: any item flagged as a
component (and not a finished good — no weapons, armor, potions,
etc.) now qualifies for caravan cross-city distribution. Smith
Brindle's 80 steel ingots no longer sit untouched while Lars
walks past.

**Foragers don't accumulate cargo forever now.** The carry-cap and
fatigue threshold check that triggers a forager's "head back to
town" transition lived inside `actForagerStep`, but the YAML
archetype's inner selector short-circuited on every successful
forage attempt and never reached that code. Foragers in workable
territory therefore never transitioned out of Foraging. Extracted
the check into a new `forager_check_thresholds` btree primitive
that runs first in the foraging selector every tick.

**Forager state-machine integrity.** If a delivery patrol got
interrupted by HP-emergency recall or hit a path-retry budget
without producing a clean `PatrolCompleted`, the mob ended up at
sanctuary with a stale `PatrolId`. The patrol executor would then
read the stale id, see the mob at a terminal-shaped waypoint,
and fire a phantom `PatrolCompleted` from sanctuary. Cleared the
patrol id at every transition into the Recalling state. Also
removed the now-redundant sanctuary-fallback in
`tickForagerDeliveringTown` (the completion listener it was
guarding against being dead is no longer dead).

**Economy dashboard shows forager throughput now.** Two bugs in
`internal/economy/health/`: capture.go read throughput records
under `m.Character.Zone` (current room — mutates as the mob walks)
when the records are written under `m.Zone` (template-stable);
scoring.go compared snake-case territory keys against display-case
shop zone names with a case-sensitive prefix compare that was
always false. The dashboard's `FromForagers` column was always 0
by construction, regardless of actual deliveries. Both fixed.

**Lars walks shorter circuits.** Greedy nearest-neighbor reorder of
the Stillwater and Thornwall runner-circuit waypoints. No
mechanical change, just less zig-zagging.

---

## 2026-05-26 — Mob aliveness 3.8 (one-shot sub-patrols: caravan runner + forager delivery)

**Caravan runs are richer now.** Ketil's caravan rolls into Thornwall
or Stillwater depot, parks the wagon (with Hob and Bran the horses
and Marta the guard), and Lars (Ketil's son) walks the goods out
to each vendor while the rest of the crew rests. The wagon never
gets dragged into an alchemy shop again. What doesn't sell comes
back to the wagon when Lars returns.

**Foragers stop getting hung up on the delivery loop.** Marsh
(Tova) and Steppe (Halix) foragers now use the same patrol-layer
machinery for their vendor circuit — retry-then-home-fallback,
combat-interrupt-and-resume, and standardized waypoint dwell.
Fernway forager (Kessa) keeps her single-stop sealed-crate handoff.

**Under the hood:** new `loop_shape: oneshot` patrol mode, new
`events.PatrolCompleted` event, `mobs.StartOneshotPatrol` /
`ClearOneshotPatrol` runtime helpers. Caravan main route shrinks
from 22 waypoints to 4 (depots + Fernway pickups). Vendor stops
live entirely on Lars's runner-circuit sub-patrols. Lars gains a
strength bump so he can actually carry a wagon's worth of cargo
between depot and vendors.

**Looking ahead:** attacking the caravan crew or wagon will carry
severe consequences once Town Justice (chunk 5.1) lands — massive
Thornwall guard faction rep loss, murder records, the works.
Don't roleplay yourself into a permanent rep hole on the way to
the next bandit ambush.

## 2026-05-26 — Scheduled NPC pacing fix (chunk 3.6 follow-up)

**Townspeople stay where their schedule put them now.** Before this
fix, Kerra paced in and out of the tavern during her evening shift
instead of nursing her tankard, and Dal bounced between the main
floor and the back corner so often she never lingered with the old
men — which made the chunk 3.6 conversation pilot effectively
invisible.

**Root cause.** Two systems both claimed authority over "where this
mob should be" and fought each other every other tick. The schedule
executor (chunk 3.2) force-sets `MaxWander=0` while a schedule is
active to suppress wander. The legacy `MobIdle` displacement guard
then read that as "this mob must never leave home" and queued
`pathto home` whenever the mob's room didn't match its original
placement room — but `HomeRoomId` was set at spawn time and never
moved, while the schedule's segment target shifts hour by hour.
So a scheduled NPC at her current segment target was perpetually
"displaced" in the eyes of the legacy guard.

**Fix.** Scheduled and patrol mobs now opt out of the legacy
displacement guard. Their executors are the movement authority and
already re-path to the correct target on the next tick if displaced
— the legacy `pathto home` was redundant for these mobs and
actively harmful. Non-scheduled, non-patrol mobs (the cases the
guard was originally written for) still get the recovery behavior.

## 2026-05-25 — NPC↔NPC idle conversations (chunk 3.6)

**Townspeople chat with each other now.** Find your way to the back
room of the Thornwall tavern and you'll catch Dal the barmaid trading
small talk with the three old men who hold down the corner table —
Fen, Gobb, and Wrex. They complain about the weather, gripe about
their backs, gossip about wagons and priests, and (if you stick around
long enough) trade pointed jabs about an old argument between Fen and
Wrex. None of it advances any quest. It's just the world being lived in.

**How it works.** Conversations are drawn from a relationship-keyed
library at `_datafiles/world/dogmud/conversations/`. Two NPCs in the
same room with a relationship edge (friend, rival, family, lover,
employer, employee per chunk 1.6) occasionally roll for a 2-4 line
exchange. Each line is one game round. The first line picks A or B at
random and the second NPC takes the next, alternating until done. If
either NPC enters combat, walks out, or falls asleep mid-exchange, the
conversation aborts silently. Both NPCs cool down 50 rounds after a
complete exchange. Walking into a room with two eligible NPCs gives
the trigger a 25% boost — observed worlds are slightly louder than
unobserved ones, which is the point.

**Authoring shape.** Type pools live at `types/<relationship-type>.yaml`
(generic banter per relationship type). Per-pair overrides live at
`pairs/<lower>_<higher>.yaml` and stack on top of the type pool. An
optional `subtypes:` map on a type pool varies flavor by the
relationship's subtype string (`fond`, `estranged`, `professional`,
`bitter`). All three knobs live under `Balance`:
`ConversationBaseChancePct` (default 1.0), `ConversationPlayerArrivalBoostPct`
(default 25), `ConversationCooldownRounds` (default 50). Full schema
in `docs/schemas/conversation.md`.

**Retired: upstream GoMud's old conversation system.** The replaced
system was name-keyed (matching initiator/participant by mob name)
rather than relationship-keyed, lived in the same package path
(`internal/conversations/`), and was dormant in DOGMud — no DOGMud mob
authored conversation files; only 11 small Frostfang sample YAMLs
referenced it (rats squeaking at rats, beggars muttering at beggars).
Those samples are retired. The `converse` mob command is removed.
`MobConverseChance` config knob is removed.

**Architecture note: import cycle.** The replacement package needed to
sit next to `internal/mobs/` without forming a cycle (mobs imports
conversations for the loader). Solved with a small `MobConversant`
interface in conversations + a `internal/conversationadapter/` bridge
package that both `internal/hooks/` and `internal/usercommands/` use.
No production behavior change vs the alternative.

## 2026-05-25 — NPC schedules + sleeping mechanics (chunks 3.2 + 3.3)

**Townspeople have daily routines now.** Blacksmith Kerra, Tavern
Keeper Marek, and Temple Priest Olen in Thornwall City follow
hour-by-hour schedules. Kerra wakes in her new loft above the
forge at 6, hammers at the anvil 9-6 (with active crafting
output), drinks at the tavern 6-10pm, sleeps in the loft 10pm-6am.
Marek opens up at 10am after a slow morning in his quarters above
the tavern, works the long shift, sleeps upstairs. Olen rises
before dawn, runs dawn and afternoon prayers at the temple, joins
the regulars at the tavern for an evening cup, sleeps in his cell.
Three new "above-shop" home rooms added (Kerra's loft above the
forge, Marek's quarters above the tavern, Olen's chamber above
the temple). Find the smith outside of work hours and you'll need
to track her down somewhere else.

**Schedule infrastructure.** New schedule YAML format in
`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. Schedules
cover all 24 hours, swap idle commands per segment, steer the
NPC between rooms via existing pathto plumbing, and gate
crafter ticks via per-segment activity. Validators panic at boot
on coverage gaps, unreachable target rooms, or unresolved
references — pre-push boot-test catches authoring drift. Spawn
override places scheduled NPCs at the current segment's target
room on cold start and on respawn, so the world feels "already
in motion" at any hour. New admin command
`mob schedule <instId>` for debugging.

**Sleep is a real mechanic.** Players can `sleep` (no slash) in
any room. While asleep, all pools (health, stamina, conviction)
regenerate **five times faster** than normal. But sleep is
dangerous: the entire first round of attacks against a sleeper
**automatically critical-hits**. Sleeping in hostile territory
is an invitation to assassination. Wake by taking any damage, by
someone failing a steal or pickpocket against you, by a shout in
your room, by a player entering your room carrying a light source,
or by typing `stand`. NPCs follow the same mechanics: scheduled
NPCs at their sleep segments are asleep (visible as `(asleep)`
in the room's "Also here:" line), wake on the same triggers, and
re-sleep automatically after a grace cooldown if the segment is
still active. Crime severity uplift for attacks against sleepers
is deferred to a later chunk (Town Justice) — the data is already
queryable at crime-record time when that lands.

**Combat: damage piping + first-hit-crit.** The damage pipeline
gained a forceCrit parameter that bypasses the normal Z-score
threshold. `handleCombatRound` now snapshots victims with the
Sleeping flag at start-of-round so all attackers in the round
share the crit payoff before the cancel-on-damage flag clears
the buff mid-round. Other future first-hit-crit triggers
(surprise attack, backstab) can hook the same snapshot site
without further pipeline changes.

**Fixed: NPC double-spawn at home rooms.** Pre-existing latent
bug exposed by the new schedule system. When a scheduled NPC
walked away from their home room for hours (Kerra at her loft
overnight), the home room could unload from memory; on reload,
the spawn loop didn't recognize the still-alive scheduled NPC
and spawned a duplicate. The fix adds an orphan check: before
spawning, the engine looks for any live mob already matching
this room's home + mob id, and reattaches the spawn slot
instead of duplicating. Also resolves a related "crafter NPC
emits crafting messages outside their workshop" symptom — the
duplicate NPC had no proper schedule context and would craft
from wherever it was.

**Fixed: `.gitignore` was silently ignoring content.** The bare
`dogmud` pattern was catching every path with `_datafiles/world/dogmud/`
in it, requiring `git add -f` for every world content commit and
risking silent loss of new YAMLs. Anchored to `/dogmud` so it only
matches the binary at repo root, as intended.

## 2026-05-23 — End-of-day hotfix bundle + chunk 3.1

**SECURITY: web client login password no longer mirrored on screen.**
Two-part race condition fixed in the web client login flow. Part one:
the `TEXTMASK` command that switches the input box to password mode
was sent AFTER the password prompt, so anything typed before the
event-loop drained showed as plaintext in the input box. TEXTMASK is
now sent synchronously BEFORE the prompt. Part two: even with a
masked input box, hitting Enter echoed the typed buffer back to the
main MUD scrollback (separate code path from the input-box mask). The
submission-time echo now renders mask characters per byte instead of
the raw password buffer. Telnet clients were never affected (they use
per-character echo + IAC ECHO suppression). Web client users who
played in the last few weeks should consider changing their password.

**Heals no longer help your enemies.** Communion of Flesh and any
other `helparea` spell would heal the mob you were currently fighting
in addition to your allies — the dispatch code populated the player
target list but left the existing aggro-target mob in the heal target
list. Heals now only reach allies (charmed companions) plus players
in the room; hostile mobs are correctly skipped.

**Apex Ironwind drops now respect their declared rarity.** The
Windscour Wyrm and Stone Beetle Queen were dropping their Chrysalis
Core at 100% instead of the intended 5% / 10%. Carried items always
drop at 100% by engine design; per-item dropchance overrides that.
Both apex mobs now carry the core with explicit dropchance values
matching the mob's overall itemdropchance. (9 other mobs world-wide
have similar patterns; logged for content audit, not auto-changed
since some are intentional always-drops.)

**Mob surprise attack now does per-weapon strikes.** When a hidden
mob ambushes you, it now fires a surprise attack PER weapon wielded
— the same mechanic players have always had. Previously mobs only
got the backstab-crit promotion on a single swing. Most stealth mobs
in the world wield a single weapon so the difference is small; rare
multi-weapon stealth mobs got a meaningful damage boost on the
opening round. (Companion mobs benefit too.)

**Companions keep their gear through logout.** Items handed to a
companion, and gear they have equipped, now persist across logout
and login. Previously the gear was attached to the runtime mob
instance which was destroyed at logout. Saved state writes to the
character file alongside the existing companion progression. Note:
the `dismiss` command still destroys companion gear (release is the
explicit "lose the gear" path); only the implicit logout path is
fixed.

**Picking up gear no longer blocked by combat.** You can now `get`
items while fighting. Realism takes a hit (you'd probably keep
swinging, not pause to pick up a sword), but the alternative —
watching wandering mobs walk off with the gear from kills you just
made — felt worse. Other combat-time restrictions (drop, gearup,
equip) are unchanged.

**Sonic shout and toxic bite now respect the right defenses.** Both
mutations previously dealt raw stat-derived damage that ignored
defender mitigation. They now flow through the unified damage
pipeline: sonic shout is gated by Conviction mitigation (mental
resilience), toxic bite's bite damage is gated by Physical mitigation
(armor). High-mitigation targets take less; bare targets take roughly
what they took before. Toxic bite's poison DoT magnitude is
unchanged (DoT pipeline routing is a separate followup).

**Internal cleanup.**
- New `time_of_day` `range:` parameter on the btree condition for
  hour-precise gating (chunk 3.1, foundation for upcoming NPC
  schedules). Existing `period: day` / `period: night` binary form
  unchanged.
- Fold-recall now auto-looks the destination room on arrival (mirrors
  the existing death-respawn auto-look pattern).
- Leaderboards no longer include admin or AI-flagged accounts.
- 36 invalid biome typos fixed across 6 zones (`marsh`/`river`/
  `plains`/`ruins` → registered biome names). 4 warren NPCs in the
  Labyrinth of Low Tunnels reclassified from `aberration` species to
  `human` so they can wear gear properly.
- Orphaned `ForagerForageDwellRounds` + `keyForageTimer` config knobs
  deleted (left over from chunk 2.9's dissolve-cadence-into-YAML
  refactor; future devs tuning them would have seen no effect).
- Surprise-attack lifted to shared `actions.SurpriseAttack` helper so
  player and mob paths can't drift.
- `mobcommands/charge.go` delegates trip resolution to
  `actions.ExecuteTrip` instead of reimplementing the math.
- New `lock` and `unlock` mob verbs (standalone, since player keyring
  logic doesn't apply to mobs).
- New `try_any_active_mutation` btree primitive — mobs and companions
  that evolve new active mutations at runtime can now fire them
  autonomously, preferring rarer ones first.

**Deferred to followup chunks (known gaps):**
- Vendor backfill from forager chests — chests are deposit-only right
  now; the "overflow" promise still needs a chest→vendor flow
  mechanism. Critical.
- Poison DoT magnitude pipeline routing (toxic-bite's DoT half).
- Single-target mutation btree dispatch (`blinding-spit` +
  `toxic-bite` can't yet be fired from `try_mutation_active*`).
- Web client vitals bar doesn't visualize the reserved-pool portion
  yet (server-side reserved-pool getter + GMCP payload + client
  rendering needed).
- 9 other mobs world-wide with carried items + non-100
  `itemdropchance` may have the same drop-rate mismatch as the apex
  Ironwind pair; per-mob triage queued.

## 2026-05-23 — Mob Aliveness 2.10 Followups

**Foragers stash their overflow.** Tova, Halix, and Kessa now bring
their unsold goods home to a locked chest at the end of each vendor
trip — unlocking, depositing, and re-locking before settling in to
rest. Tova gained a new private reedwoven hut deep in Stillwater
Marsh (just north of Spring Pool) where her lockbox lives; Halix
and Kessa use the chests they already had in the steppe alcove and
fernway camp respectively. The chests are real locks: players with
picklock skill or a stolen key can break in, but doing so is theft
and the foragers don't forget.

**Companions and mobs use any mutation they've evolved.** A new
behavior-tree primitive lets mobs autonomously fire whichever
active mutation they currently have, preferring rarer mutations
first. This matters for companions: a companion who evolves a new
active mutation during play will start using it in combat without
requiring their archetype to be hand-edited.

**Sonic shout and toxic bite hit harder against the right defenses.**
Both mutations previously dealt raw stat-derived damage that ignored
target armor. They now flow through the unified damage pipeline:
- Sonic shout (Willpower-driven) is gated by Conviction mitigation —
  resistance comes from mental resilience, not physical armor.
- Toxic bite's bite damage (Strength-driven) is gated by Physical
  mitigation — armor matters now.
- Toxic bite's poison DoT magnitude is unchanged (DoT pipeline
  routing is a separate followup).

Net effect: damage magnitudes shift in both directions depending on
the target's mitigation. High-mitigation targets take less; bare
targets take roughly what they took before.

**Hidden mobs now strike with every weapon, not just one.** Mobs that
ambush from stealth previously got a single backstab-crit-promoted
swing on their opening round. They now fire a full per-weapon burst
from concealment, matching the player path exactly. Most stealth mobs
wield a single weapon so the difference is small; multi-weapon stealth
mobs (rare) get a meaningful burst on their first round.

**Internal cleanup.**
- Surprise attack lifted to `internal/actions/surprise_attack.go` —
  player and mob paths now share one implementation.
- `mobcommands/charge.go` delegates trip resolution to
  `actions.ExecuteTrip` instead of reimplementing the math.
- New `lock` and `unlock` mob verbs (standalone implementations;
  the player keyring concept doesn't apply to mobs).
- New `try_any_active_mutation` and `try_store_excess` btree
  primitives.
- New forager state `StateStoring` inserted between Delivering and
  Recalling for foragers with `storage_chest_room` configured.

**Known limitation flagged for future work:**
- **Forager chests are deposit-only right now.** Items go in but
  don't come back out — there's no chest-to-vendor flow mechanism
  yet. Without that, chests accumulate indefinitely and don't
  actually backfill vendor inventory the way the "overflow cache"
  design promised. A followup chunk will add one of four sketched
  solutions (forager-withdraws-on-next-vendor-trip is the leading
  candidate).

**Deferred to followup chunks:**
- Vendor backfill from forager chests (the missing other half of the
  overflow design — critical, high priority).
- Poison DoT magnitude pipeline routing.
- Single-target mutation btree dispatch
  (`try_mutation_active_at_target`).

## 2026-05-23 — Mob Aliveness 2.10: PvM/MvP/PvP/MvM Parity Audit

**Mobs can now use mutation abilities in combat.** All six active
mutations — blinding flash, blinding spit, healing gel, pacifism aura,
sonic shout, toxic bite — work identically whether a player triggers
them or a mob does. Companions inherit this naturally: any companion
with an evolved active mutation now uses it autonomously per its
archetype's combat AI. Mob archetypes opt in via a new behavior-tree
action `try_mutation_active`, which can list a single preferred mutation
key or an ordered fallback list (first available wins).

**Healing-gel re-balance.** The mutation was secretly using a flat
stat-derived heal (15% of Vitality) instead of the percentage-of-pool
heal the rest of the regen system uses. Corrected to 25% of HealthMax,
which scales with character power rather than just Vitality. For most
characters this is a small buff; for high-HealthMax / low-Vitality
builds it's a noticeable one.

**Sonic shout and toxic bite still bypass the unified damage pipeline.**
Surfaced during the lift but intentionally preserved verbatim — these
will be re-balanced in a future cleanup pass alongside other
pipeline-bypass cases. No observable change to current behavior.

**Sonic shout's "stun" is not actually a stun.** Documented as
prone-knockdown + caster self-deafen (the `ConditionStunned` referenced
in older help text doesn't exist as a code constant). No behavior
change — just naming/help clarity for future updates.

**Surprise attack confirmed to have full parity.** The audit initially
flagged this as a gap; verified that both player and mob paths
implement identical mechanics (per-weapon strikes from hidden state
when the special-move cooldown is available). Unifying the two parallel
implementations into one shared helper is queued as a refactor
followup.

**Companion gear loss reminder.** Heads up — when you dismiss a
companion, any gear they're carrying or wearing is destroyed. Strip
them first if you want to keep it. (Existing behavior; a helpfile
update is queued so this is clearer in-game.)

**Internal cleanup.**
- Six `mutation_*` commands lifted from `internal/usercommands/` into
  the shared `internal/actions/` package (~480 LoC deleted on the
  player side, ~120 LoC added on the mob side, robust shared preamble
  helper).
- Dead `selljunk` mob command deleted — registered but had zero callers
  in any YAML, hook, or AI code. First case study for the new
  "delete divergent verb" parity verdict.
- Full audit of all 119 non-admin player commands and 63 mob commands
  classified against a 6-bucket parity scheme. Tables embedded in the
  chunk spec.
- Runtime parity checker (`AuditCommandParity` startup hook) cleaned
  up: ~40 commands the audit identified as Orthogonal were never added
  to the allowlist; backfilled. 50 startup warnings → 3 (the three
  remaining are deliberate reminders of queued followup chunks).

**Phase 2 of the mob aliveness roadmap is complete** (chunks 2.1–2.10
+ 2.2a, 11 total). Tactical-verb vocabulary substantially closes the
gap between player and mob commands; remaining gaps are tracked as
followup chunks in MEMORY.md.

**Deferred to followup chunks:**
- **Forager locked-chest workflow** — extend forager NPCs to unlock
  their owned storage chest, place/remove goods, re-lock. Bundles the
  `lock` and `unlock` mob verbs.
- **Throwable mobs** — gated on a future "real ranged weapon system"
  design pass so projectile mechanics are designed once for both
  thrown and ranged.
- **Picklock parity** — wontfix. The interactive minigame is intentional
  player-only design; mob lock-bypassing will use a different model when
  needed.
- **Runtime-evolved mutation btree dispatch** — currently a mob (or
  companion) that evolves a new active mutation at runtime won't fire
  it until its archetype's `try_mutation_active` node lists the new
  key. Three fix paths sketched in the followup memory entry.
- **Surprise-attack unification refactor** — extract a shared
  `actions.SurpriseAttack` helper so the player and mob paths can't
  drift.
- **charge.go trip-math duplication** — mob `charge` reimplements trip
  resolution instead of delegating to the shared `actions.ExecuteTrip`.

## 2026-05-22 — Mob Aliveness 2.9: Mob Forage + Salvage

**Foragers join the unified action pipeline.** Tova (Stillwater Marsh),
Halix (Ironwind Steppe), and Kessa (Fernway South) — the three routine
forager NPCs — now run their per-tick forage roll through the same
`actions.Forage` entry point that the player `forage` command uses.
Previously each forager mob had its own per-mob behavior YAML driving
a Go state machine with private foraging logic. Now they share a single
`forager` archetype whose YAML composes `try_forage`, `try_salvage`,
and `wander_territory` btree primitives. The multi-state daily cycle
(Resting → Traveling → Foraging → Delivering → Recalling) stays in
Go via `forager_step`; only the per-tick Foraging loop dissolved into
YAML.

**Salvage is a real verb on the mob side.** The existing mob
`salvage corpse` mobcommand always worked, but its corpse-finding and
yield-rolling logic was its own self-contained code path that didn't
share anything with the player's multi-round Activity-machine version.
Both paths now converge on a new `actions.Salvage` single-tick core.
Player wrapper retains the multi-round CraftingState scheduling (so
the progress UX is unchanged); each per-tick resolve calls into the
shared action. Mob path calls it directly. The `salvage_returns` and
recipe-reverse-lookup yield math is now the same code regardless of
who's salvaging.

**Forage and salvage as btree primitives.** Strategic NPCs can now
compose `try_forage` and `try_salvage` in their archetype trees
without re-implementing the gathering logic. The `try_forage`
primitive returns Success on item found, Failure on miss / cooldown
/ wrong-biome. The `try_salvage` primitive defaults to "first eligible
corpse in room"; an `item_uuid` parameter overrides to specific-item
mode. The `wander_territory` primitive delegates to the existing
forager profile's territory-neighbor logic so a foraging mob still
respects its assigned patrol bounds.

**Latent bug fix bundled in.** The previous mob salvage path emitted a
"The X corpse is no longer here." message when a player's targeted
corpse disappeared between activity start and the final tick. The
initial Task 4 refactor dropped that message; the follow-up restored
a cause-agnostic equivalent ("You can no longer find the corpse you
were working on.") so multi-round salvage activities always close
with player-facing feedback.

**Internal cleanup.** `tickForagerForaging` (~40 lines of Go) and
`npcAttemptForage` (~35 lines) deleted from `behaviortree/actions_forager.go`.
The fatigue counter and carry-cap transition triggers moved to the
top of `actForagerStep` as state-entry guards. Three per-mob behavior
YAMLs removed. ~62 net lines deleted from the state machine code.

**Known deferrals** (logged as followups in MEMORY.md):
- Forager fatigue cadence now ticks only when `forager_step` is
  invoked (full-fallthrough of the YAML foraging selector), not on
  every Foraging-state tick. The 600-round watchdog still prevents
  runaway state. Observable cadence change for foragers in workable
  territories — they may stay in Foraging longer than before. Fixable
  via a dedicated `forager_foraging_tick_bookkeeping` primitive if
  smoke reveals impact.
- The `ForagerForageDwellRounds` config knob and `keyForageTimer`
  constant are now orphaned (written but never read). Mark deprecated
  or remove in a cleanup pass.
- The restored corpse-vanished message is generic — the mob's name
  is no longer threaded through. Wire `SalvageOptions.TargetCorpseName`
  in a follow-up if the specific message matters.

## 2026-05-22 — Mob Aliveness 2.8: Scout / Track / Scan

**Scouts that actually patrol.** Goblin scouts on the Ironwind steppe
now sweep adjacent rooms each idle tick, spot approaching travelers
one room out, and close the distance to engage instead of standing
frozen until you walk onto their tile. The new `scout` archetype runs
a five-branch loop: panic-flee at critical HP, self-defense, search
the current room for hidden threats, scan adjacent rooms for hostiles,
or pursue a fleeing aggro target across room boundaries via active
tracking.

**Three new mob-callable verbs.** `scan`, `track`, and `search` are
now first-class behavior-tree actions — mobs can peek into adjacent
rooms, read visitor trails, or sweep the current room for hidden
entities. The player-side commands lift the same logic into a shared
`internal/actions/` layer, so every behavior is symmetric between
player and mob paths.

**Lookouts get scan-before-ambush.** Bandit lookouts, tunnel watchers,
goblin sentries — any `lookout`-archetype mob — now sees you coming
one room out via `try_scan` and calls for help before you arrive. The
existing `player_enter` ambush still fires; the new branch just lights
up earlier.

**Thieves get silent stealth detection.** A thief that's about to lift
your coin purse first checks the room for hidden rivals via
`try_search`. The detection is **completely silent** — no flee, no
call for help, no behavioral leak — because betraying the detection
would contradict the scout-only-awareness contract. The thief gains
internal soft-target awareness and continues normal thief behavior;
you can't tell from outside whether you've been spotted.

**Leaders chase fleeing aggro targets.** Pack alphas and bandit
chiefs no longer give up at the room threshold. When a target flees
combat, the leader's idle branch runs `try_track` to read the
adjacent-room visitor trail, then `move_toward_tracked` to pursue.

**Active Tracking is a real buff now.** Previously the player `track`
command silently applied buff 26 (Conviction Surge — a +15 strength
combat buff) as a duration token, gifting any player running
forensic recon a free damage bonus. Authored a dedicated buff 86
(Active Tracking, 25-round duration, no statmods) and migrated the
four AddBuff sites + six RemoveBuff sites away from the misuse.

**"Tracking forever" bug fixed.** The room-description renderer in
`roomdetails.go` was gating only on misc-data, never on the buff
itself. When the buff expired or was removed, the misc data persisted
and the "Tracking X… they went north" line kept firing indefinitely.
Added a `HasBuff(86)` outer gate that clears stale misc data on next
room view, plus symmetric cleanup at every existing RemoveBuff site.

**Shadow lifecycle.** Buff 87 (Shadowing) is a sister to buff 86 —
25-round duration, applied on a successful `shadow <target>` and
removed on stop, spotted, or natural expiry. The shadow auto-follow
consumer in `go.go` now gates on `HasBuff(87)` so a stale shadow
state can't drag the player to a phantom destination.

**Mob-target shadow finally works.** Chunk 2.7 added the shadow verb
but only wired auto-follow for player targets — shadow on a mob
applied the buff and set misc data, then went silent when the mob
moved. New `MobRoomChange_ShadowFollow` hook closes the gap: when a
shadowed mob moves, any hidden shadower in the old room auto-moves
with them and gets the same post-move spotted-check as the player-
target path.

**Universal escape gates.** Two new hooks (`MobDeath_TrackingCleanup`
+ `PlayerDespawn_TrackingCleanup`) clear tracking/shadow misc-data
and buffs 86/87 from every character pointing at a dying mob or
leaving player. No more phantom tracking on a corpse or a logged-off
friend.

**Stealth-detection messaging dropped.** When a hidden mob spots
you sneaking into their room, you no longer see "X notices you" —
that message leaked the mob's name when the mob was itself hidden to
you. Same drop for "You no longer feel sneaky" — if you can't see
who spotted you, you can't really know you've been spotted from your
own POV either. The Hidden buff just disappears from your conditions
display. Room observers still see the sneaker visibly emerge.

**Cause-agnostic buff end text.** Buff 86's "The trail grows cold;
your focus slips" and buff 87's "Your quarry slips from view; your
focus breaks" both implied a specific cause (the target got away)
even when fired on timer expiry next to a still-present target. Now
both read "Your focus on the trail/quarry breaks" — fits every
cleanup path (timer, target found, target died, manual stop).

**Thief no longer summons the city guard.** The thief archetype's
`search-before-steal` graft originally called for help on detection,
which for Thornwall mobs resolved via faction allies to the city
gate guard — narratively wrong, since bandits don't summon law
enforcement. The branch is now silent search only.

**Internal cleanup.** `track.go` and `search.go` use the shared
`CalcSearchScore` helper instead of inlining the Perception +
SkillMultiplier formula. The `combat` package import was dropped
from both files as a result.

**Known deferrals** (logged as followups):
- `has_aggro` btree condition doesn't exist; the scout and leader
  archetypes use scan-based detection instead of a direct "do I
  have aggro" gate. Adding the condition would let archetypes
  branch more precisely on combat-state.
- `mob_target_lost` event doesn't exist either; combat-aggro-lost
  is detected indirectly via the scan/track chain on `mob_idle`.
- Faction-rep-based hostile determination on the mob side falls
  back to "any non-charmed player counts as hostile" — proper
  per-faction hostility checking is logged for a future pass.

## 2026-05-20 — Chunk 7: Centralized messaging framework

**Every player-facing line of text now flows through a single
pipeline** (compose → normalize → anonymize → color → wrap →
deliver). The chunk-6 Perception state machine (shipped dormant last
week) is now its consumer. ~2300 broadcast callsites across the
codebase migrated to the categorized API.

**Combat narration colored by weapon and outcome.** Each swing carries
its own color band based on the weapon's damage type. Warhammers and
maces render in warm orange-brown; swords and spears in dusty rose;
claws and bites in dark red-brown; bows in light sky blue; wands and
staves in lavender; fists in sandy brown. Defenses are in greens:
dodge (leaf), parry (lime), block (forest). The 6-armed test
character with four different weapons sees four different colors per
round.

**Spell colors by school.** Cast prep and fold lines render in pale
steel-cyan. Resolves route by the spell's declared school: elemental
in red (fire / ice / lightning damage), enhancement in warm gold
(buffs / shields), mental in lavender (charm / illusion), vital in
sage mint (heal), manifestation in rose pink (summon). Disruptions
and backfires render in warning amber.

**Progression banner.** Skill and stat advancements now render as a
boxed banner — `SKILL ADVANCEMENT` / `STATISTIC INCREASED` centered
on a 64-column rule, with a `from-tier → to-tier` line on the rare
quality-band crossings. Replaces the old `*** You feel your X skills
sharpening! ***` one-liner.

**Style auto-corrects on every broadcast.** "A aggressive" → "an
aggressive", "damage damage" → "damage", missing periods auto-
appended, sentence starts auto-capitalized for combat prose.

**Per-player line wrapping for tabular displays.** `set linewidth N`
(range 40–240, default 80) matches your terminal. The MOTD banner
and inbox separator track this; full table renderers (status,
inventory, who, help) honor it for Go-side widths, with templates
still drawing 80-column boxes.

**Infrared observers see "a figure".** Players using infrared vision
in a dark room now see anonymized red shapes instead of named
characters — names and pet names are stripped from the visual feed
before delivery.

**Username retune.** Username (yellow → cool blue 153), mobname
(cyan → warm tan 180), petname (orange → teal-cyan 108).

**Bug fixes shipped alongside the framework.**
- Companion-name leak: a player's pet name no longer appears in feeds
  of blind or dark-room observers who shouldn't be able to see it.
- Grapple-position name leak (sibling fix): grapplers' names also
  route through the sight gate properly.
- Sneak persistence: stepping into sneak no longer immediately fires
  "You no longer feel sneaky" — buff #9 now persists per the
  Awareness-FSM lifecycle, removed only when the FSM transitions out
  of Hidden.
- Mob corpses + loot drop reliably: a registration-order bug between
  the Death observers caused mob instances to despawn before the loot
  / corpse drop fired. Consolidated into a single observer with
  explicit ordering.
- Room descriptions preserve their side-by-side minimap layout: the
  pipeline's wrap stage no longer wraps pre-laid-out template
  content.

**Internal cleanup.** Duplicate `canSeeInRoom` helpers consolidated.
`sendRoomTextDarknessAware`, the naive byte-count `wrapText`, and
all `SendTextLegacy` / `SendTextVisualLegacy` shims deleted.

**Known deferrals** (logged as followups):
- Tabular templates (status / inventory / who / help boxes) still
  draw 80-column borders even when `set linewidth` is wider.
- Pre-T9 IsQuiet / Deafened filter logic on the `RoomId` events
  branch may be partially bypassed by the per-recipient fan-out;
  affected callers are out of scope for this chunk.
- Standing-combat moves (trip / kick / bash) can fire from grapple
  positions and emit a hit message but no actual state change. Needs
  position-aware variant routing like kick already has (stomp on
  prone, knee on grappled).

The combat-state-machines arc that began with chunk 0 (2026-05-13) is
fully wired up: chunks 0-6 built the substrate (Combat Phase,
Awareness, Life, Activity, Position, Presence, Perception); chunk 7
ties it together and is what the player sees.

## 2026-05-19 — Chunk 6: Perception state machine (dormant)

**Internal-only change.** The engine now tracks every character's
visual state — Sighted or Blinded — as a proper state machine. Today
no gameplay surface uses this; existing dark-room and blindness
handling behaves exactly as before. The primitive is in place for a
future broader messaging-framework upgrade that ties together
color-coded combat text, infrared "red shapes" rendering, line
wrapping, and the long-standing bug where messages can leak through
blindness or darkness with character names visible.

The combat-state-machines arc that began with chunk 0 (2026-05-13)
is now complete — six FSMs (Combat Phase, Awareness, Life, Activity,
Position, Presence, Perception) all shipped. Mob aliveness substrate
work can resume.

## 2026-05-19 — Chunk 5: Presence state machine

**Cleaner AFK and idle handling.** The engine now tracks every
character's "presence" — whether they're actively in the world, idle,
AFK, or disconnected — through a dedicated state machine instead of
scattered checks. Functionally identical for most cases. The one
visible change: an AFK player in a dangerous room can STILL be
attacked (intentional — going AFK in a dangerous place was always a
risk).

**Mob hibernation.** Mobs that have been bored for a while (no players
nearby for a stretch) now go Dormant — they skip their per-round tick
to save engine work. The moment a player enters their room or attacks
them, they wake up to normal Active behavior. Shopkeepers, foragers,
caravan crew, and charmed companions never go Dormant — they're
exempt from idle-out so the living-economy systems keep running
smoothly.

**Quieter sunset.** Legacy ManualAFK, AFKMessage, BoredomCounter, and
PreventIdle fields are gone, along with the MaxMobBoredom config knob.

Chunk 5 closes another step of the combat-state-machines arc; chunk 6
(Perception) is what remains.

## 2026-05-19 — Chunk 4f: Position balance + smoke

**Spell disruption in grapples is now Willpower-mediated.** Previously,
being knocked prone or grappled automatically broke any spell you were
casting. Now, your Willpower mediates a per-round concentration check
— a strong-willed caster can sometimes complete a spell from
underneath, while a distracted one rarely will. The hardest positions
(crucifix, back mount) remain brutal disruptors; the most lenient
(guard from underneath, where your hands are free) gives high-Wil
casters a real fighting chance.

**Comprehensive position-system smoke** across grapple entry,
advancement, dominant-position striking, eat/drink restrictions,
third-party hooks, AI tiebreaker, submission interrupt, and helpfile
language. Followup polish items logged for future chunks.

**Helpfile coverage audit** across grapple, cast, attack, submission,
flee, prone, stand, trip, bash, and related help topics. Removed
mechanical-value leaks (raw formulas, percentage thresholds, cooldown
round counts); tightened language wherever chunk 4f's chance-based
disruption invalidated older wording.

Chunk 4 (Position) is now closed.

## 2026-05-16 — Rich-grapple system live: chunks 4a + 4b + 4c

Three Position-FSM sub-chunks shipped end-to-end on
`feature/mob-aliveness-1.3-crimes` in one day. The 14-state
Position machine + per-round control-axis drift + weapon-reach
utility together make grappling tactically deep: positions
develop over rounds, control shifts based on opposed rolls, and
weapon choice in a grapple actually matters. Branch is merged
to `development`, not yet pushed to prod.

### 4a — Position FSM scaffold (DORMANT)

Built `internal/state/position/` on the chunk-0 framework as the
14-state taxonomy underlying the rich-grapple system. Ships
dormant: zero behavior change, legacy `CombatPosition` enum + all
command writers/readers untouched. 4b cuts over.

- 14 geometric states: Standing / Prone / Supine / Clinch /
  BackStanding / Mount / SideControl / KneeOnBelly / NorthSouth /
  Crucifix / BackGround / HalfGuard / Guard / Turtle. Prone/Supine
  split intentionally — submission paths, recovery difficulty,
  and back-take vulnerability diverge.
- Per-state data: `StandingData` (empty), `ProneData` /
  `SupineData` (Reason + MinRecoveryRounds + KnockdownSource),
  shared `GrappleData` (Reason + Partner + ControlLevel) across
  the 11 grapple states.
- ~75-edge transition graph, 22 trigger constants, 19 Character
  predicates (`IsStanding` / `IsProne` / ... / `IsOnFloor` /
  `IsGrappling` rollup), 10 btree primitives, Life-Dead cascade
  observer (`internal/hooks/Position_Cascades.go`).
- `ControlLevel` enum (Neutral / InControl / LosingControl /
  BecomingControlled / Controlled) reordered so Neutral is
  iota=0 — Go's zero value defaults match the spec (catches
  literals like `GrappleData{Partner: ref}` without explicit
  ControlLevel assignment).
- Behavior Matrix PO-001 through PO-045 PASS or SKIP per
  chunks-0-3 convention.

Spec at `docs/superpowers/specs/2026-05-16-state-chunk-4a-position-fsm-design.md`,
plan at `docs/superpowers/plans/2026-05-16-state-chunk-4a-position-fsm.md`.

### 4b — Position control axis + writer/reader cutover

Lit up the 4a scaffold. Cut over all 11 command-site writers
and ~25 reader sites from the legacy `CombatPosition` enum to
the new FSM, added per-round opposed control rolls with stamina
+ encumbrance penalty curves, threshold-triggered position
transitions, gradient/transition/stamina messaging, 6 new btree
control-axis primitives, and a periodic consistency checker.
Sunset the legacy `CombatPosition` enum + `PositionRoundsMin` +
`GrappleControllerId` + `ConditionGrappleController` +
`internal/characters/combatposition.go` at the end. Net delta:
-169 lines across the sunset commit (`6a9697d5`).

- **Per-round control mechanics:** `Position_GrappleTick.go`
  fires per round for every grappler. Opposed Strength +
  Unarmed-combat roll scaled by stamina + encumbrance penalty
  curves. Margin → control-level delta via `MarginToDelta`
  capped at ±1 per round. Two-consecutive-Controlled gate
  prevents single-round flukes from triggering position
  downgrades.
- **Asymmetric stamina cost:** controller pays less per round
  than controlled side (encourages opportunistic top-control
  play instead of immediate submission attempts).
- **Threshold transitions:** Controlled → `DefaultEscapeTarget`
  (Mount → HalfGuard, etc.). Pair-coordinated via
  `TransitionPair`.
- **Messaging contract** (`Position_Messaging.go`,
  per-grapple-cooldown-gated): gradient messages on control
  shifts, transition messages on position changes, stamina
  warning when resource at risk. YAML config + templates load
  at boot from `_datafiles/world/dogmud/grapple-messages.yaml`.
- **6 new btree control-axis primitives:**
  `mob_is_in_control`, `target_is_being_controlled`,
  `mob_low_grapple_stamina`, `target_low_grapple_stamina`,
  `mob_position_threshold_winning`,
  `mob_position_threshold_losing`. Together with the 10 from
  4a, 16 total position primitives end of 4b.
- **4 formal pair invariants** enforced via `TransitionPair` +
  tested via `ValidateGrapplePair` + backstopped by periodic
  `Position_ConsistencyCheck` observer that scans live grapple
  pairs every N rounds and force-breaks any pair that drifts
  out of invariant.
- **Sunset:** `CombatPosition` field, `PositionRoundsMin`,
  `GrappleControllerId`, `ConditionGrappleController` constant,
  `internal/characters/combatposition.go` (legacy enum +
  `IsGroundPosition` / `IsGrapplePosition` /
  `GetSpeedMultiplier` / `GetPositionColor` / `GetWorstPosition`
  helpers). Per-state data slots (`ProneData.MinRecoveryRounds`,
  `SupineData.MinRecoveryRounds`) replace the legacy fields,
  with a new `Position.ExtendRecoveryRound()` helper for stomp.
  `c.IsController()` derives from `GrappleData.ControlLevel`.
- **Smoke debugging** surfaced a long-standing shared-state-
  machine bug in `mobs.newMobByIdInternal` — `mob := *m`
  shallow-copied the template Character including pointer-typed
  Life / CombatPhase / Position / Awareness / Activity
  machines AND the `combatPhaseWired = true` guard. Every
  spawned instance shared the template's machines; observers
  wired on the template fired with the template's `*Character`
  (`MobInstanceId=0`), so the mob despawn cascade silently
  skipped. Fix: `Character.ResetForMobInstance()` nils the
  machine pointers + clears the guard after the shallow copy,
  so the next `Validate()` builds fresh per-instance machines
  and re-fires `OnCharacterCreated`. Saved as a class-of-bug
  memory (`feedback_shallow_copy_shared_pointers.md`) since
  this pattern is easy to repeat anywhere the codebase clones
  a Character / Mob struct.
- **Mid-flight fixes** during the smoke loop: grapple pair
  atomic break on death, `Character.userId` seeded on
  `LoadUser` and `LoginUser` (was 0, causing `TransitionPair`
  to fail with `ErrPartnerRequired`), prompt `{pos}` token
  sourced from CombatPhase not legacy Aggro, 2-consecutive-
  Controlled gate added, grapple-tick drift capped at 1 per
  round, hidden defenders forced Visible in
  `handleCombatRound`, highwayman idle emotes stripped (were
  breaking stealth on hostile lookouts), silent flee-block /
  cornered paths got messaging, flee blocker resolution
  refactored to shared `combat.ResolveFleeBlockers` helper.
- **Behavior Matrix:** PB-001 through PB-080 across
  `position_test.go` and per-package integration tests. Mix
  of PASS / SKIP. Chunks 0-4a regression clean. 176-test
  position package suite green.

Spec at `docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md`,
plan at `docs/superpowers/plans/2026-05-16-state-chunk-4b-position-control-axis.md`.

### 4c — Position × Weapon Utility (reach model)

Made weapon choice in a grapple matter. Single `Reach float64`
(meters) field on `ItemSpec` plus a default-by-subtype lookup;
position radius curve in the combat package; `radius / reach`
formula floored at 0.15 wired into the per-swing damage path.
Bladed weapons (Slashing / Cleaving / Stabbing / Shooting) in
grapples render with Bludgeoning vocabulary so the fiction
tracks the damage penalty. Phase-1 YAML migration is zero by
design — every existing weapon inherits its subtype default;
per-item overrides land post-smoke as balance feedback comes in
(`lake-iron-hook-spear` was the first such override — Stabbing
default 0.3m was wrong for a 2.0m spear).

The cheap chunk between heavyweight 4b and the upcoming 4d.
No FSM changes, no new btree primitives, no sunsets. ~12 file
touches, ~400 LOC production code.

- **Reach taxonomy** in `internal/items/reach.go` covers all
  in-engine weapon subtypes. Natural attacks 0.1–0.5m (fist
  0.1, claws/bite 0.15, sting 0.2, slam 0.3, gore 0.4,
  whipping 0.5). Melee: Stabbing 0.3, Cleaving 0.9, Slashing
  1.0, Bludgeoning 0.8. Shooting 1.0 (melee-fallback). Caster:
  wand 0.4, sceptre 0.6, staff 1.5. Authors leave reach empty
  for normal items (subtype default applies); override per-item
  only for outliers.
- **Position radius curve** (`internal/combat/reach.go`):
  Standing / Prone / Supine / Turtle unbounded (no penalty).
  Clinch + BackStanding share standing-grapple radius (0.5m).
  The 8 ground-grapple states share ground-grapple radius
  (0.3m). Three new balance knobs:
  `ReachStandingGrappleRadius`, `ReachGroundGrappleRadius`,
  `ReachUtilityFloor`.
- **Pipeline integration:** `CalcReachAdjustedItemMult(weapon,
  attacker)` composes weapon `damage_multiplier` with the
  reach utility. Single call site in `combat_helpers.go:
  buildWeaponSetup`. Kick variants route through the same
  helper; grapple/trip/bash stay reach-agnostic (force-driven,
  not weapon-driven).
- **Bludgeon narration:** when `ShouldBludgeon(reach, radius)`
  fires for a bladed weapon, the attack-message subtype swaps
  to Bludgeoning at the GetAttackMessage site in
  `buildAttackMessages`. Caster (Wand / Sceptre / Staff) and
  natural-blunt (Fist / Claws / Bite / Sting / Slam / Gore /
  Whipping) subtypes exempt. Existing Bludgeoning templates
  use `{itemname}` interpolation, so "You bash the iron
  longsword's pommel into the bandit" renders without bespoke
  pommel-strike vocabulary.
- **Smoke-verified in production** (twice — first session
  confirmed clinch bludgeon swap; followup with new test
  fixtures confirmed warhammer no-swap + wand/staff caster
  exemption):
  - Longsword standing: full slashing vocab + damage
  - Longsword in Clinch: "CRITICAL BASH... steel longsword"
    (bludgeon swap firing)
  - Warhammer in Clinch: "CRUSHES" — identical vocab to
    standing (Bludgeoning subtype exempt from swap)
  - Wand in Clinch: "CRITICAL ARCANE STRIKE" + "jab... willow
    wand" (caster exemption — vocab preserved)
  - Staff in Clinch: "crash your oak staff" (caster
    exemption, staff channel preserved)

Spec at `docs/superpowers/specs/2026-05-16-state-chunk-4c-position-weapon-utility-design.md`,
plan at `docs/superpowers/plans/2026-05-16-state-chunk-4c-position-weapon-utility.md`.

### Smoke followups (post-chunk-4c)

Three small additions surfaced as the smoke tests played out:

- **`help attack` reach section.** The chunk-4c T9 doc pass
  edited `attack.template`, but `attack.md` (also present in
  the dogmud helpfiles dir) shadows the `.template` in the
  engine's help-lookup order. Edited `attack.md` directly to
  add the reach + grapple paragraph alongside the legacy
  chance-to-hit / crit-chance prose. Grep confirmed `attack`
  was the only chunk-4c-touched helpfile with both `.md` and
  `.template` siblings.
- **`mob heal [instId]` admin command.** Restores a mob's
  Health / Stamina / Conviction to max so testers can run
  sustained combat against an otherwise-killable mob to
  observe multi-round mechanics (grapple drift, weapon-reach
  narration, per-round control cascades). Two forms:
  `mob heal <instId>` heals a specific mob, `mob heal` (no
  id) heals every mob in the room. Permissioned under
  `mob.spawn`. Helpfile updated at
  `_datafiles/.../admincommands/help/command.mob.template`.
- **`training_post` mob (id 9005, Test Arena zone).** Hostile,
  brawler AI, 5000 max HP / 5000 stamina, vit-100, sharp
  stick. Hits back so combat doesn't end early but pulls
  every punch — designed for sustained combat-mechanic
  observation. Spawn with `mob spawn 9005` or
  `mob spawn training post`. Replaces the need to repeatedly
  spawn sparring partners that die in 4-6 rounds against an
  over-leveled smoketester.

### Roadmap status

Chunks 4a, 4b, 4c all Done in `COMBAT_STATE_ROADMAP.md`. Next
sub-chunk in the rich-grapple series is **4d — Submission
rework**: opportunistic submissions gated on (Position,
ControlLevel), submission outcomes (choked, damaged limb, tap,
continue), rework/sunset the current `submit` special-attack
command. Mob aliveness work stays paused for chunks 4d-6.

---

## 2026-05-12 — Phase 2 tactical: chunks 2.4 + 2.5 + 2.6

Three Phase-2 tactical chunks shipped on
`feature/mob-aliveness-1.3-crimes` in one day (14 / 41 done).
Branch carries chunks 1.1–2.6; merged to `development`, not yet
pushed to prod.

### 2.4 — Mob `consider` + threat-aware behaviors

Consolidated the `consider` math via the actor pattern (mirroring
chunk 2.1's `buy` consolidation) so players and mobs share the
same code path. Reframed from the original "covet a player's
gear" half (dropped — players don't drop gear) into reactive
lookout and opportunistic predator patterns.

- New shared function `actions.Consider(actor, target) → ConsiderResult`.
  Player wrapper collapses to ~15 lines (~830 lines deleted);
  `internal/mobcommands/consider.go` is the parallel mob wrapper
  (`MobActor.SendText` no-op so the math runs silently).
- New btree primitives: `target_power_ratio_above` /
  `target_power_ratio_below` (condition) and
  `target_weakest_mob_in_room` (action). Target resolution chain:
  `Event.UserId` → `Aggro.MobInstanceId` → `Aggro.UserId`
  (matches `actions.ResolveAggroTarget`).
- `mob.HatesMob` predicate gates predation — faction/pack
  awareness without coupling to the 1.2 substrate.
- Demo wiring: `lookout` archetype gains `player_enter` ambush
  branch (`target_power_ratio_above: 1.0` — outclass before
  ambushing); new `predator` archetype copies generic_fighter
  + adds a leading `mob_idle` predation branch
  (`ratio_below: 0.85`); 3 ironwind wolves (steppe 205, young
  206, scarred 223) flip to predator. Alpha wolf 215 retained
  `leader` archetype to preserve rally/warcry behavior; future
  `predator_leader` hybrid logged as follow-up.
- PowerScore audit deliverable: new "Power Scoring & Gear
  Contribution" section in `internal/combat/context.md`
  documenting how equipment flows through the existing
  `ValueAdj` / mitigation pipes (no math changes).

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.4-mob-consider.md`.

### 2.5 — Mutations on mobs (body-plan gating + intrinsic mutations)

Generalized chunk 2.2a's incorporeal-only mutation support into
a full body-plan gating model. Species declare what body parts
they have; mutations declare what they require. Species can
additionally declare intrinsic mutations that stack additively
with acquired mutations at character init.

- Schema: `Species.BodyParts []string` + `IntrinsicMutations
  map[string]int` (yaml `body_parts:` / `intrinsic_mutations:`).
  Canonical seven-tag set: `arms, hands, legs, eyes, mouth,
  skin, tail`. `MutationSpec.RequiresArms bool` → `RequiresBodyParts
  []string`. Boot-time validation panics on unknown tags or
  unknown mutation ids in intrinsic_mutations.
- New: `Character.ApplyIntrinsicMutations(species)` helper —
  cap-aware additive merge at init time (`MutationMaxRank = 4`,
  chunk-2.2a convention). Called from mob spawn + player creation.
- Gating sites: random-roll pool (`GetWeightedPool` now takes
  `*species.Species`), curated `SpawnMutations` path on mob YAMLs
  (latent bug fix — was applying unconditionally), 4 mid-game
  acquisition sites in user-tick / btree quest action / quest
  engine bridge.
- Migration: all 35 existing species + 4 new elemental species
  (sand, storm, ice, smoke) — total 39 species YAMLs touched.
  17 mutation YAMLs gained `requires_body_parts:` declarations.
  Five `instance_planar_oasis` elemental mobs repointed: king
  kept on magma + new mob-YAML `spawnmutations: [large]` (top-
  level, NOT under a `mutations:` map — caught in smoke); queen
  moved to new ice species (dropped her chunk-2.2a
  `incorporeal:4` override since her crystal/water form is
  corporeal per description); prince moved to new smoke species.
- Cleanup: redundant `mutations: { incorporeal:4 }` overrides
  removed from 4 `summons` mobs (wraith/spectre/fire/air) —
  incorporeal is now intrinsic on the species.

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.5-mutations-on-mobs.md`.

### 2.6 — Sunset legacy tactics engine

Reframed from the original "fix the Edrin priority race"
band-aid into the structural fix: deleted the legacy
`internal/mobai/` tactics engine entirely and migrated all 44
tactic-using mobs to the behavior tree (btree) system. The
Edrin priority-race bug is now structurally impossible (btree
selectors are inherently priority-ordered, no async reaction
queue racing `InitiateCast`).

- ~1,144 net lines of legacy code deleted. `internal/mobai/`
  directory removed entirely (10 files: tactics.go, reactor.go,
  actions.go, types.go, memory.go, triggers.go + tests). The
  `CombatMemory` substrate (grudge tracking across flee /
  re-engage) was preserved at `internal/mobs/combat_memory.go`
  — it was used outside the tactics engine.
- Zero new btree primitives — existing `mob_has_buff` + invert
  decorator covers the legacy `missing_buff:N` trigger.
- 5 existing archetypes (`generic_fighter`, `predator`,
  `leader`, `lookout`, `tank_taunter`) gained a shared
  `mob_hurt + mob_health_below:25 → flee` panic-flee branch as
  FIRST child. tank_taunter additionally gained a
  `mob_hurt + mob_health_below:20 → callforhelp` branch
  (absorbing the `tank` preset). ambusher gained a
  `mob_combat_round + target_is_casting → trip` branch
  (absorbing the `ambusher` preset's third rule).
  Post-smoke fix: panic-flee REMOVED from tank_taunter — flee
  preempted callforhelp at the threshold boundary; tanks
  semantically shouldn't flee.
- 1 new shared archetype `defensive_caster` absorbs 4 mobs
  from the `defensive_caster` and `caster_backline` presets
  (goblin_shaman 219, tunnel_shaman 74, bandit_caster 285,
  elemental_queen 321).
- 5 new per-boss archetypes preserve unique spell rotations:
  `boss_edrin`, `boss_sylara`, `boss_rhett`, `boss_soren`,
  `boss_chrysalis_phantom`.
- 44 mob YAML migrations strip `tactic_preset:`, `tactics:`,
  `reaction_delay:`, `tactical_discipline:`. The
  corresponding 4 `Mob` struct fields removed.
- Known follow-up: Edrin/Sylara's `conviction-ward` opening
  cast has no buff self-gate (conviction-ward is a shield
  spell with no `buff_id`). Bosses re-cast wastefully after
  the shield expires — behavior is not broken, just wasteful.
  Polish item for a future tuning pass.

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine.md`.

---

## 2026-05-09 — Phase 1 substrate complete (chunks 1.6 + 1.7)

Two more aliveness substrate chunks shipped on `feature/mob-aliveness-1.3-crimes`,
closing out **Phase 1** of the MOB_ALIVENESS_ROADMAP (7 / 40 done).
Branch carries chunks 1.1–1.7; not yet merged to development.

### 1.6 — NPC-to-NPC Relationships

Mob templates now declare a kinship/friendship/rivalry/lover/employer-
employee graph inline in their YAML. Engine builds an in-memory
graph at startup with auto-mirror — symmetric edges (family, friend,
rival, lover) reverse the same type, asymmetric (employer ↔ employee)
flip. Subtype is per-side flavor ("brother," "wife," "drinking-companion").
Permissive validation: bad edges warn-not-panic.

- Public API: `RelationsOf`, `RelationsOfType`, `KinOf`, `AlliesOf`,
  `RivalsOf`, `RelationsBetween`, `AreRelated`, `EmployerOf`,
  `EmployedBy`, `AllRelations`, plus mutation `Add` / `Remove` /
  `ChangeType` (in-memory v1; persistence overlay deferred).
- Admin command `relationship show / between / add / remove / list`
  + helpfile.
- Future consumers: 4.5 reactive goal seeding (revenge), 3.6 NPC↔NPC
  idle conversation.

Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`,
plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.6-npc-relationships.md`.

### 1.7 — World-Model Facts

Standing-fact registry plus per-NPC awareness store. The
`recentGossipEvents` TempData that the gossip pipeline used for
event dedup is now a persistent on-disk awareness file —
`heard_events` (bounded FIFO via `FactsHeardEventsMax`, default 32)
sit alongside `known_facts` in the same `facts.awareness/{mobId}.yaml`.
`buildGossipLine` was migrated; existing event-only output is
preserved, and known facts now mix into the gossip candidate pool
(70% events / 30% facts when both pools are populated).

- Three withdraw signals: manual `Withdraw`, time-based
  `expiry_round` + `PruneExpired` sweep, auto-withdraw via
  `withdraw_on_respawn_of` field on the fact (new RoomChange
  listener fires when the bound mob's instance enters a room).
- Lazy-filter on read for awareness × registry join — withdrawn /
  expired facts are skipped without active cleanup.
- Worldevents got a stable `Id uint64` field (atomic-counter at
  emit time) so awareness can reference events by id.
- Mob YAML extension: `knows_facts: [factId, ...]` for inline
  authoring; seeded into awareness at `mobs.LoadDataFiles`.
- Admin command `fact list / show / declare / withdraw / expire /
  prune-expired / awareness / teach / forget / forget-all` +
  helpfile.
- New `fact-default` gossip template family ("I heard {description}"
  / "Word is, {description}" / "They say {description}").

Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.7-world-facts-design.md`,
plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.7-world-facts.md`.

### Backfilled `context.md` for chunks 1.1–1.6

Per a new aliveness-roadmap maintenance rule: every chunk that
creates a new `internal/<package>/` ships a `context.md` in the
established DOGMud style. Chunks 1.1–1.5 missed this; chunk 1.6's
plan included the backfill. Now present:
`internal/{opinions,factions,crimes,knowledge,bounties,relationships,facts}/context.md`.

### Path-doubling fix for chunks 1.4 / 1.5 / 1.7

Chunk 1.7's review caught that knowledge, bounties, and facts
packages all added `world/dogmud` to a `DataFiles` config that
already includes it — runtime data was landing at
`_datafiles/world/dogmud/world/dogmud/{knowledge,bounties.yaml,facts.yaml}`.
Same bug class that chunk 1.2 fixed for opinions/factions originally.
All three packages now use the unwrapped path. No data migration
required since chunks 1.x are feature-branch only and never reached
prod.

## 2026-05-06 — Crafter output routes to shop

Hotfix for a regression where crafter mobs' gear-grade crafts (iron
daggers, bucklers, leather armor — anything with combat stats)
landed in the mob's backpack instead of shop stock. Players saw
only the sub-component crafts (steel ingots, etc.) in the shop list.
Cause: the Priority-1 self-gear-upgrade selector flagged gear-grade
recipes as "upgrades" for the mob whenever an equipment slot was
empty, which is most slots on a shopkeeper crafter like Kerra. The
craft success path then routed to `mob.Character.StoreItem` rather
than `shopInv.AddStockAtRound`. Fix: all crafted output now lands
in shop stock regardless of craft selection priority. Priority-1
still triggers gear-grade crafts even when narrowly unprofitable
(so shops carry actual weapons/armor, not just intermediates).

## 2026-05-05 — Economy Scoring Refactor

Five-axis economy health scoring replacing the single weighted-fill score
on the admin dashboard. Spec at
`docs/superpowers/specs/2026-05-05-economy-scoring-refactor-design.md`,
plan at `docs/superpowers/plans/2026-05-05-economy-scoring-refactor.md`.

### What changed

- **Five score axes** — Stock, Throughput, Input Rate, Logistics Health,
  Shop Gold — each answering a different question. Old `PerShopScore` is
  retained as `StockScore` (just renamed in the dashboard).
- **Throughput score** measures Time-to-Refill (TtR): the player-facing
  "how long does iron ore stay out of stock" experience, weighted heavily
  toward common materials so crafting-grind viability drives the score.
- **Input rate score** measures items entering the supply per game-day
  (forager deliveries + restock cycles), per zone, against an
  auto-derived target.
- **Logistics health** for caravans and foragers is now a composite of
  cycle rate and cargo flow with hard multipliers for stuck (×0.4) and
  despawned (×0). Halix-despawning and Kessa-stuck failure modes now
  read near-zero on the dashboard instead of moderate.
- **Shop gold score** surfaces merchant liquidity: a merchant with no
  gold can't buy from players, which was previously invisible.
- **Per-rarity-tier restock cadence** replaces the single global ticker.
  Commons (tier 50) refill every 1 game-hour; rares (tier 10) every 5
  game-days as a slow backstop above forager/sale input.

### Data layer

- New persistent counters on `ShopInventory`: `SalesCount`, `BuysCount`,
  `RestockCount`, `StockEvents` (rolling 7-game-day window),
  `CurrentDepletion`. Drives TtR scoring.
- `LbsDelivered` cumulative counter on caravans and foragers. Drives
  logistics cargo-flow scoring.
- All zero-default; existing shop YAMLs and snapshots load cleanly with
  no migration step.

### Dashboard

- 5-card top row: Overall, Stock, Input, Throughput, Shop Gold.
- New Throughput table: per-shop TtR by tier band, per-window medians.
- New Input Rate table: per-zone rate with source breakdown
  (forager / restock) and tier mix.
- Logistics panels gain Multiplier and Lbs Delivered columns.
- Stock table relabeled and gold-score column added; noisy
  per-window stock-score-delta cells removed.

### Configuration

New `Balance` knobs (all overridable in `_datafiles/config.yaml`):
`RestockCadenceTier{50,40,30,20}Hours`, `RestockCadenceTier10Days`,
`TtRTargetTier{50,40,30}Hours`, `TtRTargetTier{20,10}Days`,
`TtRWindowGameDays`, `LogisticsStuckRounds`, `LogisticsStuckMultiplier`,
`ScoreWeight{Stock,Input,Throughput,ShopGold}`. Defaults set in code so
old config files continue to load.

### Smoke-test follow-ups

Same-day fixes that surfaced once the new dashboard had eyes on
zone-by-zone health:

- **Eager-spawn shop-bearing rooms at boot.** Shopkeeper mobs in zones
  without players were never instantiated — only the explicit
  `systemNPCAnchorRooms` list (foragers + caravan) got pre-spawned.
  Result: Stillwater, Sanctum Basin, Watchers Crossing showed flat
  zero on the input rate dashboard because no `MobIdle` ticks ever
  fired for their shopkeepers. `main.go` now scans every loaded room
  at boot and `Prepare()`s any with `HasShop()` spawninfo.
  `IsEssential()` extended so shop-bearing rooms aren't unloaded by
  `RoomMaintenance`.

- **Crafter consumption tracking.** Crafter mobs now use round-aware
  `RemoveStockAtRound` / `AddStockAtRound` so their consumption
  marks `CurrentDepletion` and refills push `StockEvent`s — the
  TtR throughput score sees crafter-caused depletions instead of
  only player buys. New `ConsumedByCrafterCount` counter on
  `ShopInventory` captures per-shop crafter outflow. `SaveShop` is
  called after successful crafts (and salvage) so freshly-crafted
  output items survive server restart.

- **Per-ingredient stock-reserve floor.** Crafter mobs no longer
  drain their own ingredient stock to a single unit. New
  `CrafterIngredientReservePct` config knob (default 0.25) keeps
  at least 25% of `MaxStock` of each ingredient available for
  player purchase. Per-ingredient check, hard floor of 1 for tiny
  `MaxStock` cases.

- **Unrated-item tier-50 fallback.** 145 of 213 weapons / armor /
  consumables predate the rarity_tier system and have no tier
  field — they were invisible to the new per-tier `RestockTier`
  ticker. `RestockTier` now defaults `RarityTier == 0` to 50
  (commons, hourly cadence). Tutorial NPCs (Adela's tutorial gear)
  restock again. Per-tier audit + tagging tracked as a follow-up.

- **Forager fixes.**
  - Halix marked `non_combatant: true` so hostile steppe mobs can't
    aggro. He was dying mid-cast-fold-recall to goblin/wolf hits;
    death calls `Suicide` which permanently destroys the instance.
  - Kessa & Tova got a direct-teleport recall replacing
    `cast fold-recall`. The fold-cast machinery only advances
    inside the combat loop, so foragers recalling outside combat
    set `CastingState` and froze forever. Direct teleport bypasses
    the cast system entirely.
  - `ForagerRestDurationRounds` exposed as a config knob (default
    40 rounds, down from a hardcoded 120) for cycle-pacing tuning.

- **Prewarm zone-key fix.** `prewarmThroughputFromPersistedFiles`
  for both forager and caravan keyed the cache by snake_case
  directory name while runtime callers use display-name zone via
  `mob.Zone`. Result: phantom duplicate cache entries and
  `SaveAllThroughputs: no cached entry` errors on every shutdown.
  Prewarm now keys by the YAML's `zone:` field (display form) so
  prewarmed and runtime entries share a key.

- **Validator fix for shared component tags.** `ValidateRecipeIngredientTags`
  picked one arbitrary item per `ComponentTag` for validation.
  Map iteration is randomized in Go, so when multiple items shared
  a tag (four bottles all tagged `bottle`), boot intermittently
  panicked if the chosen item lacked the recipe's discipline.
  Validator now checks "at least one item with this tag has the
  discipline" — eliminates the boot-roulette panic.

- **Stillwater bank.** Town tier-1 settlement now has a counting
  house bank with NPC clerk (mob 356, room 5100). Tutorial-style
  bank flavor matching the existing Thornwall bank pattern.

- **Multi-quantity sell** (`sell 5 iron-ore`, `sell all iron-ore`,
  `sell all.iron-ore`) mirroring the existing multi-buy parser.
  Bare `sell all` with no item rejected as too easy to fat-finger.

- **Bank storage stacking.** Stackable items (components, food,
  drink, potions, grenades, ammo) now consume one slot per stack
  in the player's bank instead of one slot per unit. Lazy on-load
  migration of legacy `Items []Item` into the new `Slots
  []StorageSlot` shape. Storage fees billed per slot. No flag day.

## 2026-05-04 — Vendor Polish Hotfix

Two post-merge fixes caught during smoke testing the same day.

### Bug fixes

- **Shopkeepers re-seeding at 500g instead of YAML values.** Three
  layers of bug:
  - `RegisterMobShop` hardcoded `startingGold = 500`, ignoring
    `mob.Character.Gold` from the YAML — Phase 7's bumps to 1000g
    specialist / 5000g general never flowed through.
  - `PrewarmShopForSpawnPlacement` (which runs at boot for every shop
    placement before real mobs spawn) built a synthetic Mob without
    forwarding `Gold`, so even after the first fix the prewarm path
    re-floored everything to 500g.
  - Persisted shop YAMLs from the buggy boot survived a server
    restart; only a fresh wipe of `_datafiles/world/dogmud/shops/`
    forces re-seed from the (now-correct) seeding code.

  Specialists now correctly seed at 1000g, generals at 5000g.

- **Vendors rejecting tagged items they should buy** (Maren refusing
  cattail cloak, Kerra refusing arena tower shield). `EvaluateBuyRules`
  was pricing items not in the vendor's stock list at the 5× scarcity
  ceiling (`current=0, restock=1` → ratio 0 → `PriceCeiling = 5.0`),
  pushing the buy offer above the gold-reserve floor and self-rejecting.
  Now: stocked items still get full dynamic scarcity pricing; unstocked
  items use flat `value × BuyRatio`. Opportunistic vendor buys work
  regardless of whether the shop normally stocks the item.

### Deploy step

Wipe the shop save directory on prod **before restarting** so the
gold reseeds from the new code path:

```bash
./tools/economy/wipe_shop_state.sh
```

Players' personal gold and inventories are untouched.

## 2026-05-04 — Vendor Types & Economy Polish

Big economy overhaul. Buy rule rewrite, per-vendor audit, tier-50/40
baseline restock layered onto caravan/forager flow, forager stuck-state
watchdog, dashboard rework with stock-score-delta + per-rarity-tier
throughput bars.

### Vendor system overhaul

- **Item-side `vendor_categories` tag.** Every salable item now carries
  one or more discipline tags (`alchemy`, `blacksmithing`, `cooking`,
  `enchanting`, `jewelcrafting`, `tailoring`). Cross-cutting mats like
  iron ingot are multi-tagged (`[blacksmithing, jewelcrafting,
  tailoring]`). Lore / flavor / removed-from-game items get
  `not_salable: true` instead of relying on value gymnastics.

- **New buy rule.** Single-rule tag-overlap replaces the old 5-rule
  chain (quest / craft-material recipe-walk / gear-upgrade / potion /
  general-fallback). Reject conditions in order: quest item, declining
  potion, untagged item, vendor's `craft_support` doesn't match any of
  the item's tags, vendor at MaxStock, or buying drops below gold
  reserve. Removes ~80 lines of rule-helper code.

- **No more gear-upgrade purchases.** Specialist shopkeepers no longer
  buy random equipment — they're non-combatants who never wore what
  they bought anyway. The behavior was vestigial.

- **Apothecary Ilsa now buys all alchemy mats.** Previously the
  recipe-walk gating rejected mats not in her specific recipe list.
  The new tag-overlap rule accepts every alchemy-tagged item.

### Per-vendor audit

- **6 NPCs reframed as questgivers / flavor mobs:**
  Korvath (52), Yenna (53), Sigrid (333), Haral (278), Whisper (273),
  Bram (348). Stripped `crafter` / `crafterskill` /
  `crafterrecipeids` / `crafterrestockmaterials` / `craft_support` /
  `shop` fields. Korvath + Yenna keep `non_combatant: true` for
  questline integrity. Bram drops the `noncombat_shopkeeper`
  archetype entirely.

- **Specialist shopkeeper gold bumped to 1000g** (was 200/300/500
  variable). 12 NPCs: Kerra, Voss, Thornwall food vendor, Tess, Vael,
  Maren, Brynn, Tov Brann, Brindle, Ilsa, Edda, Kess.

- **General store gold bumped to 5000g** (was 400/500). 4 NPCs:
  Adela, Brecca, Siv (fence), Wulf.

### Restock pacing

- **Tier-50 and tier-40 mats now refill at every shop** on the
  existing crafter tick. Caravan-served zones (Stillwater, Thornwall)
  used to fully suppress that path, leaving common mats reliant on
  caravan/forager throughput. Now baseline tier-50/40 supply layers
  on top, while rarer tiers (30/20/10) still flow through caravan +
  forager exclusively.

### Forager reliability

- **Stuck-state watchdog.** Foragers wedged in any state for more
  than `ForagerStuckThresholdRounds` (default 600) get force-reset
  to Recalling — they head home, dump satchel into the lockbox, and
  re-cycle. Logs a `Warn` on every reset. Should end the periodic
  Halix "(not active)" mystery.

- **Dashboard distinguishes despawned vs idle.** A forager whose
  mob isn't currently spawned shows `(despawned)`; a live mob with
  empty BTreeState shows `(idle, no state)` plus a structured Warn
  log. Adds `StuckRounds` field for at-a-glance stuck detection.

### Dashboard rework

- **Stock-score-delta replaces gold-delta** in the per-window
  columns (1h/6h/1d/3d/1w). Each shop's `StockScore` is
  `sum(Current) / sum(MaxStock)`; the column shows the change in
  percentage points between snapshots. Gold value still visible as a
  static column.

- **Tier-color bars replace bucket-color bars.** Five tier classes:
  50 (grey), 40 (green), 30 (blue), 20 (purple), 10 (gold). Applied
  to shop stock bars and to new caravan/forager throughput rows.

- **Per-rarity-tier throughput bars** for caravan + forager. Each
  delivery to a destination shop bumps a per-tier counter on the
  corresponding mob; dashboard renders the window-delta as a
  proportion-stacked tier bar. Caravan pickups don't count — only
  destination drop-offs.

- **Names work for un-spawned shopkeepers.** When the NPC isn't
  currently in the world, the dashboard now falls back to the mob
  template's name instead of showing `#<mobId>`.

### Persistence

- **Two new gitignored runtime directories:**
  `_datafiles/world/dogmud/foragers/<zone>/<mobId>.yaml` and
  `_datafiles/world/dogmud/caravans/<zone>/<mobId>.yaml`. Track
  cumulative `DeliveriesByTier` counters across reboots. Boot
  prewarm + graceful-shutdown save mirror the shops/ pattern.

- **`NotSalable bool` field on ItemSpec.** Replaces the brittle
  "Value <= 0" skip in the vendor validator with an explicit opt-out.
  38 lore / flavor / legacy items got `not_salable: true`.

### Manual deploy step

⚠️ **Before starting the new server on prod**, wipe the persisted
shop save state so shops re-seed fresh from the new mob templates:

```bash
./tools/economy/wipe_shop_state.sh
```

Or directly: `rm -rf _datafiles/world/dogmud/shops/`. Players' personal
gold and inventories are untouched — only NPC shop state (gold drift,
current stock levels, last-restock round) is reset.

### Migrations

- **Recipe discipline shuffle.** `master-lockpicks` moved from
  jewelcrafting → blacksmithing; `reinforced-disarm-kit` moved from
  blacksmithing → jewelcrafting. Players who learned either recipe
  under the OLD discipline get their NEW-discipline skill bumped to
  the recipe's minimum (20 for master-lockpicks, 15 for
  reinforced-disarm-kit) so they don't lose craft access. One-shot,
  gated by misc-data key.

### Validators

- **`items.ValidateVendorCategories`** — boot-time check that every
  salable item carries a valid `vendor_categories` value. Cold boot
  panics on offending items; reload logs structured Error.

- **`crafting.ValidateRecipeIngredientTags`** — ensures every recipe
  ingredient resolves to an item carrying the recipe's discipline.
  Catches typos like `item_tag: lake-mintt` at boot.

## 2026-05-02 — Forager + Caravan Followup

Five fixes in the now-shipped forager + caravan stack, plus a caravan-cadence
tweak.

### Bug fixes
- **Whisper off the caravan rotation.** Room 507 (The Listening Post) is
  Whisper's quest-only spot in the locked, trapped, phantom-guarded section
  of Thornwall — never a standard merchant. Previously the caravan tried to
  restock her on every Thornwall pass. She's now removed from
  `thornwallVendorRooms`.
- **System NPCs spawn at boot.** Caravan master (room 4042) and the three
  foragers (Tova/4123, Halix/3040, Kessa/4197) now have `room.Prepare(false)`
  fired against their anchor rooms during the boot data-file load. Previously
  these mobs only spawned the first time a player walked into the room — and
  forager anchors are wilderness, so they could go offline indefinitely. The
  `/admin/economy/` dashboard's "(not active)" forager rows are gone after a
  clean boot.
- **Foragers no longer deadlock at sanctuary.** Stage 3.4's carry-ratio
  rest-extension would park a forager forever if vendors were saturated and
  her satchel never drained. Foragers now dump satchel surplus into a new
  per-sanctuary lockbox container on Recall arrival, so the satchel always
  empties between cycles. The carry-ratio gate is retained as a backstop for
  the lockbox-full case.
- **Kessa actually delivers to the caravan.** The previous mechanism required
  Kessa and the caravan to coincide at North Road 4038 in the same dwell
  window, which never happened reliably. New mechanism: Kessa drops her
  fernway-bucket items into a persistent **shipping crate** at 4038 and
  heads home; the caravan drains the crate into its wagon on its next
  pass. No timing dependency. The flag-based `caravan_load` mechanism is
  deleted entirely — real items now move through the wagon (Stage 3.4
  vendor-restock path).

### Content
- **Sanctuary lockboxes** at the three forager anchor rooms (4123 Stillwater
  Temple, 3040 Ironwind Steppe, 4197 Forager's Camp). Difficulty-10 lock,
  500-item capacity, fresh combination each forager cycle (the lock's
  `RotationSeed` bumps on every dump, invalidating any cached keyring
  entries). Players who pick the lockbox get the forager's surplus
  materials but must redo the picklock minigame each cycle.
- **Roadside shipping crate** at North Road 4038 (`crates/4038-fernway_shipment.yaml`).
  Visible to players as a noun, but every interaction (`get`, `look in`,
  `put`, `picklock`, `lock`) returns flavor text — only the caravan and
  Kessa modify it via state-machine code.

### Tuning
- **`CaravanDepotDwellRounds: 720 → 360`.** Halved. Foragers now run from
  boot and never deadlock, so they dominate day-to-day throughput regardless
  of caravan cadence. Halving the depot dwell roughly doubles caravan
  visibility in each town — event-style deliveries beat once-per-day realism
  here.
- **New config knob `ForagerLockboxCapacity` (default 500).** Caps the
  per-forager sanctuary lockbox; if a player goes a long time without
  picking a forager's lockbox open, the box can saturate and the forager
  reverts to rest-extension behavior until space opens up.

### Engine
- **`gamelock.Lock.RotationSeed`** added. `SetLocked()` bumps it. When >0,
  it's mixed into the `util.GetLockSequence` hash so a re-locked container
  produces a new combination — invalidating any cached keyring entry.
  Default 0 keeps every existing lock's combination unchanged.
- **New package `internal/sealedcrate/`.** Player-untouchable, room-bound,
  capacity-bounded delivery container. Persists at
  `_datafiles/world/dogmud/crates/<roomid>-<label>.yaml`. Mutated only by
  forager + caravan tick functions; all player commands short-circuit.
- **`Room.SealedCrate`** field + `Room.AttachSealedCrate` + boot loader at
  `main.go` end of `loadAllDataFiles`.

## 2026-05-01 — Mob Aliveness Roadmap (planning)

**Note:** Planning doc only — no engine, content, or config changes.

- **`MOB_ALIVENESS_ROADMAP.md`** added at project root. Long-term plan
  for making NPCs feel alive: persistent disposition memory, factions,
  citizenship/justice, NPC schedules, motivations/goals, equipment-
  awareness, bounty hunting, and mob/player command parity.
- 39 chunks across 6 phases — Substrate, Tactical fill-in, Routine
  layer, Strategic layer, Cross-cutting features, Audit & polish — with
  a progress tracker at the top. Living doc; status updates as chunks
  ship. Each chunk gets its own spec/plan when picked up.
- Five MEMORY-tracked items absorbed into the roadmap: peacefulquest →
  faction system, Companion Phase 5 (mutations), tactics-cast
  preemption gap, PvM/MvP/PvP/MvM parity gaps, Stillwater town-flavor
  pass.

## 2026-05-01 — Economy Health Dashboard

**Note:** New `/admin/economy/` web dashboard for monitoring NPC
supply chain health. Backend-only release — no in-game commands or
content changes. Player-facing change is one config bump (idle
timeout).

### Dashboard
- **`/admin/economy/`** — five score cards (Economy / Shops / Caravans
  / Foragers / last snapshot), per-discipline rollup of shops grouped
  by `craft_support:` tag (blacksmithing, alchemy, tailoring, cooking,
  jewelcrafting, enchanting, general), per-shop detail with stock bars
  colored by supply bucket, caravan + forager tables with cargo bars
  in pounds. Auto-refresh 30s/60s/2m. Manual "Snapshot Now" button for
  ad-hoc before/after comparisons.
- **All tables sort alphabetically** (discipline name, shop name,
  caravan name, forager name) for predictable row order.
- **Scores:** 0-100 colored red <40 / yellow 40-70 / green >70.
  Per-shop score weights item fills by `RestockQty`; per-discipline
  score is the mean of shops in that bucket; caravan/forager scores
  count Thornwall→Thornwall and Resting→Resting cycles across the
  last 168 hourly snapshots with a stuck-penalty if a state has been
  held >5000 rounds. Overall economy score is weighted 0.6/0.2/0.2
  (shops/caravans/foragers) with renormalization for components with
  insufficient history.
- **Hourly snapshot ticker** writes to
  `_datafiles/economy/snapshots/{unix_ts}.yaml` (gitignored runtime
  state). Auto-snapshots pruned past 30 days; manual snapshots never
  pruned.
- **Delta columns** at 1h / 6h / 1d / 3d / 1w show gold deltas per
  shop and per discipline against the closest historical snapshot
  within ±50% tolerance.
- **Boot prewarm** populates the shop cache eagerly: every persisted
  shop YAML loads on startup, AND every shop-bearing mob's spawn
  placement (from room `spawninfo` blocks) is pre-registered without
  spawning the actual mob. Result: dashboard shows the full set of
  shops + foragers at boot, not just ones in zones a player has
  visited. Inactive forager profiles render as `(not active)` rows
  so the dashboard always shows all 3 (Tova / Halix / Kessa).

### Schema additions
- **`craft_support:` field** on every shop-bearing mob YAML (22
  files). One-of-7 valid values: `blacksmithing`, `alchemy`,
  `tailoring`, `cooking`, `jewelcrafting`, `enchanting`, `general`.
  Source of truth for the dashboard's discipline rollup.
- **Startup validator** (`shops.ValidateShopMobTags`) panics if any
  shop-bearing mob is missing or has an invalid `craft_support:` tag.
  Server refuses to boot until every shop is categorized. On
  `/reload` the validator logs a structured Error with remediation
  hint instead of panicking, so the running server stays up while
  you fix the listed mob YAMLs.
- **Persisted shop YAMLs auto-migrate** from the mob template's
  `craft_support:` value on next boot — no manual edits to the 7
  existing runtime files in `_datafiles/world/dogmud/shops/`.
- **Cargo metrics in pounds** — caravan + forager `cargo_weight` /
  `cargo_capacity` use real carry weight (5000 lbs for the wagon).
  Forager cargo capture also walks ComponentItems + PotionItems for
  foragers that equip a component bag or bandolier (Halix's spear
  case). Wagons unchanged — they don't equip.

### Config knobs (Balance section)
- `EconomySnapshotIntervalHours` (default 1)
- `EconomySnapshotRetentionDays` (default 30)
- `EconomyScoreWeightShop / Caravan / Forager` (defaults 0.6 / 0.2 / 0.2)

### Player QoL
- **`MaxIdleSeconds` 1800 → 18000** (30 min → 5 hours). Players being
  kicked after 30 min of idle was friction for roleplay sessions.
  10x bump. AfkSeconds and ZombieSeconds unchanged — only the hard
  kick.

### Runbook
- See `docs/economy/dashboard-runbook.md` for what each card means,
  how snapshots work, troubleshooting, and the process for adding a
  new vendor discipline.

## 2026-04-30 (evening) — Stage 3.4 Hardening + Pricing Pass (dev only)

**Note:** Smoke-test fixes and late-day polish on the Stage 3.4 economy stack. Promotes to `master` with the rest of the economy stack.

### Forager fixes
- **Halix anchor moved** from Thornwall Temple Interior (468) to Sheltered Ridge Alcove (3040) where Hermit Kael also camps. The original anchor put a Steppe forager in city center with forage range one zone away — round-trip walks blew through state-machine timeouts. The Steppe-side anchor matches Tova's and Kessa's pattern (anchor near forage range; walk into town only to deliver).
- **Forager Vella renamed to Tova** (mob 371, Stillwater Marsh). Disambiguates `look vella` from the long-running Mistress Vella Thorne (mob 355, Stillwater town).
- **Forager state machine no longer re-issues `fold-recall`** while already casting — was resetting cast progress every idle tick.

### Caravan hardening
- **Auto-reset watchdog at the top of every caravan tick.** If the caravan has been stuck in a single state longer than 5× the configured dwell (floor 300 rounds), state resets to `ThornwallDwell` with a `mudlog.Warn` entry. Recovers from orphaned-state corruption after restarts or unusual party deaths without admin intervention.
- **New admin command `caravan reset [<instanceId>]`** — manual reset for one caravan leader (numeric arg) or every caravan leader (no arg).
- **Party-aware hostile check** stops the caravan from abandoning members. Previously the leader's `hostilesInRoom` only checked the leader's room — a follower fighting alone in a different room got left behind. The new `partyHostilesNearby` walks every party member's room.
- **Shop persistence after caravan + forager visits.** Stock changes now persist to disk inside `VisitVendorsInRoom` (was in-memory only — a panic lost an in-flight cycle's deliveries).
- **Wagon equipment slots suppressed.** New `hide_equipment_slots: true` flag on the Mob struct hides the empty Equipment block in `look mob` for entities like the wagon that don't wear gear.
- **Boot panic fix:** wagon name shortened to match its YAML filename per the engine's `ConvertForFilename(name)` convention.

### Pricing + accessibility
- **Pricing pass on 26 mat YAMLs (Approach B)** — rarity-tier-aligned base values. The dynamic shop multiplier already does most rarity work via scarcity (0.25×–5.0× swing); base values now sit at band midpoints rather than encoding rarity twice. Bands: tier-50 = 1–3g (commodities), tier-40 = 5–25g (standard), tier-30 = 25–75g (regional), tier-20 = 80–500g (uncommon). Biggest corrections: Hive Fragment 500→25g (was tier-20-priced but tier-40-tagged), chrysalis_shard 6→80g, gold_wire 8→80g, mutation_catalyst 10→100g, ironbark_shaving 4→25g, raw_gem 5→25g. Stillwater pearl + Chrysalis Core unchanged at 400/500.
- **Starting player gold bumped 25 → 250.** With the new mat prices, a fresh character couldn't afford even one mid-tier craft attempt; 250g lets them try.

### Other small fixes
- **Companion spawn stamina:** previous spawn path set Health and Conviction to max but never set Stamina, so companions spawned at 0 SP and were immediately stamina-broken.
- **Steal gate ordering:** `skullduggery.steal` skill-rank check moved AFTER target validation. Stealing from a `player_attack_immune` mob now surfaces the immune rebuff first instead of the misleading "not advanced enough" rebuff.
- **Mob.Cast diagnostics:** surfaces `InitiateCast`'s silent early-exit reasons (AlreadyCasting / OnCooldown / InvalidSpell / NoTarget) at debug level. Caught by smoke test as a tactics-cast preemption gap (logged as followup).
- **Follow auto-timer removed.** The 10-min auto-expiry in `modules/follow` dropped follow with no in-fiction reason. Teleport drops, death drops, and explicit `follow stop` still apply.

### Developer docs
- `internal/items/context.md` gains three new sections (Rarity Tiers, Pricing Bands, Supply Pipeline) so the items package's developer doc reflects the post-3.4 economy.

## 2026-04-30 — Stage 3.4: Real Item Transfer (dev only)

**Note:** Final stage of the caravan/economy effort. Once this lands
on `development`, the entire economy stack (Stages 3.0b through 3.4)
promotes to `master` as a coherent update.

- The caravan now physically hauls items: a new wagon mob (374) with
  ~5000 carry capacity rides with the caravan party. `look wagon`
  shows the actual cargo. Two draft horses (Hob 375, Bran 376) pull
  it. All three are player_attack_immune.
- Wagon dies if the caravan is wiped at the bandit camp; cargo
  distributes to bandit inventories (round-robin, capped per bandit's
  carry capacity), with leftovers as wreckage corpse loot. Players
  who kill the bandits afterward get the cargo. Wagon corpse renders
  as "splintered wagon wreckage" with custom description.
- Vendor stock caps now derive from item `rarity_tier` × shopkeeper
  `stock_multiplier` (default 1.0). 51 mat YAMLs got rarity_tier set:
  15 tier-50 (common), 17 tier-40 (standard), 14 tier-30 (regional),
  5 tier-20 (uncommon — pearl, gold wire, chrysalis core/shard/
  catalyst). Tier 10 reserved for future ultra-rare content.
  Future big-city shops can set stock_multiplier > 1.0 for
  proportionally larger stock.
- Foragers now physically deliver items from their satchels to vendor
  inventories (no more abstract RestockBuckets). Items that don't fit
  stay in the satchel for next vendor / next cycle.
- New forager rest extension: when carry > 50% on return home,
  forager stays at sanctuary instead of cycling back out. Prevents
  futile loops in saturated economies — foragers wait at sanctuary
  until players consume from vendors and re-open delivery space.
- Caravan vendor stops are now BIDIRECTIONAL — caravan delivers
  items it brought AND picks up items the local vendors produce in
  abundance, hauling them across town. Pickup is gated by `Current
  >= MaxStock/2` so the caravan doesn't extract from a struggling
  vendor. Pays off the "wholesalers seeking arbitrage between
  regions" worldbuilding from the Stage 2 caravan.
- Chrysalis Core (40010) re-sourced: removed from Aberrant Chrysalis
  in Sanctum Basin tutorial. Now drops 10% from stone beetle queen
  (228) and 5% from windscour wyrm (229) in Ironwind Steppe.
- 6 new mob override fields: carry_capacity, health_max, stamina_max,
  corpse_name, corpse_description, stock_multiplier.
- New btree action `distribute_cargo_to_hostiles` for the wagon's
  death handler.
- New config knob `ForagerRestCarryThreshold` (default 0.5) for the
  rest extension.
- ItemSpec gains `rarity_tier` field. Mob struct gains 6 spawn-time
  override fields. Corpse rendering honors mob's CorpseName +
  CorpseDescription overrides.

## 2026-04-30 — Stage 3.1: Forager NPCs (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- Three new forager NPCs feed the supply pipeline that 3.0b wired up:
  - **Vella, the Marsh Forager** (mob 371) anchored to Stillwater
    Temple Interior (4123). Wanders Stillwater Marsh (rooms
    4177-4196), engages prey wildlife (marsh rats, dragonfly
    swarms), salvages corpses, delivers Stillwater + base + overlap
    mats to the 8 Stillwater vendors directly.
  - **Halix, the Steppe Forager** (mob 243) anchored to Thornwall
    Temple Interior (468). Walks the safe northern half of Ironwind
    Steppe, delivers base + overlap mats to the 9 Thornwall vendors.
    Statpool 225 (Ironwind is more dangerous than the marsh).
  - **Kessa, the Fernway Forager** (mob 366) anchored to a new
    Forager's Camp (room 4197, attached west of 4170 Tangled
    Bracken). Walks up to North Road 4038 to meet the caravan; the
    caravan distributes Fernway mats to both towns symmetrically.
- All three are `player_attack_immune: true` (rebuff like
  shopkeepers). They engage prey wildlife on a per-forager
  whitelist, drink a healing salve at HP < 75%, and cast fold-recall
  at HP < 50%. Each carries a thematic 1H weapon (gaff hook,
  hunting spear, hand axe) and a leather bandolier with healing
  salves.
- New behavior tree primitive `forager_step` drives the per-forager
  state machine (resting → traveling → foraging → delivering →
  recalling → loop). Three new conditions support it:
  `mob_can_safely_engage`, `mob_inventory_at_threshold`,
  `mob_hp_below_recall_threshold`.
- New `internal/economy` package mirrors the 3.0b mat-audit-matrix
  as a Go map. New `RestockBuckets([]string)` shop method gates
  vendor refills by supply bucket. Foragers and the caravan both
  use it; only slots whose item-id matches a carried bucket get
  topped up.
- **Caravan changes:**
  - Cycle slowed from ~900 to ~1620 rounds (~2 game days) by
    bumping `CaravanDepotDwellRounds` from 360 to 720. Foragers
    are now the day-to-day reliable supply; the caravan feels
    like a delivery-day event.
  - Two new substates inside each transit leg
    (`outbound_fernway_pickup`, `inbound_fernway_pickup`): the
    caravan dwells briefly at North Road 4038, detects the Fernway
    forager, and acquires the `fernway` bucket flag.
  - `caravan_load` MobState tracks which buckets the caravan
    currently carries. `VisitVendorsInRoom` consumes it, so a
    Stillwater-only caravan run won't restock Fernway slots.
- New room mutator `sanctuary` standardizes the "high-regen room"
  mechanic. Replaces the hardcoded `roomRegenMultiplier` switch in
  the auto-heal hook. `MutatorSpec` gains a `regenmultiplier float64`
  field; multipliers stack multiplicatively.
- Sanctuary mutator wired on:
  - Thornwall Temple Interior (468) — preserves prior 5x regen
  - Sanctum Basin tutorial zone (rooms 101-120) — preserves prior
    5x regen
  - Stillwater Temple of Stillwater (4123) — gains 5x regen for
    the first time, supports Vella's recall destination
  - Forager's Camp (4197) — gains 5x regen, becomes a known safe
    rest stop in Fernway South
- Three new low-tier 1H weapons: marsh gaff hook (10033), steppe
  hunting spear (10034), Fernway handaxe (10035).
- Six new balance config knobs gate forager behaviour:
  `FernwayPickupDwellRounds` (6), `ForagerForageDwellRounds` (8),
  `ForagerCarryThresholdPct` (0.75), `ForagerHPRecallThresholdPct`
  (0.50), `ForagerHealPotionThresholdPct` (0.75),
  `ForagerWaitTimeoutRounds` (150).
- The temple-regen hint generalizes to reference the sanctuary
  class — temples + camps + tutorial all read as one mechanic.
- `ForageCore` (Task 6 originally) consolidated to `internal/forager`
  package so both the player Forage command and the NPC forager
  routine share one yield-table source of truth.

## 2026-04-28 — Stage 3.0a: Stillwater Marsh Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room wetland zone west of Stillwater, themed as marsh
  giving way to upland steppe at the southern terminus. Connects
  from Mill Creek Footbridge (4133) via a new west exit; terminates
  at Far Bog Heart (4195, biome: plains) with a one-way view of
  the Dustwalk beyond.
- Five new wildlife mobs (366-370): river otter, marsh rat,
  dragonfly swarm, snapping turtle, bog adder. **Only the bog
  adder is hostile to players** AND it `hates: [rodent]` — it
  hunts the marsh-rats in adjacent rooms (mirror of 3.0c's
  wolf-hates-boar dynamic, but combined with the only-hostile-to-
  player role into one mob).
- The river otter is the **first non-badger consumer of the
  mustelid species** (24) added in Stage 3.0c — validates the
  species investment.
- All 6 existing Stillwater forage mats (lake-iron, marsh willow
  bark, lake mint, freshwater clam, skitter-shrimp shell,
  Stillwater black pearl) get fresh territory to spawn in. No
  new mats added.
- Stage 3.0a is the territory groundwork for Stage 3.1 forager
  NPCs — the marsh is now big enough for a Stillwater-anchored
  forager to wander, gather, and recall to depot when injured.
- Coord map gains 48 Stillwater catch-up rows (the doc was
  missing all of Stillwater) plus 20 new Stillwater Marsh rows.

## 2026-04-28 — Stage 3.0c: Fernway South Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room zone south of the existing Fernway, themed as deep
  forest tapering to the steppe edge. Connects from Fox Den (4156)
  via a new south exit; terminates at Steppe Edge (4175, biome:
  plains) with a one-way view of the Dustwalk beyond.
- New mustelid species (24) — fills a real gap in the species set
  (we had rodent and canine but nothing for badgers, weasels,
  otters). First consumer is the forest badger; future zones with
  otters or weasels reuse immediately.
- Six new wildlife mobs (360-365): wild hare, roe deer, honey bees,
  feral boar, timber wolf, forest badger. Only the badger is
  hostile to players — the rest are atmosphere or forage support.
  Wolf is `hostile: false` but `hates: [boar]` — emergent
  intra-zone hunt dynamic where the wolf may engage boars without
  threatening the player.
- The 6 existing Fernway forage mats (oak bark, shadowcap mushroom,
  wild hare meat, beeswax, blood-moss, pine pitch from 3.0b) gain
  fresh territory to spawn in. No new mats added.
- Stage 3.0c is the territory groundwork for Stage 3.1 forager
  NPCs — the forest is now big enough for a Fernway-based forager
  to wander, gather, and recall to depot when injured.

## 2026-04-28 — Stage 3.0d: NPC Fold-Recall (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `fold-anchor` and `fold-recall` resolvers now accept `actions.Actor`
  rather than `*users.UserRecord`. Mobs can cast both spells via the
  existing tactics dispatcher and the new Go-hook switch in
  `resolveMobSpell`. Player behavior is unchanged.
- New mob YAML field `fold_anchor_room: <roomId>` pre-stamps a mob's
  fold-recall anchor at spawn. The runtime is then identical to a
  player who already cast `fold-anchor`.
- Old Edrin (mob 275) gets `fold-recall` as a panic spell at
  `health_below:30` priority above his existing flee — he recalls to
  the cluttered back room (4037) when injured. Useful smoke-test rig
  for the new pipeline.
- Caravan crew Ketil/Marta/Lars (mobs 357-359) get the same treatment
  with anchor at the Thornwall Market Square depot (465). Wipe
  insurance for the bandit camp ambush — if their HP drops they
  recall instead of dying, keeping the restock service running.
- Stage 3.0d does NOT add forager NPCs or logistic recall triggers
  (e.g., `inventory_full → cast fold-recall`). Those are Stage 3.1's
  job. Caravan recall is individual, not group-aware: each crew
  member recalls on their own panic threshold.

## 2026-04-28 — Stage 3.0e: Corpse Salvage (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `salvage <corpse>` now works on room-resident corpses, not just
  inventory items. Animal-group mobs yield leather strip + sinew;
  humanoid-group mobs yield cloth strip + leather strip. Each material
  rolls independently against the salvage skill curve. Salvage kit
  required (sold by Fence Dealer Siv, 1g).
- The corpse is consumed on completion (mirrors tagged-item salvage
  behavior — the activity has cost regardless of roll outcome). If the
  activity is interrupted (combat, movement) the corpse stays untouched.
- Added **sinew** (40068), a tough animal-tendon mat sourced from
  corpse salvage on animals. Wired into 2 existing recipes: tailoring's
  Artisan's Satchel (heavy-duty seam binding) and blacksmithing's
  Lake-Iron Hook-Spear (haft lashing).
- 40002 leather strip and 40007 cloth strip reclassified in the audit
  matrix from "Defer to 3.0e" → "Mid-tier overlap (corpse-salvage
  sourced)". Source pipeline now decided. Vendor inventories continue
  to NOT stock these mats — corpse salvage is the v1 source.

## 2026-04-28 — Stage 3.0b: Material Region Split (dev only)

**Note:** This is a dev-only landing. The full economy stack (Stages
3.0b through 3.4) sits unmerged on the `development` branch and ships
to prod (`master`) as a coherent update once Stage 3.4 lands.

- Added 6 new Fernway forest materials: oak bark (40062), shadowcap
  mushroom (40063), wild hare meat (40064), beeswax (40065), blood-moss
  (40066), pine pitch (40067). Each consumed in 1-2 mid/high-tier
  recipes spanning at least 2 craft schools, giving forager-gathered
  Fernway mats real demand once Stage 3.1 ships. (Beeswax tailoring
  recipe wiring deferred to Stage 3.0e corpse salvage.)
- New audit matrix at `docs/economy/mat-audit-matrix.md` classifies
  all 67 raw materials into regional supply buckets (Stillwater,
  Thornwall, Fernway, base, mid-tier overlap, deferred-to-3.0e,
  quest/specialty). This is the durable artifact that subsequent
  stages (foragers, corpse salvage, real item transfer) consume.
- Reshaped vendor inventories across the 17 caravan-served vendors
  into mirrored same-craft pairs. Same-craft Stillwater + Thornwall
  vendors now stock the same mat slot lists, with regional pricing
  asymmetry reflecting the caravan markup (e.g., lake mint 10g at
  Stillwater Apothecary Ilsa, 15g at Thornwall Apothecary Voss).
  Cloth/leather/cord/sinew slots dropped pending Stage 3.0e (corpse
  salvage); the audit matrix flags them for 3.0e to wire properly.
- ~12 mid/high-tier recipes updated to wire demand for the new
  Fernway mats. No new recipes invented; existing recipe corpus
  expanded with one new ingredient slot each.

## 2026-04-27 — Stage 2: Thornwall ↔ Stillwater Caravan System

- Added the **Thornwall ↔ Stillwater caravan**: a three-NPC delivery
  crew (Ketil, Marta, Lars) that runs a continuous loop visiting every
  vendor in both towns and triggering restock on arrival. Cycle takes
  roughly **one in-game day** (~1 hour real time). The caravan rests at
  the Thornwall Market Square depot, departs for Stillwater, visits
  each Stillwater vendor in order, rests at Stillwater's North Square,
  returns to Thornwall, visits each Thornwall vendor, then loops.
- Vendor mobs in caravan-served zones (Stillwater, Thornwall City) **no
  longer auto-restock** on a per-mob timer — they restock only when the
  caravan visits. Vendors in non-served zones (Watchers Crossing,
  Sanctum Basin, etc.) keep the legacy auto-restock unchanged. Both the
  non-crafter merchant tick and the crafter material tick respect the
  served-zone gate.
- The caravan crew can be examined and talked to but **cannot be
  attacked by players** — same rebuff as a shopkeeper. Wired into
  attack/bash/grapple/kick/shoot/taunt/throw/trip and steal commands
  via a new `Mob.PlayerAttackImmune` flag. Caravan crew will fight
  bandits along the road and have been statted to win.
- **Bandit pack at the North Road camp** (lookout, fighter, caster,
  Soren) detuned by ~25–30% across the board so the road is challenging
  but passable for solo and small-group players. The pack also picks
  up `hates: caravan` so they engage the caravan when it passes through
  4052 — every cycle the brawl plays out, the bandits respawn for the
  next pass.
- **New `caravan_step` btree action** drives the cycle (`internal/caravan`
  package owns route data + state machine; `actions_caravan.go` wires
  it into the behavior tree).
- **New config knobs** (`Balance.CaravanServedZones`,
  `Balance.CaravanDepotDwellRounds`) so cadence and zone coverage are
  tunable live.
- **`lookfortrouble` mob command extended** to scan for hostile mobs by
  group hate (in addition to the existing player + species-hate scans).
  Bandits with `hates: [caravan]` aggro on caravan-group mobs in their
  room.

## 2026-04-25 (late evening) — Stillwater Zone + Two Quests + Town Flavor + Engine Polish

### New Zone — Stillwater (Zone 2.2)

- **47 new rooms** opening north of Ashwick via a 10-room interlude
  (the Fernway road). Town spine: gate, lakefront square, Pike &
  Lantern inn, Brindle's smithy, constabulary, north square. Lake
  quarter: docks, Crab-Trap Beach, the Reedy Foreshore, boat pier,
  cave mouth, and a 4-room cave dungeon ending at the Hollow Sump.
  West quarter: cooper's lane, healer's cottage, Ulla's parlor +
  her late husband's untouched workshop, cemetery, sluice pond,
  watermill, tailor's cottage, travelers' camp, the old chapel
  ruin, the wardstone circle, and a boat-builder's yard.
- **22 named NPCs.** Innkeeper Sigrid, weaver Edda, smith Brindle,
  apothecary Ilsa, pearl-carver Kess, storekeeper Wulf, fishmonger
  Tov Brann, dock master Arn, constable Drunn, temple priest Seren,
  miller Bram, old fisherman Hodder, old cottager Gyda, Ulla, the
  child Pip, the caravan crew (Ketil + Marta + Lars), and assorted
  others. Each carries dialogue with cross-references to other
  townsfolk; gossipers (Hodder, Gyda, the barmaid Neva) broadcast
  world-event news on the engine's gossip system.
- **All seven crafting stations on-site:** forge (Brindle's smithy),
  alchemy bench (healer's cottage), loom (tailor's), cooking fire
  (Pike & Lantern + bakehouse), jeweler bench (pearl-carver's
  garret), enchanting circle (wardstone), watermill grain.
- **Sethome anchor:** `set home stillwater` respawns at the Temple
  of Stillwater (room 4123).

### Two New Quests

- **Quest 19 — The Lake-Caves Bounty** (combat / bounty). Constable
  Drunn posts an escalating bounty on the cave creatures spilling
  into the shallows. Five steps with multiple completion paths:
  partial reward (150g) for clearing the shrimp + drowned hunters,
  full reward (500g) for bringing back a leviathan tooth from the
  sump dweller boss. Both Drunn and dock master Arn will accept
  the tooth. Dialogue acknowledges your choice.
- **Quest 20 — Ulla's Silence** (lore / investigation). Ulla
  finally asks someone to look through her late husband Elgar's
  things in the workshop above her parlor. The trail leads through
  six spiral-marked sites across town and the western ruins — a
  pre-Chrysalis breadcrumb that Elgar was researching when he went
  into the deep. Smith Brindle was supposed to descend with him
  and didn't, and has been finishing the spear Elgar ordered ever
  since. The kingfisher Vella buried at the cemetery is for
  whoever knows enough to look. Eleven steps, single zone, with a
  flag-tracked ending choice (whole truth vs partial). Players
  who completed Quest 19 with the full path receive an extra
  acknowledgment from Ulla.

### Crafting & Items

- **Forage extension.** New `water` biome added to the foraging
  system; swamp + water yields extended with five new materials:
  cattail-down (40055), marsh-willow bark (40056), lake mint
  (40057), freshwater clams (40058), lake-iron nodules (40059).
- **Four new craftable recipes:** Lake-Iron Hook-Spear
  (blacksmithing), Lake-Tonic of Steady Hand (alchemy → buff 82
  Steady Hand), Cattail-Down Cloak (tailoring), Stillwater Lake
  Chowder (cooking). Plus three quest-related craftables:
  Hunter-Eel Scale Vest (skullduggery affix), Stillwater Pearl
  Pendant, and the Drowned-Veil enchantment (back-slot, conviction
  reservoir).
- **New cave drops:** skitter-shrimp shell, drowned-hunter hide,
  Stillwater black pearl (boss-drop, 15% chance), leviathan tooth
  trophy (boss-drop, 100% chance — the proof for the bounty).

### Engine & Balance

- **Vitality progression rebalanced.** `StatProgressionMultipliers.
  vitality` bumped from 1.0 to 4.0. Vitality has no high-frequency
  caller (no skill primaries it), so its actual call count was
  ~4-5x lower than other 1.0-multiplier stats — players who
  weren't tank-styled (taking constant damage to trigger the
  regen-progression curve) saw vitality lag behind every other
  stat. The 4x multiplier brings effective progression rate in line.
- **Per-item drop chance.** New `Item.DropChance` field plus
  `ShouldDrop()` helper. Mob suicide refactored so equipped and
  carried items use the same drop-gating helper. Lets boss
  signature drops (Stillwater black pearl) ride a 15% roll
  instead of always dropping.
- **Skill-affix progression patch.** `Character.GetSkillLevel()`
  now includes StatMod contribution — fixes the orphaned skill-
  affix path so item statmods like `skullduggery: +7` actually
  count toward the player's effective skill level. Volcanic plate
  and similar instance-loot skill bonuses now work as intended.
- **Mapper cache reload.** New `mapcache` argument to admin
  `reload` command flushes the per-zone mapper cache without a
  server restart — useful after editing room mapsymbol/maplegend
  fields.
- **Quest engine SOP hardening** (sketch-quest skill). New
  required gates: player-POV walkthrough per step, trigger-
  mechanic ranking table (★★★★ ask-quest down to ☆ unguessable
  magic words), thousand-mudder test, narrator-overreach guard,
  and the `consume_item` requirement on every quest-engine
  item_give trigger to suppress give.go's behavior-tree
  fallthrough (which was firing the noncombat archetype's
  "declines politely" emote AFTER quest acceptance).

### Story / World

- **"What the Moons Keep" — V3 polish pass.** The novel went
  through a multi-round adversarial review pass: Section 1-3
  critical/HIGH severity fixes, Round 2 voice consistency fixes,
  Round 3 verbal-tic sweep, V3 aggregate review, V3 polish
  pre-DE pass. ~270 lines added, ~300 lines removed across the
  full ~735KB manuscript.
- **Stillwater carries a quiet unsolved mystery.** The pre-
  Chrysalis spiral motif appears at five sites the player can
  find, plus a sixth in Elgar's workshop. The Voss family quest
  reveals what Elgar was researching, but not the deeper question
  of WHAT the symbol meant or WHO carved them. Sealed for future
  content.

## 2026-04-25 — Player Rename + Account Delete + Name Collision Prevention

### Gameplay

- **Players can now rename their character.** Use `rename <newname>` to
  request a new name. Rename is cooldown-gated (default 7 days, configurable
  via `Balance.CharacterRenameCooldownHours`). You'll see a yes/no confirmation
  (default no) before the change takes effect.
- **Players can permanently delete their character and free the username.**
  Use `deletecharacter` for a two-stage confirmation: first yes/no (default no),
  then type your exact character name to confirm. The deletion is immediate
  and irreversible — your user file is removed and the name becomes available
  for a new character.
- **Companion naming now correctly prevents mob-template collisions.**
  When naming a companion or pet, the system now checks that the name doesn't
  match any mob template in the world — previously this check was missing,
  allowing companions to shadow core NPCs or monsters.

### Engine & Balance

- **Centralized name validation.** All player, companion, character, and pet
  name checks now flow through `users.ValidateActorName()`, ensuring consistent
  rules across the game.
- **Boot-time name collision audit.** On server startup, if any mob template
  name matches an existing player character, the server logs a warning. This
  helps catch and prevent collisions in production without blocking startup.
- **Admin command rename → renameitem.** The admin `rename` command for items
  moved to `renameitem` to free up the `rename` verb for player use.

## 2026-04-24 (late night) — World Mob Audit Complete + Engine Polish

### Gameplay

- **Every zone now uses the behavior archetype system.** North Road,
  Thornwall City, Marches Spur Road, Ashwick, Watchers Crossing,
  Thornwall Outskirts, Dustwalk Road, and the Labyrinth of Low Tunnels
  joined Sanctum Basin and Ironwind Steppe in the migration. Mobs in
  these zones now react consistently — fighters pile on, lookouts
  call for help, leaders rally their packs, shopkeepers and questgivers
  decline politely instead of fighting back.
- **Pack reactions across the world.** New routines connect mobs that
  belong together: bandit packs on Marches Spur Road, the smuggler
  ring beneath Thornwall City, and the chrysalis-touched mobs in the
  lower district. Hit one and the rest hear it.
- **Thornwall City has its own ambusher.** The chrysalis skulker
  joins the cave's pale lurker and blind stalker as hit-and-fade
  predators — strike from hidden, flee on hurt, slip into shadow,
  strike again.
- **Wilderness now has prey.** Hares, grouse, sparrows, squirrels,
  toads, mice, chickens, and similar small wildlife flee instead of
  fighting. They're still attackable for food and reagents — they
  just don't stand and die anymore.
- **Sylara of the steppe now speaks naturally.** Her dialogue was
  written with third-person stage directions ("Sylara inclines her
  head...") that the engine spoke aloud. Rewritten to first-person
  speech with stage directions moved into the bracketed narrator
  hints where they belong.

### Fixes

- **Hidden-tag no longer lingers mid-combat.** Three fixes layered:
  (1) `CancelCombatBuffs` now strips permabuff entries with the
  cancel-on-combat flag — previously the active buff was expired but
  Validate re-applied it from the permabuff list. (2) The `camo-skin`
  mutation switched from granting a permanent `hidden` flag to a
  proper `stealth_bonus` (matches `chameleon-skin`'s pattern) —
  mutations no longer leak the hidden tag into combat text.
  (3) The buff system suppresses start text when refreshing an
  already-active buff, so ambusher idle ticks don't spam "{mob}
  disappears into the shadows" every round.
- **Surprise strikes now show their dedicated text.** The btree's
  `actAttack` was always setting `DefaultAttack` aggro — even when
  the attacking mob was hidden. Mobs now properly promote to
  `SurpriseAttack` when striking from hidden, triggering the
  `*[SURPRISE ATTACK]*` prefix and the backstab crit bonus that
  ambushers were supposed to get all along.
- **Ambushers attack proactively when a player is in their room.**
  Previous design only fired on `player_enter`; if a player came
  back to a room where an ambusher had re-hid, the ambusher just
  sat there. Now they fire surprise strikes whenever they find
  themselves idle, hidden, and with a player present.

### Behind the scenes

- Dead startland and tutorial zones removed (players couldn't reach
  them; default `StartRoom: 113` lives in Sanctum Basin). Mob IDs
  1–5, 57, 58 freed for reuse.
- Effective archetype coverage: 100% of stock-combat mobs across
  every zone. The handful of skips (Old Edrin, Olen, Phantom, Sable,
  Pell, Dal, loot goblin) all have custom per-mob behavior trees
  that override archetype anyway.

## 2026-04-24 (late evening) — Ironwind Steppe Audit + Boss Behaviors

### Gameplay

- **Cave stalkers now ambush from the dark.** Pale lurkers and blind
  stalkers spawn hidden, open with a surprise strike when a player
  enters their room, then flee the moment they take damage and
  re-hide in an adjacent room. Maximum-nuisance hit-and-fade cycle.
- **Stone Beetle Queen calls her swarm.** Boss behavior: when wounded
  or when one of her brood is hurt, she calls for help — pulling
  cave beetles from adjacent rooms. Vitality bumped to match her
  tank role.
- **Windscour Wyrm goes two-phase.** Above 50% HP the wyrm fights
  its slow, devastating baseline rotation. Below 50% HP it rages —
  tail-sweep knockdown rotations on every round. Vitality bumped to
  support the pacing.
- **Prey animals flee when hit.** Hares, grouse, lizards, squirrels,
  toads, moths, tumble beetles, and dry creek crayfish now retreat
  to an adjacent room when attacked instead of standing and dying.
  They remain attackable for hunting.

### Behind the scenes

- Two new behavior archetypes: `ambusher` and `prey`.
- Custom per-mob btrees for the Stone Beetle Queen (228) and
  Windscour Wyrm (229).
- Ironwind Steppe now has 43/43 archetype coverage.
- No engine changes — all behaviors reuse existing primitives.

## 2026-04-24 (evening) — Sanctum Basin Mob Audit + Tutorial Content

### Gameplay

- **Sanctum Basin NPCs now offer tutorial guidance for newer gameplay
  systems.** Each of the nine non-combat NPCs covers a curated set of
  topics through their dialogue: ask Korvath about salvage or
  enchanting, ask Yenna about potion aging or the bandolier, ask Saris
  about spell discovery or manifestation, ask the Combat Trainer about
  rally/warcry or companions, ask Fen about tracking or packs, ask the
  Warden about respawn grace or aggro, ask the Scholar about mutations,
  ask the Chrysalis Priest about the Awakening, ask Merchant Adela about
  bartering or encumbrance.
- **Non-combatants now react when you try to attack them.** Trying to
  attack (or target with a harmful spell) an NPC who cannot be attacked
  now triggers an in-character emote from that NPC — a raised eyebrow
  from a questgiver, a step back from a shopkeeper. Rate-limited to one
  reaction per NPC per round so companion and party auto-assist cannot
  spam it.

### Behind the scenes

- Four new behavior archetypes: `noncombat_questgiver`,
  `noncombat_shopkeeper`, `noncombat_passive`, `combat_passive`. Every
  Sanctum Basin mob is now tagged with a `behavior_archetype` value.
  This is the first zone in a larger migration to the archetype system.
- New btree event `player_attack_rejected` fired from attack.go and
  from HarmSingle spell rejection in cast.go.
- All tutorial content is delivered via dialogue YAML `patterns`, which
  is deterministic and prod-safe (no LLM dependency).

## 2026-04-24 — Discovery Rate Stat Offset

### Gameplay

- **Spell and recipe discovery now scales with Perception + skill.**
  The decay that slows discovery as you learn more spells/recipes
  is now partially offset by your Perception stat and the relevant
  skill (Spellcasting for traditional spells, Manifestation for
  manifestation-school spells, or the specific crafting skill for
  each recipe). A newbie discovers at the current rate; a seasoned
  character with invested Per + skill discovers roughly 1.8× faster
  at 20 known — closing the late-game discovery drought without
  flooding new characters with learn-messages.
- **Offset mechanic:** Per contribution reaches 1.0 at Per=300,
  skill contribution reaches 1.0 at rank 100, combined via
  `1 - (1 - per)(1 - skill)` and capped at 0.8 (effective decay
  floor = 20% of base). Either Per or skill alone gives a partial
  benefit; the combination unlocks the full cap.
- **Mobs benefit too.** Caster mobs with high Per + Spellcasting
  will expand their spell repertoire faster than before — a
  battle-hardened mob learning from repeated casts.

### Config

- New `Balance` knobs: `DiscoveryPerceptionScale` (default 200),
  `DiscoverySkillScale` (default 100), `DiscoveryMaxDecayOffset`
  (default 0.8). Set `DiscoveryMaxDecayOffset: 0` to disable the
  offset mechanic entirely and revert to the prior flat-chance
  formula.

## 2026-04-22 (evening) — Pack Tactics Revamp + QOL Batch

### Gameplay

- **Priests and unrelated civilians no longer aggro you after fighting
  bandits.** The old group-hostility system flagged every mob sharing
  any group tag (including taxonomic ones like `humanoid`) as hostile
  when you hurt a bandit. Bandits and temple priests both have
  `humanoid`, so attacking bandits made Olen the priest swing at you
  on sight. Replaced with a routine-scoped pack-reaction system:
  packmates now have to share a specific `routine` string
  (`bandit_camp_guard`, `wolf_pack_ironwind`, etc.) to react to each
  other being attacked. Priests, merchants, guards, and wildlife
  unrelated to your fight stay peaceful.
- **Packs coordinate via behavior trees now.** A bandit fighter,
  caster, lookout, and leader in the same camp respond to one of
  them being attacked the way their role suggests. Fighters pile in.
  Casters shield the tank, then heal the most-wounded packmate, then
  engage. Leaders open with rally + warcry self-buffs, then engage.
  Lookouts yell for help, then engage.
- **Pack cries carry to adjacent rooms.** Mobs in neighboring rooms
  whose routine matches the caller's now move toward the commotion.
  Previously each room's mobs stayed oblivious to a fight next door.
- **Charmed wild creatures no longer snitch on their pack.** A
  charmed mob counts as your companion, not a packmate, so the
  pack doesn't react when you fight its former brothers.

### Fixes

- **`lookfortrouble` now respects the respawn grace buff.** Mobs
  scanning for targets when you arrive in a new room will skip a
  grace-protected player entirely instead of picking you as the
  "best" target and attacking through the grace check in the combat
  pipeline. Closes the Duard prod repro where mobs started on a
  respawning player inside the 3-round grace window.
- **Thornwall — Rift Chamber no longer overlaps Records Office.**
  The Rift Chamber was geographically beneath the Temple District
  but was authored with east-exit adjacency to the Records Office,
  putting both rooms at the same coordinate on the mapper. Rift
  Chamber is now reached by `down` from the Temple District.
- **North Road — Bandit Camp no longer overlaps the Inn's common
  room.** The Camp Approach intermediate room had been bent south
  to dodge one overlap and ended up dropped on top of another.
  Removed the Approach entirely; the bandit camp now hangs off the
  main road at a single exit.

### Quality of Life

- **`craft` listing shows the enchant target slot.** Enchanting
  recipes now print `(targets: weapon)`, `(targets: body armor)`,
  etc. so you know which slot an enchantment will land on before
  you spend materials.
- **Companion roster shows mutations.** `companions` now lists each
  mutation your companion has acquired underneath its name, stats,
  and status line. `companions <name>` shows the full set in the
  detail view.
- **Progression dashboard shows exact rank and grandmaster tier.**
  The player overview on the skill progression web page now shows
  the raw rank next to the tier description, and skills above 75
  display as "Grandmaster" instead of falling off the top of the
  previous tier chart.
- **`identify` is in the starter spellbook.** New characters can
  identify unidentified items without waiting for a scroll drop or
  shop purchase.

## 2026-04-22 — Combat Bug Fixes + Novel Canon Correction

### Fixes

- **Two-handed weapons no longer grant a spurious fist attack.** When
  you equip a 2H weapon, the pair-partner slot (offhand for the main
  hand, extra arm 2 for extra arm 1, extra arm 4 for extra arm 3) is
  cleared to an empty-item marker. The attack-collection code read
  that as "empty arm → generate fist" and produced a bonus unarmed
  swing from the arm physically occupied by the 2H weapon. Most
  visible on a 2H + extra-arm-shield loadout: the bogus fist then
  swung 3–4× per round via the normal swing-count formula, producing
  noticeable extra unarmed hits every round. That arm is now
  correctly treated as occupied.
- **Flee now actually flees.** The flee command sets aggro to a
  Flee-type state with no target so the combat loop can run the
  escape attempt on the next round. The round-start aggro validator
  was invalidating that no-target aggro (only SpellCast was in the
  allowlist), and the fallback then grabbed a new target before the
  flee attempt could run — silently losing you a round of attacks
  and keeping you in combat. Flee is now on the allowlist alongside
  SpellCast.

### Novel

- **Bloom-harvest canon consistency fix in "What the Moons Keep"
  (Chapters 18 and 22).** The scenes showing the captive woman on
  the pallet described her as "heavily mutated" with elongated
  limbs and frond-fingers — contradicting the book's rule that
  *hollow* means the absence of the Chrysalis change, and that
  Bloom is produced from hollow people by definition. Rewritten to
  show her unchanged but in visible reaction to something her
  captors introduced: puncture marks at both elbows and the hollow
  of the throat, skin flushed and faintly raised at each site,
  sweat despite the cold stone. Vane's spoken summary to Maren and
  her Ch 22 return-visit observations updated to match. Two parallel
  audit agents (Ch 1-17 and Ch 19-30) found no other instances of
  the same contradiction.

## 2026-04-21 — Tank Companions, Death Cleanup, and Two Instance-Save Fixes

### Gameplay

- **Tank companions (flesh golem, earth elemental, magma elemental)
  now actually hold aggro.** Previously their taunts missed ~70% of
  the time because their stat distribution and base charisma were
  tuned for general brawlers, not front-liners. Two changes stack:
  - A new "tank" stat archetype allocates 25% Charisma and 20%
    Vitality out of the stat pool (up from ~7% Charisma under the
    old "fighting" archetype).
  - Species base Charisma raised on all three: flesh golem 5→115
    (top-tier raised pet), earth elemental 5→70, magma elemental 5→80.
    These are imposing, otherworldly creatures; low-single-digit
    Charisma was a default that never got tuned.
- **Tank + generic fighter companion AI archetypes.** Flesh golem,
  earth elemental, and magma elemental run a tank routine:
  interrupt-on-cast, taunt-if-not-holding-aggro, bonus-damage kick
  when the target's prone or clinched, rally + warcry self-buffs,
  then a bash/grapple/trip knockdown cascade. Steppe spirit wolf and
  zombie run a simpler generic CC rotation. Tank archetype uses the
  new generic "bellows a thunderous challenge" taunt text instead
  of the wolf-themed howl.
- **Death no longer kills you twice.** Three long-standing respawn
  bugs fixed together:
  - **Poison, bleeding, and other conditions now clear on death.**
    Previously a poisoned player respawned at 5% HP still poisoned —
    the next DoT tick killed them again. Now the Conditions slice
    wipes alongside buffs.
  - **Inbound aggro clears on death.** Every mob in your combat room
    that was targeting you ends aggro when you die, and your
    companions' aggro clears too. Respawn arrives in a clean slate.
  - **3-round grace period post-respawn.** New `Respawn Grace` buff
    (id 81, NoAggroTarget flag) prevents mobs from acquiring aggro
    on you for 3 rounds after respawn. Configurable via
    `Death.RespawnGraceRounds` (default 3; 0 disables). PvP
    griefers: you can no longer chain-kill a respawning player.
- **Single combat hit no longer applies stacked penalties.** A
  pre-existing bug: the combat loop could queue the death-processing
  `suicide` command multiple times in the same round, applying two
  (or more) separate stat-decay + skill-rust rolls from one death
  event. A round-based dedupe flag (`Character.LastSuicideRound`) now
  guards against this.
- **Crafting blocks all 7 combat commands consistently.** Previously
  only rally and warcry checked `IsCrafting`; bash / trip / grapple /
  kick / taunt would let you swing your way out of a craft. All 7
  now universally reject with a friendly "focused on your work" message.
- **Tank companion rally/warcry don't burn cooldowns re-casting.**
  The behavior tree now skips rally/warcry when the buff is already
  active on the companion; they move on to other moves in the
  priority list instead.

### Fixes

- **Sable portal and other room-exit routes no longer break
  randomly.** An asymmetric bug in the room instance-save system:
  the save side correctly skipped structural fields (exits,
  description, nouns, etc.) via `instance:"skip"` struct tags, but
  the load side used raw `yaml.Unmarshal` which doesn't see the tag.
  Pre-fix corrupt files kept overwriting the template on every
  load. `LoadRoomInstance` now restores skip-tagged fields from a
  fresh template copy after the overlay unmarshal.
- **Summoned companions always start fresh.** The old
  `mobs.instances/summons/` file-based persistence was keyed by
  room, not by owner, and leaked progression across
  summon-dismiss-resummon cycles and across players. Removed the
  file layer entirely; companions now persist only via
  `CompanionInfo` on the owner's user YAML.

### Under the hood

- **`actions.CommandIsReady` is the single source of truth for
  combat-command gating.** New btree action `command_best_of` used
  by the archetypes queries CommandIsReady before issuing; a drift-
  detection test enforces that CommandIsReady and each Execute*
  agree on readiness. Signature flipped from `*mobs.Mob` → `Actor`.
- **New "tank" stat archetype** (internal/mobs/mobs.go) — 25% Cha,
  20% Vit, 15% each Str/Dex/Wil, 10% Per. Joins the existing
  "fighting" and "casting" archetypes.
- **`NoAggroTarget` buff flag + `characters.SetUserUntargetableCheck`
  callback.** Avoids the users↔characters import cycle; follows the
  same pattern as `rooms.SetCompanionTransport` and
  `rooms.SetBTreeStateEvictor`.
- **Tank-taunt aggro-pull now works for mob taunters.** Was
  previously gated on `attackerUserId > 0` (player-only); now also
  handles the mob-taunter path via `actor.GetMobInstanceId()`, so
  companion tanks' taunts actually switch the target's aggro.

## 2026-04-20 — Companion AI Overhaul (Two Archetypes + Follow + Regen)

### Gameplay

- **Summoned companions now fight intelligently.** Two new AI
  archetypes cover the five mage-crafted summons:
  - **Melee self-buffers** (vampire, fire elemental): maintain
    offensive + defensive self-buffs, then attack with bite /
    flavor moves between casts. A fresh vampire's first combat
    sequence casts conviction-surge, then iron-will, then
    conviction-ward across the first ~10 rounds.
  - **Pure casters** (wraith, spectre, air elemental): emergency-
    heal when HP drops below 40%, maintain defense, cast AoE
    damage when enemies are grouped, else single-target harm.
    Watch for heal, sparks, conviction-barrage, mind-spike,
    conviction-spike, and nerve-disruption depending on the mob
    and situation.
- **Air elemental reclassified as a caster.** Its stats (dex 20,
  perception 20, willpower 10) were always caster-shaped; this
  update gives it the spellbook and archetype to match.
- **Companions now follow their summoner through every movement
  path** — walking, recall, portal, fold-recall, sable, admin
  teleport. Mid-cast wind-ups abort cleanly to keep up (conviction
  spent during a cancelled cast is forfeit, same as a player
  self-interrupt). Aggro on a target no longer in the new room
  ends automatically.
- **Thornwall Temple (and other sanctuary rooms) now heal your
  companions too.** The 5x regen boost in the temple previously
  only applied to players; your pet sitting next to you at Olen's
  altar got normal regen. Now it matches yours. Same fix extends
  to the Sanctum Basin tutorial rooms and the testing arena.

### Under the hood

- **Behavior-tree archetypes are now a first-class concept.** Mob
  YAML gains a `behavior_archetype: <name>` field that resolves to a
  shared tree file at `behaviors/archetypes/<name>.yaml`. Resolution
  order: per-mob btree file wins, then archetype, then legacy. This
  unlocks future work where NPCs can switch archetypes at runtime
  (e.g., a caravan guard taking up banditry).
- **Spells carry `categories:` tags.** Free-form strings used by
  archetype AI to filter a mob's spellbook by purpose:
  `self_defense`, `self_offense`, `self_heal`, `harm_single`,
  `harm_multi`. Applied to 12 existing spells.
- **New btree action `cast_best_in_category`** picks the
  highest-scoring (`base_folds × cost`) spell in a category from the
  mob's spellbook, skipping already-active buffs, components,
  summons, and insufficient CP. Self-gates on the shared
  special-move cooldown, so mobs naturally alternate between cast
  rounds and attack rounds.
- **New event `mob_combat_round`** fires per mob combatant BEFORE
  legacy AI, so archetypes are authoritative for mobs that declare
  one. Legacy `preferredSpell` (shield-first priority) no longer
  preempts the archetype.
- **`multiple_enemies` btree condition is now perspective-aware.**
  A summoned caster no longer treats its summoner and fellow
  companions as "enemies" when deciding AoE vs single-target. Wild
  mobs (like bandit_leader) preserve original behavior.

### Latent engine fixes surfaced by this work

- **`applyMobSelfEffect` now handles `buff` effect_type.** Used to
  only handle heal and shield — buff-type spells (conviction-surge,
  iron-will) fell through silently when mobs self-cast them. Now
  they correctly apply, including per-buff tick snapshots.
- **Shield-active detection uses `ConditionShield`** instead of the
  equipment-layer `HasShield()` helper. The magical shield a spell
  applies is tracked as a condition, not as worn equipment.

## 2026-04-18 — Combat unification, target resolution, bleedout removal, lots of fixes

### Gameplay

- **Bleedout removed.** Health <= 0 = dead, for both players and mobs.
  No more "downed" two-tier rule, no PlayerDrop event, no
  CoupDeGraceRounds. One-shot kills and DoT kills are now possible.
  ~270 lines of bleedout-specific code removed.
- **Death respawn at 5% of max pools (was 100% since the shadow-realm
  removal).** Restores the "death run brake" that was unintentionally
  dropped during the JS Audit Phase 4c. Respawning weakened means you
  have to recover before your next attempt at whatever killed you.
  Configurable via new `Death.RespawnPoolFraction` knob; per-pool
  minimum of 1 so respawn doesn't immediately re-trigger the death
  check. Operators who want full restore can set 1.0.
- **PvP combat gains parity with PvE.** As part of the
  combat-quadrant unification (see below), PvP now correctly applies:
  - Adrenaline buff (low-HP stamina boost)
  - Return damage (thorns / spikes / spell deflection feedback)
  - Lifesteal (vampiric weapon enchant)
  - Moon-mod stat shifts on mutated combatants
  These were all missing from the legacy PvP-only handler.

### Combat & AI Fixes

- **MvM (mob vs mob) parity gaps closed:**
  - Defender mobs now receive `OnCritReceived` on crit hits (PvM/MvP/PvP
    already did).
  - Attacker mobs now fire `OnCriticalSuccess` / `OnCriticalFailure`
    callbacks on crit rolls (was only firing OnSkillUse).
  - Attacker mobs now emit room-visible stat-gain messages when
    `OnStatUse` returns true (was discarding the bool).
- **PvM defender `combat_start` AI signal preserved.** Previously
  emitted in a function that the unification was about to delete; now
  in the unified resolver, gated to PvM only. Reactive AI for first-
  round mob deaths no longer silently breaks.
- **Legacy MvP ConditionShield double-dip removed.** Player defenders
  with Minor Shield were getting magnitude/2 reduction applied on top
  of the magnitude already counted by the mitigation layer. Stage 11.4
  leftover from before mitigation was unified. Single application now.
- **Crit feedback on PvM/MvP no longer drops attacker text.** The
  return-damage room broadcast in PvM correctly excludes the
  attacking player from the third-person message.
- **Edrin engages after his revelation.** Was firing `combat_start`
  once and then sitting idle while you fought his elementals. Now has
  a `single_target` fallback tactic.
- **Behavior tree `hostile` param is now a real bool** (was string
  `"true"`). Backward compatible via `getBoolParam` helper that
  accepts both forms.
- **Knuckles only progress unarmed-combat.** Dual-wielding knuckles
  was incorrectly triggering weapon-combat progression alongside
  unarmed-combat. Extracted `isDualWieldingWeaponCombat` helper that
  checks at least one weapon routes to weapon-combat before granting
  its progression.
- **Dismiss is peaceful for crafted companions.** Mage-crafted
  companions (Summoned, Conjured, Raised) dissolve immediately
  instead of going hostile and lingering for 5 minutes. Charmed wild
  creatures keep the betrayal-turns-hostile behavior (thematically
  correct).

### Spell Fixes

- **Summon spells check their component before casting.** Previously
  summon-steppe-spirit / raise-* / conjure-* validated only their
  ComponentTag, missing SummonComponentId. The full cast animation
  ran and consumed conviction before failing at resolution with
  "You lack the required component." Now caught at cast init.
- **Fumbled spells no longer succeed.** Summon, charm, fold-anchor,
  fold-recall, and purge-affliction used to run their primary effect
  even on a fumbled cast (you took backfire damage AND got the
  summon). Now the fumble cleanly aborts the effect; component
  consumption stays unconditional (failed binding ate the catalyst).
  Covers 13 summon spells + charm + 3 Go hooks.
- **`ConditionRegen` heal-tick text** now emits per-tick "wounds knit
  closed" feedback while the regen is active.

### Commands

- **`rally` and `warcry` no longer slip through during crafting.**
  Both now check `IsCrafting()` and refuse with a thematic message,
  matching the `craft.go` re-entry pattern. (Broader audit of other
  active commands tracked as future work.)

### Refactor: Combat-quadrant unification

- **Four parallel `handle{P,M}vs{P,M}` combat handlers collapsed into
  a single `handleCombatRound(atk, def actions.Actor, ...)`** in
  `internal/hooks/NewRound_DoCombat_unified.go`. Eight named phase
  helpers (target resolve, wait round, attack roll, damage bonuses,
  crit + messaging, progression, behavior trigger, aggro + assist,
  round resolution). Routing strategy: `IsPlayer()` checks at leaf
  sites where divergence is required; no Quadrant enum.
- Future parity gaps are now structurally impossible — any new
  combat logic added to the unified handler applies to all four
  quadrants by default. Quadrant divergence requires explicit
  `IsPlayer()` gating + reason comment.
- New cross-package test harness:
  `behaviortree.SetMobTreeForTest`,
  `items.SeedAttackMessagesForTest`. Structural routing test drives
  all four quadrant pairs (UU / UM / MU / MM) through
  `handleCombatRound` end-to-end.

### Refactor: Target resolution

- **`actions.ResolveTargetActor(room, name, opts...) (Actor, error)`**
  consolidates the `room.FindByName + GetInstance + nil-check` chain
  that was reimplemented ~37 times across user/mob commands with
  subtle variations. Sentinel errors (`ErrTargetNotFound`,
  `ErrTargetVanished`) let callers give precise error messages.
- New `actions.NewUserActor` / `NewMobActor` /
  `NewUserActorInRoom` / `NewMobActorInRoom` constructors.
- Closes the latent-nil-crash class (e.g., `attack.go:27` was
  derefing a nil mob via `m.Character.Aggro` unguarded).
- Two minor UX wins fall out: `ask <player>` now errors cleanly
  ("You can't ask another player.") instead of silently
  fall-through; `party invite <mob>` errors cleanly ("You can only
  invite players to your party.") instead of "Something went wrong."

### Refactor: Rooms package

- **`AddTemporaryExit` now correctly enforces no-overwrite contract**
  while explicitly allowing the legitimate ephemeral-portal →
  ephemeral-portal overwrite case. Closes a long-standing failing
  test from the Stage 1.5 audit.
- **Instance cleanup chain consolidated in `CheckPortalTimers`** —
  TTL-triggered chain (boot players → Remove ephemeral rooms →
  EvictRoomBTreeState via callback → TryEphemeralCleanup) replaces
  the deleted CleanupEmptyInstances. Resolves the catch-22 where
  ephemeral rooms couldn't garbage-collect while their instance
  stayed registered.
- **`behaviortree.EvictRoomBTreeState` wired up via callback in
  `main.go`**, avoiding the rooms→behaviortree import cycle.

## 2026-04-17 — Code Cleanup Stage 1 complete (1.2a, 1.5, 1.6, 1.8)

**Stage 1 of the code cleanup roadmap is now complete (substages
1.1–1.8).** Four substages this day; see prior notes for 1.1 / 1.2b
/ 1.2c / 1.3 / 1.4 / 1.7.

### Stage 1.2a — Combat + Spell god-function refactor

- `handlePlayerVsMob` 286 → 39 lines.
- `handleMobVsPlayer` 236 → 82 lines.
- `applyMobEffect` 246 → 26 lines.
- New `internal/hooks/NewRound_DoCombat_resolution.go` holds combat
  phase helpers; spell case helpers inlined.
- Includes parity fix: PvM return-damage room broadcast now excludes
  the attacker (MvP already did).
- Removed dead "tame" EffectType (superseded by charm).

### Stage 1.5 — Error Handling Audit

- Audited code paths added after Phase 37.3a/b sweep.
- 3 Critical fixed: `spell_purgeaffliction` nil guard, two unsafe
  type assertions in `NewRound_BroadcastHints` and
  `RedrawPrompt_SendRedraw`.
- `mudlog.SetupLogger` panics → `log.Fatalf` (only intentional
  behavior change — the panic was uncatchable).
- Sable portal refund paths + admin dashboard nil-checks +
  behaviortree codebase all verified clean.

### Stage 1.6 — Test Coverage for New Systems

- 24 additive Go unit tests across 4 files: 6 room btree engine,
  7 Phase 4c conditions, 9 Phase 4c actions, 1 actSummonCompanion
  hostile, 1 give.go quest-engine vs btree handoff regression.
- Zero production code change.

### Stage 1.8 — Behavior Tree Engine Robustness

- **Panic-safe `DrainQueue`** via `safeExecuteDelayed` wrapper.
  Panics in delayed-action closures (typically caused by closures
  over destroyed mobs/rooms/users) are now recovered and logged
  at `mudlog.Error` instead of crashing the engine round tick.
- **`EvictRoomBTreeState(roomId)` API** with no-op-on-missing
  semantics. Wired up via callback in 2026-04-18's rooms-package
  pass.
- Negative-cache hot-reload assumption documented with
  `TODO(hot-reload)` marker. (No hot-reload exists today, so the
  cache is correct; comment for future-you when it's added.)

## 2026-04-16 — Code Cleanup 1.7: Performance Pass + Bug Fixes

### Performance (Stage 1.7)
- **Zone-activity lane split:** mobs in zones with zero players skip
  combat, progression, mutation acquisition, charm-state, and crafting
  every round. Idle mobs still tick cooldowns, buff/condition durations,
  charm duration, combat-memory expiry, and death checks so timers and
  DoTs keep working. Active-zone behavior is unchanged.
- **Registry mutexes:** `internal/mobs` and `internal/users` global maps
  now use `sync.RWMutex`. Closes a latent race between the HTTP admin
  dashboard and the main game loop.
- **PruneVisitors fast path:** empty-map early return on room cleanup.

### Bug Fixes
- **Companions no longer pack-scatter with wild mobs** — the prior
  partial fix only guarded the death-triggered flee. Per-round pack
  membership now skips charmed/summoned mobs entirely, so your pet
  elemental stays put when a wild pack alpha dies nearby.
- **Merchants can no longer be killed via group hostility** — when you
  attacked a combat mob that shares a faction group with a merchant
  (e.g., both in "townfolk"), the merchant was picking up aggro on
  their next room entry or behavior-tree tick. Non-combatants now
  ignore group hostility entirely, matching the direct-attack guard
  that was already in place.
- **Vitality progression farm via HP reservation closed** — regen-based
  stat progression used the full HealthMax as its denominator, but
  Chrysalis-enchantment HP reservation clamps current HP to a lower
  effective cap. Result: a player at "effective full" HP still
  counted as depleted and rolled vitality progression every 3 rounds
  forever. The regen calculation now subtracts reservation from the
  max, so at effective cap you hit the proper short-circuit.

## 2026-04-15 — Bugfixes & QOL (Hotfix 2)

### Bug Fixes
- **Companions can no longer be targeted by other players** — taunt,
  attack, target, and throw now block ALL companions, not just your
  own.
- **Mobs no longer sit downed for minutes** — dead mobs at 0 HP are
  now swept every combat round. Fixes dismissed companions and DOT
  kills lingering in downed state.
- **Buff text tokens fixed** — meditating, illumination, stunned,
  blinded, hidden now show character names instead of raw
  `{sourcename}` tokens.

### Security
- **Bcrypt password hashing** — ported from upstream GoMud. Replaces
  unsalted SHA256. Existing players migrate transparently on login.
  Plaintext and hash-of-hash bypasses removed. File permissions
  tightened (0777 to 0600).

## 2026-04-15 — Bugfixes & QOL

### Bug Fixes
- **Spell typos no longer waste cooldown** — casting at an invalid
  target or misspelling a spell name no longer triggers the special
  move cooldown timer.
- **Assess without corpse no longer wastes cooldown** — same fix
  for the assess command.
- **Companion autoassist toggle now works** — `companion <name>
  assist off` actually prevents the companion from joining combat.
  Fixed in all three engagement paths (player attacks, mob attacks
  player, PvP).
- **Tutorial NPCs are now non-combatant** — 8 Sanctum Basin quest
  NPCs (priest, trainer, korvath, yenna, fen, saris, warden,
  scholar) can no longer be attacked or stolen from.

### Balance
- Skill progression multipliers increased:
  weapon-combat and unarmed-combat +25%, spellcasting +25%,
  manifestation +25%, rhetoric +15%.

### Admin Tools
- **Player stats table** added to the progression dashboard's
  Player Overview tab (base+training values, color-coded).
- **File-based logging** — new `Logging` config section in
  config.yaml. Logs to both terminal and rotating file when enabled.

## 2026-04-15 — JS Audit Complete (Phases 4-5)

### Death System Simplified
- **Death no longer sends you to the Shadow Realm.** When you die,
  you respawn directly at your home location with full health,
  stamina, and conviction. Stat decay and skill rust penalties
  still apply.
- Type `sethome` to set your respawn location.
- Type `help death` for updated details.

### Mob AI: Behavior Trees
- **7 mob scripts migrated** to YAML-based behavior trees with
  perception-scaled reaction delays. Smarter mobs react faster.
- **Old Edrin upgraded** — multi-phase caster boss with elemental
  summons, reveal sequence, and tactical combat AI.
- **Chrysalis Phantom upgraded** — hit-and-run assassin with
  surprise strikes, flee-and-rehide loop, and target tracking.
- **Barmaid Dal** now heckles back at tavern patrons with 4
  randomized NPC-NPC interaction sequences.

### Room Behavior Trees (New System)
- **14 room scripts migrated** to room behavior trees — a new
  first-class system for room-level AI and event handling.
- **Sanctum Basin tutorial** fully converted — all 9 tutorial rooms
  now use room behavior trees for quest progression, command
  detection, and the Awakening Rite ceremony.
- Room behavior trees support command interception, timed NPC
  dialogue sequences, and room-scoped state.

### Spell & Buff Migration
- **Fold-anchor and fold-recall** moved to native Go hooks.
- **Purge-affliction** moved to native Go hook.
- **Chrysalis-aid removed** — vestigial resurrect spell (death
  system handles respawn). Pruned from spellbooks automatically.
- **Buff flavor text** (illumination, stunned, blinded, hidden,
  meditating) moved to YAML text fields.

### Sable Portal Vendor
- **Sable migrated** from JS to behavior tree with a new
  `open_instance_portal` action for creating instanced zones.

### JS Scripting Bridge Removed
- **The entire JavaScript scripting system has been removed.**
  The goja JS engine, all 152 JS files, and the Go/JS bridge
  code are gone. All game logic now runs natively in Go via
  behavior trees, Go hooks, quest engine triggers, and YAML.
- Admin `mob create` and `spell create` commands removed (content
  is hand-authored as YAML files).

### Bug Fixes
- Fixed quest NPC item delivery — quest engine triggers now
  properly grant rewards without behavior tree interference.
- Fixed inventory stacking by base item instead of enchant state.
- Fixed enchanting preservation of instanced zone affix bonuses.
- Fixed instance portal replacement for difficulty upgrades.
- Fixed deep copy of item slices in NewMobById (prevents shared
  state corruption between mob instances).
- Fixed raise spells accepting generic corpse targeting words.
- Fixed visual room broadcasts routing through darkness filter.

### Balance
- Mob reaction delay curve tuned — base 2.0s, max 2.0s (was 3.0s
  base, 4.0s max). Perception 100 yields ~1s delay.

## 2026-04-11 — JS Audit Phases 1-3

### Phase 3: Item Cleanup + Charm Migration
- **11 dead default item JS files deleted** — all shadowed by DOGMud
  items at the same ID or completely unused.
- **Herbalism recipe page** migrated to YAML `on_use_train_skill` field.
- **Charm spell ported to Go** — opposed roll with charisma+manifestation
  vs willpower+statpool, aggro penalties, companion registration. The
  last non-mob/room spell JS file eliminated.
- Net: 13 JS files deleted, ~380 lines removed.

### Phase 2: Companion Consolidation + Config-Driven Buff Ticks
- **13 companion spell JS files replaced** by one Go function with YAML
  config (summon_mob_id, summon_base_pool, etc.). Conjure, raise, and
  summon spells all use the same code path now.
- **~10 healing/DoT buff JS files replaced** with YAML tick config
  (tick_pool, tick_percent, tick_variance). Buff tick magnitude now
  scales with caster spellcasting skill for spell-cast buffs.
- **Chrysalis-construct spell deleted** (redundant, undiscovered).
- **Minor antidote** migrated via `start_remove_buffs` field.

## 2026-04-11 — JS Audit Phase 1: YAML Text Fields

### Code Cleanup: Spell & Buff Text Migration
- **60 JS files deleted** — flavor-only spell and buff scripts replaced
  by YAML text fields on the data definitions. No gameplay changes;
  same messages, same colors, now driven by data instead of scripts.
- **13 stub room scripts deleted** — empty JS files that did nothing.
- **20 complex spell/buff scripts slimmed** — flavor text extracted to
  YAML, logic (companion spawning, charm, teleport, healing ticks)
  remains in JS.
- **New `textutil` package** — centralized token substitution
  (`{source}`, `{target}`) and text dispatch for spell/buff messaging.
  Sets the stage for ANSI-aware line wrapping in a future pass.
- **Schema docs updated** — spell and buff schemas now document YAML
  text fields. JS files are optional for flavor-only spells/buffs.
- Net result: 185 files changed, ~60 fewer lines of code, 60 fewer
  files to maintain.

## 2026-04-10 — Instanced Zones: Arena, Planar Oasis, Randomized Loot

### Zone 2.1b: North Road — River Approach
- **10 new rooms** extending the North Road northward through river
  country toward Stillwater. Stone bridge, river ford, woodcutter's
  camp, travelers' rest, and the first glimpse of Stillwater on
  the horizon.
- **Woodcutter Hagen** — NPC at the camp with dialogue about
  Stillwater, the road, and bloodline agents seen heading north.
- **Lone Traveler** — NPC at the rest stop who foreshadows Maren's
  trail and describes Stillwater.
- **River rats** and **wild dogs** as ambient hostile wildlife.
- Milestone at Wide Bend reads "Stillwater — 2 leagues."

### Hot Fixes
- **Server crash fix**: `look` command on a mob with a stale instance
  reference no longer crashes the server. Added nil checks to
  consider, locate, and mob-look commands as well.
- **Taunt exploit fix**: can no longer taunt your own companions.
- **Crafting vendors** now know all recipes in their profession
  (Voss buys moonpetal/veilbloom, Kerra buys steel, etc.).
- **Taunt exploit fix**: can no longer taunt your own companions.

### New: Instanced Zones
- Pay the **Riftkeeper Sable** (Rift Chamber, east of Temple District)
  to open a private portal to a dangerous zone. More gold = tougher
  enemies = better loot. Party up before purchasing — only current
  party members can enter.
- Portals last 30 minutes. Instances persist until all players leave.
- **`help instances`**, **`help arena`**, **`help oasis`** for details.

### New: The Arena (Instance Zone)
- A linear gauntlet of pit fighters. Push through trash mobs,
  veterans, and the Arena Champion.
- **Death ends your run.** No recall. No re-entry.
- Enemies respawn in waves — how far can you get before they
  grind you down?
- Veterans and the Champion drop unique weapons and armor.

### New: The Planar Oasis (Instance Zone)
- A 4x4x4 wrapping cube of elemental terrain — 64 rooms where
  every direction wraps around. Navigation is the challenge.
- Elementals wander the maze. Two elite elementals and one
  elemental lord (king, queen, or prince) roam randomly.
- **Death allows re-entry.** Recall works. Guardians don't respawn
  — clear the cube methodically.
- Oasis gear is stronger than Arena gear.

### New: Randomized Loot System
- Tougher instance mobs spawn wearing randomly-generated equipment.
  The gear makes them harder to fight AND drops when they die.
- **Point budget system**: gold invested determines a bonus pool that
  is randomly distributed across damage, mitigation, stats, and skills.
- Items are prefixed by their dominant bonus: Keen (damage), Warding
  (mitigation), Empowered (stats), Masterwork (skills).
- Every item is unique — two runs at the same gold level produce
  different gear.
- Weapons favor damage bonuses, armor favors mitigation. A snowball
  effect creates focused items rather than thin spreads.

### New: Rift Chamber
- New room in Thornwall (east of Temple District) housing the
  Riftkeeper NPC and the rift archway.

### Balance
- Mob regen and companion scaling changes from 2026-04-09 are
  included (mob SP/CP regen 2%/tick, companion stat factor 150).

### Instance Framework
- Party-scoped access control — only authorized players can enter.
- Death policy per zone: ejected (arena) or rejoin (oasis).
- Recall blocking per zone (arena blocks, oasis allows).
- Difficulty scales linearly with gold (stat pools = gold * template
  multiplier).
- Instance cleanup when all players leave.
- Portal timer warnings at 5 minutes and 1 minute remaining.

---

## 2026-04-09 — QOL Batch, Grenades, Rhetoric Shouts

### New: Grenade System
- **Three throwable grenades** crafted via Alchemy:
  - **Flashbang** (Alchemy 35) — AoE stun + blind
  - **Firebomb** (Alchemy 25) — AoE physical damage
  - **Toxic Flask** (Alchemy 20) — AoE poison DoT
- New `throw` command: AoE opposed roll (Dex+Skullduggery vs
  Dex+Perception). Fumbles hit the thrower. Shares special move
  cooldown. Progresses Skullduggery and Dexterity.
- **Grenade aging**: grenades grow more potent over time, then
  decline and eventually spoil. Spoiled grenades in the bandolier
  are safely ejected to the ground. Spoiled grenades in your
  backpack have a chance to detonate when you check inventory!
- New material: **Putrid Residue** — salvaged from spoiled food.

### New: Rhetoric Shouts
- **Warcry** — AoE offense buff. Boosts physical damage for you,
  companions, and party members in the room. Scales with Rhetoric
  and Charisma (5-20%).
- **Rally** — AoE defense buff. Boosts dodge, parry, and block for
  all allies in the room. Same scaling curve.
- Both share the special move cooldown. Can be used before combat
  to pre-buff your group. Progress Rhetoric and Charisma.

### New: Tank Taunt
- Successful taunt now **forces the target to switch aggro** to the
  taunter. Essential for protecting companions and party members.
- Flesh Golem companion now taunts in combat — the "tank pet."
- New flavor text for aggro pulls.

### QOL Improvements
- **Sort** now moves potions AND grenades into the bandolier.
- **Sell** searches bandolier and component bag as fallbacks.
- **Auto-eject**: spoiled potions move to backpack, spoiled grenades
  drop safely to the ground.
- **Food spoiling**: crafted food now ages and can spoil. Spoiled
  food cannot be eaten but can be salvaged for putrid residue.
- **Inventory** label changed from "Potions:" to "Bandolier:".

### Balance
- **Mob regen**: stamina and conviction regen doubled (1% → 2% per
  tick) to match player rates. Reduces chip-away tactics on tough
  mobs and helps companion sustainability.
- **Companion stat scaling**: Charisma factor improved (divisor 500
  → 150). Companions get meaningfully larger stat pools.

### Bug Fixes
- Companions can no longer be ordered via `ask`. They respond with
  a blank stare instead. Companion help updated accordingly.
- Farmer and bloodline agent now wander the North Road as intended.
- Rhetoric help file now lists warcry and rally.

---

## 2026-04-08 — North Road Zone, Progression Balance, Bug Fixes

### New Zone: North Road — Southern Stretch
- **15 new rooms** west of Ashwick Crossroads: farmland road,
  crossroads village with tavern, Betta's farmstead, bandit camp.
- **Quest: The Caravan Guard** — deal with a bandit crew
  threatening the road. Kill their leader Soren and bring his
  iron pin back to the caravan master for a reward and a trade
  contact in Stillwater.
- **Quest: Betta's Silence** — discover a trace of a recent
  traveler in a farmstead barn. Betta asks you to keep quiet
  when a bloodline agent comes asking questions. Your choice
  has consequences.
- **12 new NPCs**: Corvin (whittler boy), Betta (taciturn farm
  wife), Haral (tavern-keeper with shop), Old Dessa (gossiper),
  Tam (loud farmhand), Caravan Master, ambient farmer, 4 bandits
  (coordinated group fight), roaming bloodline agent.
- **New loot**: Soren's Ironbound Buckler (mid-tier shield),
  bandit longsword, leather vest, lockpick set.
- Bandits fight as a coordinated unit (no pack scatter).
- Bandit lookout spawns hidden — high Perception detects on entry.

### Engine Improvements
- **pack_flee_immune** mob flag: mobs hold their ground when
  packmates die instead of scattering.
- **Dialogue root variants** now support grantsQuest, givesItem,
  and setsQuestFlag — quests can complete on NPC greeting.
- **Mob death quest notifications** moved to dedicated listener
  (was buried inside PackFlee handler).
- **2H weapon + shield equip bug fixed**: shields could be equipped
  in offhand alongside 2-handed weapons, applying invisible stats.
- **Level 4 mutation display**: now shows "extreme" instead of
  "unknown" for max-level mutations like Extra Arms.

### Difficulty-Scaled Skill Progression
- Spell and crafting skill progression now scales with difficulty.
  Harder spells and higher-tier recipes give proportionally more
  skill growth. Utility spells like Identify still progress skills,
  just slower than combat spells.
- Self-cast buff spells give reduced progression (50% by default).
- AoE spells cast in empty rooms no longer give progression.
- Spells no longer fire progression twice (was triggering at both
  cast start and spell resolution).
- Three new config knobs: `SpellDifficultyProgressionScale`,
  `CraftDifficultyProgressionScale`, `SelfCastProgressionMultiplier`.

### Spell Difficulty Pass
- All spells now have meaningful difficulty values (0-75 range).
  Previously most spells had difficulty 0.
- Removed Empathic Bond spell (redundant with Charm).
- `spells` command now shows Difficulty instead of Familiarity.
- Spell list sorted by category (utility → heals → buffs → damage
  → summon), then target scope, then difficulty.
- Neutral-type spells (conjure, raise, identify) now show "Self"
  instead of "Unknown" for target type.

### Bug Fixes
- **Companion corpse re-raise exploit**: dismissed companions can
  no longer be killed and re-raised via necromancy.
- **Condition duration display**: recasting a spell now correctly
  shows refreshed duration instead of stale "fading" text.
- **Multi-buy progression**: buying multiple items now triggers
  charisma and bartering progression for each item purchased,
  matching individual buy behavior.
- **Bartering skill**: bartering now actually progresses during
  buy and sell transactions (was never triggered before).

### Balance Tuning
- Vitality progression: crit-received base chance increased from
  5% to 25%. Regen progression base doubled (0.005 → 0.01).
- Weapon-combat and unarmed-combat skill progression multipliers
  increased ~20% (0.15 → 0.18).

---

## 2026-04-07 — Spell Deflection, Bank, Mutation Tuning, Mob AI

### Spell Deflection & Stoic Resolve
- **Spell Deflection**: high-Willpower characters have a chance to
  deflect incoming spells entirely (Willpower-based opposed roll).
  Mobs that deflect show attacker-facing messages.
- **Stoic Resolve**: high-Willpower characters have a chance to
  resist taunts and rhetoric attacks entirely.
- Both are percentage-based avoidance layers checked before damage
  resolution.

### Thornwall Bank
- New **bank room** in Thornwall with a bank clerk NPC.
- **Unlimited storage** with monthly per-item fees. Forfeiture
  warnings sent via inbox when fees are due.
- Room-level `StorageCapacity` field replaces hardcoded 20-item
  limit. Bank room has uncapped storage.
- Updated bank and storage help files.

### Mutation Discovery Tuning
- Mutations now **prefer deepening** existing mutations (70/30
  coin flip) over discovering new ones.
- **Rarity uplift**: higher-level characters are more likely to
  discover rare mutations (weighted pool scales with avg level).
- New config knob: `MutationDeepenChance` (default 0.70).

### Reactive Tactical AI
- Mobs can now have **tactical AI** that reacts to combat events
  in real time. Configured via YAML with triggers, actions, and
  discipline settings.
- **4 presets**: aggressive_melee, defensive_caster, ambusher, tank.
  Mobs can also define custom tactics that merge with presets.
- **19 mobs configured** across warren tunnels, Ironwind Steppe
  caves, bandits, named NPCs, and Thornwall enemies.
- Ambusher mobs (stalkers, lurkers, skulkers) flee after engaging,
  then re-hide for another surprise strike cycle.
- Caster mobs (shamans, Sylara) self-buff, heal, and interrupt.
- Tank mobs (beetle queen, sentries, Velk) call for help and
  protect allies.

### Combat Targeting
- Hostile mobs now **prefer player targets** over companions when
  choosing who to attack. Companions only get targeted when no
  eligible players are in the room.
- Mid-fight retargeting also prefers players over companions,
  keeping aggro on the player consistently.

### Cleanup
- Removed dead skills: `cast`, `ranged-combat`, `first-aid`.
- Player feedback files now persist across container restarts.

### Bug Fixes
- Fixed non-crafter merchant shops not restocking inventory.
- Fixed tutorial items having missing gold values (broke shop
  pricing).
- Fixed missing space in `assess` corpse essence message.
- Fixed mobs that die in round 1 never receiving their AI signal
  (combat_start now emits before the player's attack resolves).
- Fixed per-round triggers (health_below, etc.) never firing —
  added combat_round signal emission every round tick.
- Fixed ambusher preset trying to hide mid-combat instead of
  after fleeing.
- Restored backed-up tutorial room scripts.

---

## 2026-04-06 (evening) — Foraging, Salvage, Progression Dashboard

### Foraging
- **Iron ingots** now forageable in caves (common) and mountains
  (uncommon). Previously only available from merchants.

### Salvage
- **Spoiled/declining potions** can now be salvaged for binding
  paste (1-2 depending on salvage skill). Previously spoiled
  potions were useless.

### Economy
- Enchanter Vael's binding paste restock increased from 5 to 15.

### Admin
- **Progression Dashboard**: new admin tab with system-level health
  metrics for the use-based progression system.
  - Skill health scores (expected curve deviation, stall detection,
    clustering)
  - Population distribution charts (skill tiers + stat values)
  - Discovery health (spell/recipe flags: too_hidden, too_easy)
  - Player overview with tier badges and activity totals
  - Auto-refreshes every 30 seconds

---

## 2026-04-06 — Enchanting Rework, Multi-Arm Equip, Aggro Fix

### Enchanting System Rework
- Enchanting now targets **equipped items by slot**, not inventory.
  Use `craft <recipe>` for auto-targeting or `craft <recipe> weapon#2`
  / `craft <recipe> 2.ring` for specific slots.
- **18 enchantments** covering all equipment slots (was 10).
  New: Chitin Brace (wrist), Rootbind (belt), Rootwalker (feet),
  Chrysalis Bond (ring), Spore Mantle (shoulders), Thornguard
  (shield), Venomgrip (gloves), Shadowweave (back).
- **Mitigation coverage**: physical (body/wrist/feet), magical
  (shoulders/back), conviction (neck/ring).
- **Lifesteal**: Hungering Touch now heals on hit instead of
  flat damage bonus.
- **Thornguard**: shield enchant that deals return damage.
- **Two-handed weapons** get doubled enchantment effects and
  doubled reserve costs.
- All enchantments rebalanced to 5 tiers with standard reserve
  curve (1%/2%/4%/6%/8%).
- Existing enchanted items are automatically migrated on login.
- Help files updated for all 18 recipes.

### Multi-Arm Equipment Rework
- Arms are now grouped into **pairs**: (1+2), (3+4), (5+6).
  Two-handed weapons occupy a full pair. One-handed weapons and
  shields fill individual slots.
- Arm 1 is weapon-only; arms 2-6 hold weapons or shields.
  Maximum turtle build: 1 weapon + 5 shields.
- Equip syntax: `equip sword arm#3`, `equip shield 2.arm`.
  Two-handed weapons must target odd-numbered arms (1, 3, 5).
- Odd extra-arm counts (1 or 3) create a half-pair that holds
  one-handed items only.
- **Defense scores fixed**: parry and block now use the best
  rating across all equipped weapons/shields, not just main hand.
- **Gearup** (`wear all`) now fills extra arm slots, best items
  first.
- Inventory hides the partner slot when a 2H weapon occupies
  the pair (no more "Offhand: -nothing-").

### Companion Naming
- Use `companion <name> name <nickname>` to give your companion
  a personal name. Displays as "Nickname the Type" (e.g.,
  "Fred the Spirit Wolf") in room text, combat, and vitals.
- Names must be unique — no duplicates across companions or
  player characters. New characters also can't take a name
  that belongs to an active companion.
- Creature-type words (skeleton, wraith, elemental, etc.)
  added to the banned names list.
- Companion disambiguation now supports `2.earth` / `earth#2`
  syntax for targeting specific companions of the same type.
- Nicknames persist across logout/login.

### Skill Progression Overhaul
- **Ceiling fix**: combat skills (weapon-combat, unarmed-combat,
  ranged-combat) could never advance past ~apprentice due to an
  asymptotic ceiling in the progression formula. Now all skills
  can reach soft cap regardless of their progression multiplier.
- **Per-weapon progression**: each weapon in a multi-arm setup
  independently trains weapon-combat skill. Extra arm weapons
  now contribute to skill growth, not just the main hand.
- **Defender progression**: dodging trains unarmed-combat + dex,
  parrying trains weapon-combat + dex + str, blocking trains
  weapon-combat + str. Defense type is tracked per-round.
- **Manifestation multiplier**: bumped from 0.3 to 0.5, matching
  spellcasting progression rate.

### Unarmed Combat Rework
- **Both hands attack**: every empty hand is a fist. Bare-handed
  fighters get 2 fist attacks. With extra arms mutation, up to 6.
- **Mixed setups**: sword in one hand + empty offhand? The free
  hand still punches. Works with all arm slots.
- **Fist/claws weapons**: train unarmed-combat skill, not weapon.
- **No parry**: unarmed-style fighters (fists, claws, bare hands)
  can only dodge. You can't deflect a blade with your knuckles.
- **Speed bonus**: unarmed attack speed increased (1.4 → 1.8) to
  compensate for dodge-only defense.
- **No dual-wield penalty**: natural weapons fight penalty-free.

### Defense Crit Rework
- **Parry crit → RIPOSTE**: free counter-attack at half weapon
  damage. Replaces the old disarm-on-parry mechanic.
- **Dodge crit → SWEEP**: automatic trip attempt that ignores
  the special move cooldown. Can knock the attacker prone.
- **Block crit → SHIELD SLAM**: automatic bash attempt that
  ignores the special move cooldown. Can knock them down.
- All three use distinctive cyan-bold messages to stand out in
  the combat scroll.
- **Disarm reworked**: disarmed weapons now go to inventory
  instead of dropping on the ground (grapple disarm still exists).

### Balance
- Assess command now has a 6-round cooldown.
- Companion corpses can no longer be re-raised.
- Healing potion durations halved.
- Equipped items now encumber 50% less.
- Shop restock rate tripled (6 hours → 2 hours).

### Bug Fixes
- **Aggro cleanup on mob flee**: players no longer get stuck
  "in combat" after all enemies flee the room.
- **Staff combat messages**: removed all "two-handed" and
  "both hands" references. Staves can be equipped one-handed
  in extra arm slots.
- **Extra arm attack messages**: weapons in extra arms now show
  their own name in combat text instead of the main hand weapon.
- **Surprise attack arms 5-6**: surprise strike now swings all
  equipped weapons, including extra arms 3 and 4.
- **Merchant kill exploit**: non-combatant NPCs (merchants, quest
  givers) no longer flee from pack scatter, can't enter the combat
  loop, and can't be provoked into fighting.
- **Companion auto-assist aggro**: player now properly engages when
  their companion is attacked. Previously a dummy aggro with no
  target was set, leaving the player stuck swinging at nothing.
- **Target switch on dead targets**: switching targets when your
  current target is dead is now free — no skill roll, no round
  penalty.
- **Fold-recall clears combat**: casting fold-recall now ends combat
  before teleporting. New `EndCombat()` scripting API for spells.
- **Wooden shield price**: fixed inflated auto-value (was ~430g from
  legacy DamageReduction formula, now 8g).
- **Chrysalis cores**: Apothecary Voss and Alchemist Yenna now buy
  chrysalis cores.

## 2026-04-04 — Living Economy, Gear Upgrades, Spell & PvP Fixes

### Bug Fixes (Round 2)
- **Tail slot no longer shows** when the character lacks the tail
  mutation. Was caused by EnableAll() resetting the disabled state
  before the tail check ran.
- **Companions no longer despawn** from the idle boredom timer.
  Wolf spirits and other charmed companions now persist properly,
  fixing missing vitals bars in the web client.
- **Mobs targeting your companions now show red** in the look
  command, same as mobs targeting you directly.
- **Duplicate companion vitals fixed** — same-name companions
  (e.g., two skeletons) now show separate health bars.
- **Gossip quality improved** — NPCs now use different phrasing
  for local vs. distant events, and each gossiper tracks recently
  mentioned events to avoid repetition.

### Living Economy
- Merchants now track finite stock and gold. Prices rise when
  stock runs low and drop when overstocked.
- Crafter NPCs restock materials periodically (with flavor
  text) and craft items to sell — prioritizing self-gear
  upgrades, then profitable crafts, then salvage.
- Merchants will buy craft materials matching their trade,
  potions (unless they craft that potion themselves), and gear
  that upgrades their own equipment — including paired slots
  like rings and wrists. Specialists won't buy materials from
  other professions.
- Shopkeepers are now non-combatant — they cannot be attacked,
  stolen from, or targeted by harmful spells.
- Bartering skill now affects buy and sell prices at shops.

### Under the Hood
- Mobs now advance stats and skills from basic attacks, special
  moves, and spellcasting — same progression system as players.
- Combat commands (bash, kick, trip, grapple, cast) now handle
  skill progression in the shared action layer rather than
  separately for players and mobs.
- Mob howl and player taunt now share the same underlying
  conviction-damage mechanics. (Also fixed howl not applying
  the skill-weight multiplier to rhetoric.)
- Bite and hamstring are now shared actions, ready for future
  player species-gated abilities.

### Bug Fixes
- **Area harm spells no longer damage the caster's companions.**
  This was caused by the spell resolution step overwriting the
  companion-exclusion filter from cast initiation.
- Single-target harm spells now prevent targeting your own
  companion ("You can't target your own companion with a
  harmful spell.").
- Charmed mob casters no longer hit their owner or the owner's
  other companions with area spells.
- Casting an area spell with no valid targets now gives feedback
  ("Your spell erupts outward but finds no targets.") instead
  of silently consuming conviction.
- PvP is now properly blocked across all combat entry points
  (attack, bash, kick, trip, grapple, taunt, shoot, spells).
- Fixed enchanting craft command parsing for hyphenated recipe
  names (e.g. "craft honed-edge knuckles" no longer fails).
- Shop listing now shows correct finite stock and dynamic
  prices instead of infinite legacy quantities.

### Cleanup
- Removed deprecated mob commands: roar, throw, backstab.
- Renamed mob `alchemy` command to `selljunk` (converts
  inventory to gold — not related to player alchemy).

---

## 2026-04-03 — Manifestation, Companions, Necromancy, Elementals, New Zones

### New Content
- New hidden areas have been added to the world. Sharp-eyed
  adventurers may discover passages others have overlooked.
- A reclusive figure lives off the beaten path. Not everything
  is as it appears — tread carefully.
- Lockpicks and disarm kits now available from certain merchants.
- Crafters can forge superior tools at high skill levels.
- Powerful caster equipment can be found by those who earn it.

### New Mechanics
- **Defuse command** — disarm traps on locks before picking.
  Requires a disarm kit. Higher tier kits improve success.
- **Flee rework** — flee is now an opposed roll (Dex+skullduggery
  vs Dex+unarmed-combat). Rogues are better at escaping.
  Can't flee while grappled. Prone halves flee chance.
- **Fist weapons** — new weapon subtype using unarmed-combat skill.

### Quality of Life
- `idea` is now an alias for `suggest`.
- `disarm` is now an alias for `defuse`.
- `lockpick` and `pick` are aliases for `picklock`.
- Companions prevent sneaking — dismiss before stealth.
- Companion corpses cannot be raised by necromancy.

### Bug Fixes
- AOE harm spells no longer damage the caster or their companions.

### New System: Manifestation Skill
A new charisma-based skill governing summoning, conjuring, charming,
and raising undead companions. Manifestation spells use Charisma
instead of Willpower for fold rate and discovery.

### New System: Unified Companions
Pets, summoned creatures, conjured elementals, charmed mobs, and
raised undead all share a unified companion system. Summoned,
conjured, and raised companions persist across restarts. Charmed
companions are temporary — they don't survive server restarts.
All companions show in the vitals panel, respond to autoassist,
and can be buffed with help spells.

- `companion` / `companions` — view companion vitals and stats
- `dismiss` — release a companion (warning: full betrayal)
- `assess` — study a corpse for necromantic potential
- `{pet_hp}`, `{pet_sp}`, `{pet_cp}` prompt tokens

### Necromancy (6 undead types)
Raise undead from corpses. Stronger corpses support more powerful
types. Power scales 50/50 from caster stats and corpse strength.
- Skeleton, Zombie, Wraith (caster), Spectre (conviction caster),
  Vampire (life drain bite), Flesh Golem (absorbs corpses)

### Conjure Elementals (5 types)
Conjure elemental companions from nothing. Very high conviction
cost — conjuring a magma elemental drains nearly your entire pool.
- Water (tank), Earth (bash), Air (evasive), Fire (return damage),
  Magma (bash + return damage, skill gate 60)

### Charm Spell
Opposed roll (Charisma+manifestation vs Willpower+statpool%) to
convert hostile mobs into companions. Harder against targets in
combat. Duration-based with diminishing re-rolls — your hold
gradually weakens until it breaks or you reassert control.

### New Combat Mechanics
- **Return damage** — fire/magma elementals reflect melee damage.
  Also available via equipment and buffs (battlerager armor, etc.)
- **Natural bash** — earth/magma elementals bash without shields
  ("crushing slam" instead of "shield bash")
- **Grapple immunity** — wraiths, spectres, air and fire elementals
  can't be grappled or grapple others
- **Vampire bite** — life drain special attack

### Aggro Rework
Centralized aggro state management fixes multiple companion combat
bugs. Players now properly retarget when companions kill their
target, when targets flee, and when new threats appear.

### Bug Fixes
- Enchanting target search broken by multi-word recipe names
- Conditions display showed total duration instead of remaining
- Infinite gold exploit — merchants pay from own gold pool
- Companion duplication on browser refresh
- Summon species corrections (were using rodent stats)
- Pack flee excludes companions
- Stale aggro from companion kills
- Web client vitals panel resizes with companion rows

### Balance
- Melee skill progression 0.20 → 0.15 (auto-attack now works)
- Spell damage scale 1.6 → 1.2 (progression provides natural scaling)
- Merchant stats buffed (85-150 statpool, 50-300g gold)
- Corpse decay 1 hour → 4 hours (for necromancy)

## 2026-04-02 — Command Unification + Bug Fixes

### Command Unification (feature/command-unification)
Major architectural rework unifying player and mob command systems
through shared core logic. Both sides now call the same underlying
actions for all major game commands.

**Shared Actor System:**
- Actor interface in `internal/actions/` abstracts over players
  and mobs. Shared actions operate on either actor type.
- Atomic transfer primitives (TransferItem, TransferGold) with
  rollback prevent item duplication and loss.
- Registry audit at startup warns about unintentional command gaps.

**Unified Commands:** say, emote, drop, remove, equip, get, give,
go, bash, kick, trip, grapple, shoot, attack, cast, sneak, craft.

**Combat Parity:**
- Kick now selects stomp/knee variants for mobs (position-aware).
- Trip uses tailsweep for mobs with tail mutation.
- Shared combat helpers (target resolution, cooldowns, analytics).

**Progression Parity:**
- Mobs now advance stats and skills from combat, casting, crafting.
- Player auto-attack melee progression was broken (never fired) —
  now works correctly.
- Caster mobs discover new spells as spellcasting skill increases.
- Mob sneak uses opposed rolls instead of auto-succeeding.

**Mob Crafting:**
- Mobs can now craft items via the shared craft system.
- Crafting completion fires skill progression for mobs.

### Fixes
- **Hidden mob perma-stealth:** Root cause found — permabuff system
  re-added Hidden buff after every Validate(). Fixed with
  RemovePermaBuff + proper combat loop integration.
- **Hidden mob surprise attacks:** Mobs properly get [SURPRISE ATTACK]
  when ambushing from stealth. Hidden buff clears after first strike.
- **Duplicate "prepares to fight" message:** Suppressed when mob
  re-attacks the same target.
- **Sneak in combat:** Blocked for both players and mobs — sneaking
  mid-combat doesn't make sense and caused perma-hidden bug.
- **Conditions duration display:** Was showing total duration instead
  of remaining rounds (swapped return values).
- **Infinite gold exploit:** Merchants now pay from their own gold
  pool when buying items. Refuse if they can't afford it.
- **Defense hint:** Now points to `help defense` instead of a
  nonexistent `defense` command.

### Balance
- Melee skill progression reduced from 0.20 to 0.15 — the bump
  was compensating for broken auto-attack progression (now fixed).
- Spell damage scale reduced 25% (1.6 → 1.2) — progression now
  provides natural scaling.
- Merchants buffed: higher stats (85-150 statpool), gold reserves
  (50-300g), Siv armed with a dagger.

## 2026-04-02 — Bug Fixes & Polish

### Fixes
- **Enchantment idle bug:** Chrysalis enchantments (honed edge, etc.)
  no longer progress while idle. They now only tick during combat.
- **Web client side panels:** Map, Communications, and Vitals
  windows now resize and reposition dynamically when the browser
  window is resized (both horizontally and vertically). Vitals
  no longer gets cut off on smaller screens like laptops.
- **Small screen support:** Side panels are hidden entirely on
  very small screens (phones/small tablets under 768px) to keep
  the terminal usable.

### Content
- **help equipment:** New help file covering all equipment slots,
  back slot trade-off (cloaks vs backpacks), belt slot trade-off
  (belts vs bandoliers), and the component bag system.

## 2026-04-01 — Quest Engine

### New System: Quest Engine
A complete YAML-driven quest engine that replaces JavaScript
scripts for all quest logic. Quests are now defined entirely in
data files with declarative triggers and conditions.

- **9 event types:** room_enter, room_interact, item_gain,
  item_give, mob_death, skill_use, command, dialogue,
  quest_granted (for chaining steps automatically).
- **Trigger actions:** grant quest tokens, give/consume items,
  send text, NPC dialogue sequences, teleport, spawn mobs,
  teach spells, apply buffs, set quest flags.
- **Quest flags:** branching quests track which path the player
  chose. Flag-gated dialogue shows different content per path.
  Undeclared flags panic at startup to catch typos early.
- **hint command:** type `hint` for guidance on your current
  quest step. Hints give explicit directions and next actions.
- **Verbose quest debugging:** admins can enable per-player
  quest debug logging with `questdebug <player>`.

### All Quests Ported (1-16)
Every quest in the game now runs through the quest engine:
- **Quest 1 (Sanctum Trials)** — full tutorial with ceremony
  sequences, mutation grant, shopping/equip/combat/magic steps.
- **Quest 2 (Warren Compact)** — salve delivery to tunnel shaman.
  Mobs become peaceful after quest completion.
- **Quest 3 (Scholar's Collection)** — dual-item delivery with
  flag tracking for partial completion.
- **Quests 4-7** — item delivery and combat quests across
  Dustwalk Road, Watchers Crossing, and Thornwall Outskirts.
- **Quest 8 (Missing Person)** — investigation quest in Thornwall.
- **Quest 9 (Tithe Audit)** — ledger delivery to Priest Olen.
- **Quest 10 (Drowning Post)** — protection notice to Velk.
- **Quest 11 (Windwarden's Dilemma)** — opposed branching quest
  with quest flags. Choose Sylara or Rhett; the other dismisses
  you. Flag-gated followup quests (12 or 13).
- **Quests 12-13** — path-exclusive followup quests (Covenant
  vs Extraction) gated by Q11 branch flag.
- **Quest 14 (The Undertow)** — 6-step dungeon crawl with cellar
  gate, tally stick discovery, strongbox key/open interaction,
  and bribe ledger delivery. Full room_interact support.
- **Quest 15 (Peddler's Freight)** — crate delivery with combat
  or diplomacy paths.
- **Quest 16 (Herbalist's Shortage)** — dual-path herb delivery
  with bypass for players who explore first.
- **Quest 17 (Empty Cottage)** — converted to lore discovery
  (no longer a tracked quest).

### Bug Fixes
- **Quest re-grant prevention** — fixed 18 dialogue nodes across
  15 files where completed quests could be re-offered. Added
  runtime validation that warns if a quest-granting node is
  missing its end-token exclusion.
- **Quest hints improved** — all quests now give explicit
  step-by-step directions with cardinal directions and counts.
- **Dialogue hints** now display as narrator text, not NPC speech.
- **Branching quest dismissals** — wrong-path players get clear
  rejection instead of confusing keyword matches.
- **Shadow Realm combat trap** — fixed a bug where players could
  get stuck in the Shadow Realm with stale combat state after
  the warden-bandit alliance fight.
- **False skill-up messages** — skill progression messages no
  longer fire on critical failures or first mob kills when no
  real skill gain occurred.
- **Alchemy recipe cleanup** — removed legacy duplicate starter
  recipes that confused new players in the tutorial. Tutorial
  now uses healing salve instead of removed healing poultice.

### Balance
- **Moon phase effects doubled** — full/new moon bonuses and
  penalties are now more noticeable.

### Migration
- Players on removed quest steps are automatically reset to
  "start" on server startup. Quest 17 progress removed entirely.
- Quest 11 branch flags inferred from Q12/Q13 progress for
  existing players.
- Legacy healing poultice and stamina draught auto-converted to
  new alchemy equivalents.

---

## 2026-03-31 — Salvage System

### New Feature: Salvage
Break down crafted items and tagged salvageable items to recover
crafting materials with the new `salvage` command.

- **New skill: Salvage** — standalone Perception-based skill in
  the "scavenger" profession alongside Search. Recovery chance
  scales with skill via a sqrt curve (15% at novice, up to 85%
  at master). Each ingredient is rolled independently.
- **Recipe reverse-lookup** — any item produced by a crafting
  recipe can be salvaged at the matching station for free.
- **Salvage kit** — sold by Fence Dealer Siv in Thornwall's back
  alleys for 1g. Allows salvaging anywhere without a station.
  Consumed on each use.
- **Tagged items** — non-crafted items can be marked salvageable
  with `salvage_returns` on their item spec. Always requires a
  salvage kit.
- **Multi-round activity** — salvage duration scales with
  ingredient value (1-5 rounds). Interrupted by combat.
- **Item always consumed** — even if no materials are recovered,
  the item is destroyed.
- Type `help salvage` in-game for full details.

---

## 2026-03-31 — Bug Fixes & QoL

### Features
- **ASCII Charset Mode:** `set charset` toggles between UTF-8 and ASCII
  display. Legacy clients (zMUD etc.) that show garbled box-drawing
  characters can switch to clean ASCII mode. Persists across sessions.
- **Mutation Help Files:** All 40 mutations now have individual help
  pages (`help healing-gel`, `help extra-arms`, etc.).

### Bug Fixes
- **Skill progression messages fixed:** Critical hit "technique improves"
  messages were firing on every crit regardless of whether the skill
  actually advanced. Now only shows when a real gain occurs.
- **Harm spell exploit closed:** Casting harm spells with no target no
  longer grants free spellcasting progression.
- **Harmful buffs trigger aggro:** Spells like Nerve Disruption that
  apply debuffs now properly start combat, matching damage/dot/knockdown
  behavior.
- **Tutorial directions corrected:** Directions to the Training Yard
  now correctly say north-then-east (was "northeast").
- **Removed misleading combat-end message:** The generic "rage subsides"
  text no longer appears after every kill.

### Balance
- **Combat skill progression bumped:** Weapon-combat and unarmed-combat
  progression rate increased from 0.12 to 0.20. These skills were
  advancing too slowly relative to other skills.

---

## 2026-03-30 — Alchemy Rework (Phase 1-3)

### Alchemy Overhaul
- **Potion Aging:** Potions now age through five phases (Fresh →
  Fermented → Peak → Declining → Spoiled). Peak potions are 30% more
  potent. Spoiled potions cause nausea and triple toxicity.
- **Bottle Tiers:** Four bottle types control aging speed. Clay flask
  (ages 3x faster, cheap), glass vial (baseline), sealed phial (half
  speed, jewelcrafting), crystalline decanter (quarter speed, advanced
  jewelcrafting).
- **Toxicity System:** Every potion adds toxicity. Exceed your limit
  and your body rejects the brew. High toxicity causes stat penalties.
  Toxicity decays naturally over time.
- **Craft Skill Matters:** Higher alchemy skill at brew time means
  stronger, longer-lasting potions that age slower.
- **Skill-Based Detection:** Examining potions reveals aging info
  based on your alchemy skill. Novices can't tell fresh from spoiled.

### New Potions (21 recipes)
- **Pool Regen (7):** Healing salve, stamina tonic, conviction
  draught, warrior's brew, preacher's tincture, windrunner draught,
  elixir of renewal.
- **Combat/Utility (10):** Ironhide brew, mindshield elixir,
  veilguard tonic, stone stomach, cat's eye draught, swiftfoot
  essence, berserker elixir, silver tongue oil, battle trance,
  purging draught.
- **Progression (4):** Essence of growth, savant's infusion, mutagen
  brew, chrysalis catalyst. These accelerate character development
  but reserve portions of your resource pools.

### Potion Bandolier
- New belt-slot item that auto-routes potions and reduces their
  weight. Two tiers: leather (6 slots, 30% weight reduction) and
  reinforced (12 slots, 40% weight reduction). Craft via tailoring.

### New Materials
- **Moonpetal** — rare forage, night only.
- **Veilbloom Petal** — very rare forage on the steppe.
- **Serpent Venom Sac** — drops from river lurkers and blind stalkers.
- **Ironbark Shaving** — uncommon forest forage.
- Clay flask sold by Apothecary Voss.

### Consumption Rework
- Drinking a potion now checks toxicity before consuming. If you'd
  exceed your maximum, the potion is rejected.
- Aging phase affects potency: peak potions last 30% longer, declining
  potions are weaker, spoiled potions cause nausea + 3x toxicity.
- Craft skill at brew time scales potion duration (skill 20 = +20%).
- Bottle type is stamped on the potion at craft time, determining its
  aging speed for its entire lifecycle.

### Maker's Mark
- Skilled crafters (skill 30+) now leave their name on items they
  craft. Examine a crafted weapon, potion, or piece of armor to see
  "Made by {name}." Purely cosmetic — does not affect stacking.

### QoL
- Spoiled potions display as "(turned)" in inventory for alchemists.
- Potions in bandolier show in a dedicated "Potions:" section.
- Drink command pulls from bandolier first (oldest potion).
- Five new alchemy-related gameplay tips in the hints rotation.
- Old potions and recipe knowledge auto-migrate on login.

### Bug Fixes
- **Velk bribe ledger quest:** Fixed quest getting stuck at 83%.
  The dialogue was still asking for the ledger after it had been
  given. Players with the stuck quest should now be able to complete
  it by talking to Velk.
- **Sylara spirit fetish spell:** Fixed "You need a spirit fetish"
  error when the fetish was in the component bag. Spirit fetishes
  now stay in the regular backpack where the spell can find them.
- **Text wrapping:** Say, shout, whisper, emote, and party chat
  now wrap the full message (including speaker name) at 80 chars
  instead of wrapping text alone at 65 then prepending the name.
- **zMUD compatibility:** Fixed display flashing for legacy MUD
  clients that don't support GMCP. The server no longer sends GMCP
  data to clients that haven't completed the GMCP handshake.
- **Description wrapping:** Player and NPC descriptions no longer
  double-wrap with orphaned words. Descriptions are stored raw and
  wrapped once at display time. Existing player descriptions are
  auto-migrated on login.
- **Floor item stacking:** Identical items on the ground now display
  with (xN) count instead of separate lines.
- **Vendor room clutter:** Removed crafting materials baked into
  7 vendor/crafter room templates that respawned every restart.
- **Drop all:** No longer drops your gold. Use "drop N gold" to
  drop gold explicitly.

---

## 2026-03-30 — Mutations, Balance, Documentation & QoL

### New Mutations
- **Chameleon Skin** (rarity 7) — +30 stealth bonus, +10 dodge.
  Costs charisma and natural armor. Conflicts with thick-hide.
- **Tail** (rarity 8) — Adds Tail equipment slot, disables Legs
  slot. Reskins trip to tailsweep (better damage and knockdown).
  Three tail attachments: weighted cap, spiked band, bladed sheath.

### Stealth Improvements
- Characters emitting light have their sneak score halved.
- Moving while sneaking costs 50% more stamina.
- Hidden mobs now get surprise attack on their first strike.

### Spell Duration Scaling
- All spell durations now scale with fold count, spellcasting
  skill, and willpower via universal formula.
- Higher-fold spells naturally last longer. Investing in willpower
  and spellcasting extends everything.

### PowerScore Rework
- Skills are now a major factor (sqrt of total ranks × 25).
- All three resource pools count (HP + SP×0.5 + CP×0.5).
- Mutations contribute 20 points per level.
- KD ratio replaces raw kill count (kills/deaths × 10, cap 50).
- Magic/conviction offense normalized against physical.
- Defense weighted 3× more heavily.

### Defense Balance
- Dodge effectiveness 0.97→0.95, Parry 1.0→0.97, Block 1.02→1.05.
- New clinch defense penalties: dodge 0.80, parry 0.83, block 0.85.
- New grounded defense penalties: dodge 0.75, parry 0.77, block 0.80.
- Prone dodge/parry penalties 0.95→0.93.

### New Commands
- **afk** — Manual AFK toggle with optional message. Shows (AFK)
  next to your name in the room. Auto-clears on any input.
- **setdesc** — Set your own character description.

### Crafting
- Craft list now shows recipe completion tier per skill and overall.
- Subcomponent recipe thresholds lowered (steel ingot, chain links,
  chrysalis setting).

### Documentation
- Help files for all 39 spells, 47 recipes, and 4 combat skills.
- Completeness tests ensure new content always has help files.
- 15 new gameplay tips added to the hint rotation.

---

## 2026-03-29 — Combat, Stealth & Spell Balance

### Kick Rework
- **Kick** is now a powerful unarmed strike (damage doubled from 0.40 to
  0.80). Three automatic variants based on combat position:
  - **Kick** (standing target): 35% knockdown chance.
  - **Stomp** (prone target): 1.20x damage, bypasses half armor,
    extends prone duration. The payoff for knocking someone down.
  - **Knee** (grapple, in control): 1.0x damage, works in clinch/ground.
- `stomp` and `knee` are command aliases for `kick`.

### Opening Fights with Special Moves
- Kick, bash, trip, grapple, and taunt can now initiate combat by
  naming a target (e.g., `kick bandit`). No longer requires attacking
  first.

### Stealth System
- Players now detect hidden mobs when entering a room via opposed
  Perception+Search vs Dex+Skullduggery roll.
- Rogue NPCs added: Blind Stalker, Pale Lurker, Warren Scout, Tunnel
  Lookout, and Goblin Scout spawn hidden and ambush on entry.
- Thornwall Highwayman, Smuggler Runner, and Torvan Cresk use tactical
  combat stealth.

### Caster NPCs
- Elder Saris, Priest Olen, Geomancer Rhett, and Windwarden Sylara now
  have spellbooks and cast buff spells while idle. Attack them and
  they fight back with appropriate magic.

### Buff/Ward Spell Rework
- **Shield spells** now scale by spell magnitude. Conviction Ward is
  75% strength (quick/cheap). Chrysalis Cocoon is 125% strength and
  grants magical/conviction mitigation. Both last 2.5x longer.
- **Iron Will** now provides magical and conviction damage mitigation
  alongside the willpower boost. Lasts 50 rounds (was 8). Costs more.
- **Chrysalis Haste** costs more but lasts twice as long.
- **Vital Surge** regen lasts 3x longer.
- **Empathic Shroud** no longer cancels on entering combat.
- **Veil Sight** now grants see-hidden (was incorrectly giving light).
- **Skill Attunement** and **Mutation Catalyst** last 10x longer but
  cost 3x more conviction.
- All debuffs (Nerve Disruption, Mind Fog, Sensory Overload, Psychic
  Anchor) last 50% longer.

### New Commands
- **reply** — Whisper back to the last person who whispered to you.
- **rep/report** — Broadcast your vital bars to the room, party, or
  a specific player.
- **setdesc** — Set your own character description.

### Stat Progression
- Taking a critical hit now triggers stat progression: physical crits
  improve vitality, magical crits improve willpower, rhetoric crits
  improve charisma.

### Balance
- Taunt damage +50% (RhetoricDamageScale 2.0 → 3.0).
- Spell damage -14% (SpellDamageScale 1.85 → 1.6).
- Subcomponent recipe discovery thresholds lowered: Steel Ingot 10→4,
  Chain Links 15→7, Chrysalis Setting 15→7.

### Other
- Spells list now sorted by fold count (simplest first).
- Leaderboard expanded from 10 to 20 entries.
- 4 new tailoring recipes: Leather Backpack, Reinforced Travel Pack,
  Artisan's Satchel, Master's Component Case.
- Component bag capacities increased (20/40/75).
- Apothecary Voss now sells alchemy ingredients.

---

## 2026-03-29 — Equipment Slot Expansion + Component Bags

### New Equipment Slots
- **Wrist** (x2) — Bracelets and bracers now have their own slots
  instead of using the ring slot. Existing bracelet items have been
  updated.
- **Shoulders** — Pauldrons, mantles, and shoulder armor.
- **Back** — Cloaks for stats, or backpacks that reduce the effective
  weight of your carried items. Choose wisely.
- **Second Ring** — Two ring slots instead of one.
- **Component Bag** — A dedicated bag for crafting materials. Materials
  auto-sort into it on pickup. Use `sort` to migrate existing
  materials. Buy a starter pouch from Weaver Maren in Thornwall.

### Extra Arms Mutation — Levels 3-4
- The Extra Arms mutation can now reach levels 3 and 4, granting up
  to four additional arms (six total weapon slots).
- Each extra arm comes with an extra wrist slot for bracelets.
- Higher levels impose steeper charisma penalties and aggro, with
  diminishing dexterity returns.
- Combat hit penalties scale: +20 per arm beyond your offhand.

### Component Bag System
- Crafting materials with the `is_component` flag auto-route to your
  component bag on pickup and purchase.
- The `sort` command moves existing materials from your backpack into
  the bag.
- Crafting consumes from the component bag first, then your backpack.
- Component bags reduce the effective weight of their contents.

### Bug Fixes
- Extra arm weapons now correctly count toward carried weight.
- Bracelet items correctly equip to wrist slots instead of ring.
- Cloaks moved from neck slot to back slot (automatic migration
  on login for existing characters).

---

## 2026-03-29 — Combat, Spell & Crafting Fixes

### Spell Fixes
- **Sparks** now correctly hits all enemies in the room (was only
  hitting one target despite being an area spell).
- **Mend All** now actually heals (was missing effect type data).
- **Hemorrhagic Wave** rebalanced: folds 10→20, magnitude 300→225.
  Still powerful AoE but no longer trivializes encounters.
- **Healing spells can now target downed players**, enabling
  revive-style healing like Chrysalis Aid. Harm spells skip
  downed players.
- **Self-targeting blocked** for harm spells — Conviction Spike
  and Nerve Disruption can no longer be cast on yourself.
- **Player-targeted harm spells** now deal damage and trigger
  reciprocal aggro (previously did nothing).
- Pet damage messages no longer duplicate in same-room combat.

### Crafting Fixes
- **Skill level-up messages** no longer repeat on every successful
  craft. The "Your X skill reaches Y!" message only appears when
  your skill tier actually increases.
- **Recipe discovery** now filters by the skill you're currently
  crafting. No more learning enchanting recipes while blacksmithing.
- **Casting and sneaking blocked while crafting.** Moving to another
  room cancels the active craft.

### Other Fixes
- **Title command** no longer shows raw numbers. Mutation load and
  skill progress use descriptive labels (fledgling/seasoned/etc).
- **Apothecary Voss** now sells alchemy ingredients instead of
  enchanting binding paste.

---

## 2026-03-29 — Critical Fixes + Inventory Rework

### Critical Bug Fixes
- **Death loop fix**: Players can no longer get permanently stuck
  in the Shadow Realm with stale combat state. Root cause fixed
  (mobs could re-aggro dead players), plus safety net and escape
  hatch so the portal always works.
- **Spell scripts now work for players**: Fold Anchor, Chrysalis
  Aid, Summon Steppe Spirit, and other script-based spells were
  silently broken — onMagic/onCast/onWait callbacks never fired
  for player casts. All three hooks are now wired into the cast
  pipeline.
- **Fold Anchor split**: Now two spells — `fold-anchor` (set) and
  `fold-recall` (teleport back). Players who knew fold-anchor
  automatically receive fold-recall on login.
- **Quest spell rewards**: Quests can now teach spells on
  completion. The Warden's Covenant (quest 12) now properly
  grants Summon Steppe Spirit.
- **Fetish gating**: Windwarden Sylara no longer gives unlimited
  spirit fetishes. If you already have one, she refuses.

### Inventory Rework
- **Diku-style disambiguation**: Use `3.dagger` or `dagger#3` to
  target a specific item when you have duplicates. Use `all.item`
  with get/drop to affect all matching items (e.g., `drop all.potion`).
- **Inventory stacking**: Identical items now group together with a
  count, e.g., `iron ingot (x5)`. Items with different enchantments
  remain separate.
- **Equipped item targeting**: `look` and `identify` now search
  your backpack and equipment as a single pool. You can examine a
  wielded weapon without unequipping it — use `look 2.dagger` to
  reach the equipped one when a duplicate is in your pack.
- **Encumbrance display**: Carrying capacity has been rebalanced.
  The inventory command now shows a colored encumbrance tier
  (light / moderate / heavy / overburdened / crushed) instead of
  raw weight numbers. Add `{enc}` to your prompt to track it at
  a glance (`help set prompt`).
- **Multi-buy**: `buy 5 iron ingot` purchases multiple copies in
  one command. Stops early if you run out of gold or can't carry
  any more.
- **Enchanting targeting**: `craft <recipe> <item-name>` lets you
  choose which item to enchant. Works on equipped items too. Shows
  a numbered list when multiple targets match.
- **Look direction fix**: `look n` no longer matches inventory
  items when no north exit exists.

### Balance
- Carry capacity reduced ~78% (now Strength × 0.65, configurable).
  Being overweight costs more stamina to move and reduces combat
  swings.

## 2026-03-18 — Skullduggery Skill + Tutorial Fix

### New Skill: Skullduggery
The old `stealth` skill has been consolidated into **skullduggery**,
a full rogue toolkit with seven sub-commands:

- **sneak** — hide using opposed rolls (Dex+skill vs observers'
  Perception+search). Empty rooms auto-succeed. Party excluded.
- **steal** — take gold/items from NPCs or containers. Being hidden
  improves your chances. Replaces the old pickpocket command.
- **plant** — slip items onto NPCs or into containers unnoticed.
- **shadow** — tail a target between rooms while hidden (rank 2+).
  Room-entry detection checks on each transition.
- **surprise attack** — automatic extra crit hit when attacking from
  stealth. Swings all weapons with stacking hit penalties. Party
  auto-assist triggers coordinated ambushes.
- **picklock** — existing minigame, now gated behind skullduggery
  rank 1.
- **defuse** — trap disabling stub for future content (rank 3+).

### Stealth Detection Rework
- Hidden players are now rolled against when entering rooms
- New arrivals roll to spot hidden occupants in the room
- Party members excluded from all detection checks

### Stealth & Movement Improvements
- Hidden players skip room onEnter scripts (NPCs no longer greet
  you when they can't see you)
- Sneak buff now applies immediately (no tick delay before moving)
- Per-weapon surprise attack messaging shows each weapon's hit
  and damage individually

### Quality of Life
- MOTD now displays in a visible bordered box on login
- Skill-gated commands show "You aren't advanced enough at
  skullduggery for that" instead of "command not found"
- Updated help files for steal and plant with clear syntax and
  examples
- Added missing alchemy_bench station to Apothecary Lane (room 471)
- Added hidden buff (ID 9) to dogmud world buffs (was missing)

### Bug Fixes
- Tutorial casting step now accepts spell ID shortcuts and aliases
  (e.g., `conviction-spike echo` works, not just
  `cast conviction-spike echo`)
- Existing characters auto-migrate stealth skill to skullduggery
  on login

## 2026-03-17 — Bug Fixes, Hidden Object Discovery, Identify Spell

### Legacy Stat Scaling Fixes
- Map command perception thresholds rescaled for 100-baseline
  stats (was 25/50/75, now 110/135/175). New characters start
  at tier 1 zoom instead of getting max zoom immediately.

### New Spell: Identify
The old `inspect` command has been removed and replaced with
the **Identify** spell (Mental school).

- Cast `identify <item>` to reveal an item's properties using
  descriptive language (no raw numbers shown to players)
- Works on items in your backpack or currently equipped
- Costs 20 conviction, 3 folds to cast, 30-round cooldown
- Merchants still offer `appraise` as a non-magical alternative

### Appraise Rework
- Appraise now shows full item details (previously capped at
  tier 3). All output uses descriptive language instead of raw
  numbers.

## 2026-03-17 — Bug Fix Day + Hidden Object Discovery System

### Bug Fixes (9 issues from playtesting)
- Conviction regen bumped to 2% per tick (matches health/stamina)
- Removed legacy tame-on-kill messages (taming now uses spells)
- Fixed disarm crit triggering on unarmed/disabled-slot targets
- Fixed misspelled commands showing "can't do that in combat"
  instead of "command not recognized"
- Fixed 2H weapon + extra arms exploit (extra arm slots now
  cleared when equipping a two-handed weapon)
- Fixed fold-anchor recall failing due to type mismatch
- Fixed gossip system — NPCs now report mob kills and player
  mutations (event buffer was starving for events)
- Fixed bleedout test to match current rate (2 per tick)
- Added `{attack}` token for defense messages (resolves to
  "strike" when attacker is unarmed)

### New Feature: Hidden Object Discovery
Rooms can now contain hidden nouns and hidden containers that
players must actively discover using the search command.

- **Hidden nouns** — invisible until found via search. Once
  discovered, they appear in the room description and respond
  to `look <noun>` permanently for that character.
- **Hidden containers** — function like normal containers but
  are invisible until discovered. Locks still apply after
  discovery.
- Discoveries persist permanently per-character.

### Skill Consolidation: Search
The tracking and foraging skills have been merged into a single
**Search** skill governed by Perception.

- `search`, `track`, and `forage` all progress the Search skill
- All three commands now use gaussian dice rolls (Perception +
  Search skill bonus) instead of hard stat thresholds
- Forage difficulty varies by biome (farmland is easiest,
  cliffs are hardest)
- Existing players: Search rank = max(tracking, foraging).
  No progression is lost.

### Balance
- Extra-arms mutation restricted to species with arm slots
  (no more wolves with extra arms)
- Search skill progression only fires when there's something
  undiscovered to roll against (prevents AFK botting)

## 2026-03-14 — Zone Expansion, Spell Merge, Coordinate System

### New Zone: Marches Spur Road
A new 15-room zone connecting Watchers Crossing south to the Ashwick
Crossroads — the first road into the wider Windward Marches.

- **15 rooms** from scrubland through farmland to a waypoint inn and
  crossroads junction
- **The Broken Yoke Inn** — social hub with gossiper NPCs relaying
  world events
- **Peddler Malk** — road merchant and quest giver at a camp along
  the spur
- **Quest: The Peddler's Overdue Freight** — find a stolen freight
  crate. Solve it through combat (clear the bandit barn) or diplomacy
  (negotiate a toll with the bandit leader). Multiple breadcrumbs and
  elephant-path coverage.
- **Bandit Leader encounter** — non-hostile with a 5-round dialogue
  window before she attacks. Talk fast or fight.
- **Wildlife**: scrub coyotes, feral hogs, field sparrows, farm cats

### New Zone: Ashwick
Maren's home hamlet from the novel, 20 rooms east of the Ashwick
Crossroads. A quiet farming village with secrets beneath the surface.

- **20 rooms** — hamlet proper (central green, chapel, farmstead,
  ritual circle, well, Delia's cottage) plus forest outskirts
  leading into deep woods
- **Delia the herbalist** — quest giver and alchemy crafting station
- **Deacon Ferris** — lore NPC with quest-gated deeper dialogue
- **The Forager (Sev)** — a hollow person hiding in the woods,
  mirroring the novel's themes of identity and concealment
- **Quest: The Herbalist's Shortage** — someone is harvesting
  Delia's herbs. Negotiate with the forager or find an alternate
  source in a hidden Chrysalis-touched grove.
- **Quest: The Empty Cottage** — explore Maren's abandoned family
  home. Push a loose hearthstone to find a hidden letter mentioning
  "the hill" and "Voss in New Plymouth." Study a recipe page from
  the bedside table to advance your foraging skill.
- **Novel breadcrumbs** throughout — scorch mark on the ritual
  circle, inner orbit symbol at the well and chapel, the cottage's
  empty shelves and cold hearth. Layered discovery rewards
  attentive players without frontloading spoilers.
- **Wildlife**: timber wolves, briar hawks, forest foxes, village
  chickens, a cottage mouse

### Spell Changes
- **Fold Anchor + Fold Recall merged** into a single toggle spell.
  Cast once to set an anchor, cast again from elsewhere to teleport
  back. No more needing to learn two spells for one mechanic. Existing
  players with Fold Anchor gain recall automatically.
- New dedicated help template for Fold Anchor explaining both modes.

### Cartesian Coordinate System
- All 224 existing rooms now have hidden `coord` fields (x, y, z)
  for spatial validation
- Full coordinate map at `docs/coordinate_map.md`
- **3 geometry overlaps fixed** in Sanctum Basin and Ironwind Steppe
  where rooms occupied the same coordinate
- Cartesian consistency rules added to zone expansion standards

### Bug Fixes
- Fixed steppe rooms 3032/3082: replaced JS quest item scripts with
  native container-based spawns
- Removed extra mob spawn from goblin camp room 3070
- Deleted stale instance saves that were overriding template edits

### Infrastructure
- Zone expansion plan (ZONE_EXPANSION.md): 22 zones, ~600 rooms
  planned across the Windward Marches
- Geography aligned to novel canon (What the Moons Keep)
- AI player default host updated to dogmud.org

---

## 2026-03-05 — Ironwind Steppe Rebuild, Quests & Ecosystem AI

### Ironwind Steppe Zone Rebuild
The entire Ironwind Steppe zone was rebuilt from scratch on a clean
cardinal grid with proper reciprocal exits throughout.

- **Rebuilt entry area** (rooms 3000-3009) on a clean cardinal grid
- **Sagebrush Flats** expansion (3010-3015, 3018) with ambient wildlife
- **Northern wolf/hawk territory** (3019-3023) with predator encounters
- **Hollow and boar/viper area** (3024-3028) with burrowing wildlife
- **Ironwind Ridge column** and northern steppe edge (3029-3033)
- **Upper ridge** — nesting ledge to summit (3034-3038)
- **East ridge descent** — alcove to overlook (3039-3043)
- **Dry creek system** and ridge descent (3044-3048)
- **Creek basin depths** — undercut bank to boar wallows (3049-3053)
- **Lower creek basin** — junction to basin south end (3054-3058)
- **Basalt coulee system** east of creek basin (3059-3063)
- **Deep coulee goblin territory** (3064-3068)
- **Goblin camp interior** and coulee south exit (3069-3073)
- **Wolf Run** and eastern coulee edge (3074-3078)
- **Deep Wolf Run** and wolf ravine east column (3079-3083)
- **Lower wolf territory** (3084-3088)
- **Boar wallow column** and eastern grassland (3089-3093)
- **Mudflat/boar territory** and drinking pool (3094-3098)
- **Cave system** entrance through deepest chambers (3099-3114)

### Quests
- **Quest 12 audit** — Sylara now grants quest start directly,
  removed unnecessary Kael checkpoint that could brick progression
- **Quest 14: The Undertow** — new smuggler tunnel quest beneath
  the Drowning Post tavern in Thornwall City

### Ecosystem AI
- **Species-based pack behavior** — mobs now ally and flee based on
  shared species (SpeciesId) instead of broad group tags. Wolves pack
  with wolves, not with squirrels that happen to share a group tag.
- **Predator-prey combat** — `HatesMob()` rewritten to use species
  names. Added natural predator-prey hatred across the ecosystem:
  - Canines (wolves, foxes, dogs) hunt rodents and boars
  - Raptors (hawks, eagles) hunt rodents and serpents
  - Felines hunt rodents and insects
  - Serpents hunt rodents and insects
  - Arachnids (spiders, scorpions) hunt insects
  - Boars defensively attack canines
  - Trolls attack most wildlife species

### Balance
- Bumped player conviction regen from 1% to 1.5% per tick

### Bug Fixes
- Fixed broken ANSI tag in Old Citadel Plaza board noun
- Fixed scrubland dog species (was reptile, now canine)

---

## 2026-03-04 — Quest Fixes, Balance Tuning & Zone Repairs

### Quest Fixes
- **Velk/Elara quest** — made dialogue discoverable and unbrickable
- **Harn/Pell delivery quest (Quest 6)** — unbricked progression
- **Removed `requires` + `expiryPeriod` quest brick** from all
  dialogue files across the game. These combinations could silently
  brick quests when player memory expired.

### Balance Tuning
- Reduced `GlobalDamageMultiplier` from 1.75 to 1.25 for less swingy
  combat
- Potion buff improvements and helpfile additions
- Temple regen and hint system improvements
- Faster bleedout timer for downed players

### Zone Fixes
- Resolved 74 broken reciprocal exits across the Ironwind Steppe zone
- Fixed spatial inconsistency in Watchers Crossing river lurker loop
- Disconnected Ironwind Steppe from Thornwall temporarily until zone
  rebuild was complete

### Features
- **Player PvE death gossip** — tavern gossip system now broadcasts
  player deaths to the gossip channel (global, not just local)
- **setmotd admin command** — admins can now set the message of the
  day in-game

### Bug Fixes
- Fixed `FindRecipeByName` to prefer exact match, preventing wrong
  recipe selection when names overlap
- Removed Area field from status template to prevent zone name
  misalignment
- Aligned status template columns with consistent 12+13 char widths
- Added missing admin commands to help list with helpfiles

---

## 2026-03-03 — Launch Day Fixes

### Major Fixes
- **ANSI tag crash fix** — prevented nested ANSI tags in noun
  highlighting that caused client crashes. Root cause fix in
  `roomdetails.go` to skip nouns already inside ANSI tags.
- **Tinymap panic fix** — used `VisibleWidth()` instead of `len()`
  for tinymap padding, preventing panics from ANSI escape sequences
  in map rendering.
- **Instance save override fix** — added `instance:"skip"` tag to
  structural room fields (exits, nouns, signs) so instance saves
  can no longer silently override template data for these fields.

### Quest & Item Fixes
- Scholar quest now accepts totem and spore sac in either order
- Added `givesItem` field to dialogue engine for NPC item handoffs
- Fixed Watchers Crossing quest items using new `givesItem` system
- Replaced removed skulduggery quest reward
- Fixed `get all <container>` command support
- Fixed leaderboard stale stats and scholar `onGive` handler

### Combat & Mob Fixes
- Made mob commands darkness-aware (mobs no longer act normally in
  pitch-dark rooms)
- Enabled wolf vs boar predator/prey combat on the steppe
- Fixed web client auto-scroll behavior

### UI & Display Fixes
- Reorganized status template into logical sections
- Fixed per-player buy/equip tracking with purchase debug logging
- Renamed 'back corner' room noun to 'alcove' to fix room 472 crash
- Removed ANSI tags from descriptions where the word is also a noun
  key
- Removed long-range exits from Thornwall City templates
- Fixed Thornwall cardinality, Brecca shop inventory, copper ring
  naming
- Web portal "Who's Online" now uses Title instead of removed
  Profession field

### Documentation
- Added deployment troubleshooting guide for git sync and Docker
  cache issues
- Added compose file warnings and port conflict troubleshooting
