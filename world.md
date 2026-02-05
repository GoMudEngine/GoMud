# Delusions of Grandeur MUD - Design Document

## Executive Summary

**Delusions of Grandeur** (DOG) is a MUD set on Gaius, a colony world where a crashed ship's survivors have forgotten their origins. A symbiotic organism infects the population, making beliefs and convictions become reality—manifesting as "magic" and physical mutations. The game features skill-based progression, a belief-powered magic system, statistical combat resolution, a resource-based economy, and rich NPC interactions in a world that blends medieval fantasy with hidden sci-fi elements.

---

## Table of Contents

1. [World & Setting](#world--setting)
2. [Core Mechanics](#core-mechanics)
3. [Combat System](#combat-system)
4. [Magic & Mutations](#magic--mutations)
5. [Economy & Crafting](#economy--crafting)
6. [Death & Respawn](#death--respawn)
7. [Travel & Stamina](#travel--stamina)
8. [NPCs & AI](#npcs--ai)
9. [Social Systems](#social-systems)
10. [Communication Channels](#communication-channels)
11. [Calendar & Festivals](#calendar--festivals)
12. [Major Locations](#major-locations)
13. [The Royal Bloodline & Ancient Relics](#the-royal-bloodline--ancient-relics)
14. [Implementation Notes](#implementation-notes)

---

## World & Setting

### Gaius - The World

**Planet Gaius** orbits a G/K-type dwarf star (0.91 solar masses, 0.754 solar luminosity, radius 0.825) at 0.95 AU with an orbital eccentricity of 0.06. The planet receives approximately 83.5% of Earth's solar radiation, creating a naturally cooler world with a "cool temperate" baseline climate.

**Planetary Characteristics:**
- **Year Length:** 344.125 days
- **Day Length:** 25 hours (using modified time units: 1 Gaius second = 0.9 Earth seconds, 1 Gaius minute = 100 Gaius seconds, 1 Gaius hour = 50 Gaius minutes, 20 Gaius hours = 1 Gaius day ≈ 90,000 Earth seconds)
- **Axial Tilt:** 27.5° (compared to Earth's 23.5°)
- **Seasons:** Short hot summer, moderate spring and fall, short harsh winter (Northern Hemisphere, where play occurs)
- **Tropics:** Extend to 27.5° latitude
- **Arctic Circles:** Begin at 62.5° latitude

**Climate Implications:**
- Larger tropical regions and more extreme seasonal temperature swings than Earth
- Powerful storm systems during seasonal transitions
- The Northern Hemisphere experiences a short, intense summer when the planet is at perihelion (closest approach to the star)
- The eccentricity creates approximately 10-15% difference in seasonal lengths

### The Three Moons (Secret: They're Orbiting Ships)

The populace believes three moons orbit Gaius. In truth, these are colony ships from the original expedition. The people's belief in the moons' influence on magic makes it real.

| Moon Name | Orbital Period | Orbital Distance | Visual Size | Affects Stat |
|-----------|---------------|------------------|-------------|--------------|
| **Swiftmoon** | 4.7 days | ~200,000 km | Large (2x Luna) | Dexterity |
| **The Wanderer** | 10.6 days | ~350,000 km | Similar to Luna | Perception |
| **The Eye** | 21.1 days | ~550,000 km | Small, bright | Charisma |

**Moon Phase Effects on Stats:**
- **New Moon** (dark): -5% to associated stat
- **Crescent**: -2% to associated stat
- **Half**: 0% (baseline)
- **Gibbous**: +2% to associated stat
- **Full**: +5% to associated stat

**Special Lunar Events:**
- **Triple Full Moon** (~every 44 days): All three stats boosted, major cultural celebration (The Convergence Festival)
- **Triple New Moon** (~every 44 days): All three stats penalized, feared time
- **The Convergence** (perfect alignment): Extremely rare, legendary event

Because the moons have low mass (20% of Luna) but are closer, tidal patterns are complex and erratic. True darkness is rare, as at least one moon is usually visible, affecting nocturnal life and plant biology.

### History - The Crash

Approximately 10,000 years ago, a colony ship carrying humans from Earth crashed on Gaius. The crash resulted in:
- Massive loss of life
- Loss of almost all advanced technology
- Survivors scattered and forced to rebuild from nearly nothing
- The three orbiting "moons" are the other ships from the expedition, still in orbit

**Technology Level:** Medieval Europe with magic (actually mutation-based powers)

**Lost Knowledge:** The truth of Earth, the ships, and technology has been almost entirely forgotten. Only fragments remain as myths, legends, and misunderstood relics.

### The Chrysalis - The Infection

A symbiotic organism infects almost all humans on Gaius. The infection is environmental—present in the air, water, and soil of Gaius itself.

**Infection Characteristics:**
- **Onset:** Manifests at puberty; never before
- **Progression:** Becomes more pronounced with age until death
- **Mechanism:** Beliefs and convictions of the infected become physically real
- **Manifestation:** Physical mutations and reality-altering powers (perceived as "magic")
- **Transmission:** Environmental (nearly everyone infected); off-worlders can contract it
- **Resistance:** A small population is genetically resistant (recessive on multiple alleles)
- **Heredity:** Mutations reset each generation, but individuals often unconsciously emulate parents/mentors

**Cultural Perception:** The infection is worshipped by the dominant religion as "The Chrysalis" or "The Awakening"—a divine gift that allows transformation and power.

**Pre-Pubescent Humans:** Children are baseline humans with no mutations or powers until puberty.

### The Continent - Thera

The game takes place on **Thera**, the primary continent of Gaius (other continents may exist but are unexplored).

**Region:** **The Windward Marches** - A diverse region analogous to the Pacific Northwest, featuring varied microclimates in a relatively small geographic area.

---

## Core Mechanics

### Attributes

Players and NPCs are functionally identical, with 6 primary stats, derived secondary stats, and skills.

#### Primary Stats

| Stat | Description | Affects |
|------|-------------|---------|
| **Dexterity** | Physical coordination, speed | Hit chance, dodge, initiative, number of attacks |
| **Strength** | Raw physical power | Damage, carrying capacity, breaking objects |
| **Vitality** | Physical health and resistance | Health pool, poison/disease resistance, stamina |
| **Perception** | Learning ability, awareness | Skill progression rate, spell learning by observation, noticing details |
| **Willpower** | Mental fortitude and conviction | Conviction pool, resistance to mental effects, spell power |
| **Charisma** | Force of personality | Conviction pool, NPC reactions, prices, charm/persuasion |

**Starting Stats:** Randomly generated with a mean of 100 and standard deviation of 15. No reroll mechanic—progression is through use.

**Stat Distribution:** Because of the standard deviation, having a wide spread between different stats is normal and expected.

#### Secondary Stats

| Stat | Formula | Description |
|------|---------|-------------|
| **Health / Max Health** | Mostly Vitality + small portion Willpower | Physical health. 0 = comatose. -10 = death. |
| **Stamina / Max Stamina** | Mostly Vitality + fair portions Willpower & Strength | Physical stamina. 0 = comatose. -10 = death. Average is ~125. |
| **Conviction / Max Conviction** | Mostly Charisma + fair portion Willpower | Faith and reality-shaping ability. 0 = comatose. -10 = death. |
| **Encumbrance / Max Encumbrance** | Based on Strength + mutations | Weight/volume currently carried vs. max capacity. Some items encumber more than weight suggests (awkward or magical). |

### Skills

**Philosophy:** Broad skill categories (~150 total) organized into skill trees to avoid 1500+ micro-skills while maintaining meaningful choice.

**Skill Categories:**

#### Combat Skills
- Melee Combat
- Unarmed Combat
- Ranged Combat
- Defense/Evasion

#### Magic Skills
- Spellcasting (general)
- Psionics
- Elemental Magic
- Enhancement Magic
- Mental Magic
- Vital Magic

#### Crafting Skills
- Blacksmithing
- Leatherworking
- Alchemy
- Cooking
- Carpentry
- Tailoring
- (Additional crafting skills as needed)

#### Survival Skills
- Tracking
- Stealth
- Climbing
- Swimming
- Foraging
- Animal Handling

#### Social Skills
- Persuasion
- Intimidation
- Deception
- Bartering
- Teaching
- Pickpocketing

#### Knowledge Skills
- Herbalism
- First Aid
- Linguistics
- Lore
- Appraisal

**Starting Skills:** New characters start with a suite of rank 1 skills (Unarmed Combat, basic survival skills, etc.)

**Skill Caps:** No hard cap, but progression difficulty increases exponentially. Moving past rank 50 for skills requires concerted, sustained effort. Early progression is relatively easy.

### Progression System

**No Traditional Levels or XP.** Progression is entirely use-based and organic.

**Progression Triggers:** Very small chance to increase attributes/skills when:
- Rolling a critical success
- Rolling a critical failure
- Taking a wound or dropping health/stamina/conviction very low
- Completing notable deeds or misdeeds

**Progression Modifiers:**
- Events can increase/decrease likelihood of progression (minor, subtle bonuses)
- **Deliberate Training:** Players can train skills deliberately (e.g., practicing on dummies) but with significantly lower progression chance than actual use, creating a soft cap
- **Below Baseline:** Stats below 100 progress faster back toward 100
- **High Perception:** Increases progression rate across the board

**Difficulty Curve:** Modified exponential curve—easy at first, increasingly difficult as you improve, very hard to push past soft caps.

### Dice Rolling & Action Resolution

**System:** Statistical distribution-based rolling.

**Roll Calculation:**
1. Determine mean: Relevant Attribute + Skill Level
2. Standard deviation: 15% of the mean
3. Roll a number from a distribution with that mean and standard deviation

**Critical Success/Failure:**
- **Critical Hit:** Roll is 2 standard deviations above the mean
- **Critical Miss:** Roll is 2 standard deviations below the mean

**Opposed Rolls:** Most actions (attacks, social skills, etc.) are contested:
- Attacker rolls their offensive score
- Defender rolls their defensive score
- Higher roll wins

**Examples:**
- **Melee Attack:** Attacker rolls (Dexterity + Melee Combat skill) vs. Defender rolls (Dexterity + Defense skill)
- **Persuasion:** Persuader rolls (Charisma + Persuasion) vs. Target rolls (Willpower + appropriate resistance)

---

## Combat System

### Initiative & Turn Order

**Initiative Determination:**
- Rolled once at combat start based on: Dexterity + relevant combat skill + modifiers from negative ailments (encumbrance, wounds, exhaustion)
- Turn order maintained for entire combat
- New combatants joining mid-fight slot into existing turn order based on their initiative roll

**Number of Attacks Per Round:**
- Determined at the start of each round
- Baseline: 1 attack
- Factors: Weapon speed, skill level, Dexterity, encumbrance, mutations
- Progression: 3 attacks is fairly common for very skilled combatants; 5+ attacks is extremely rare even for the most exceptional (exponential scaling)

### Positioning

**Initial Implementation:** Simple positioning—prone, sitting, standing, knocked down.

**Future Implementation:** ASCII overhead combat map with flanking bonuses for attacks from sides/behind.

### Attacks & Damage

**Attack Roll:** (Dexterity + Combat Skill) vs. (Defender's Dexterity + Defense Skill)

**Damage Calculation:**
- Base weapon damage multiplier × (Strength or relevant stat)
- Modified by armor

**Armor Mechanics:**
- Physical armor reduces damage
- Can be hit repeatedly without taking damage if armor is sufficient
- **Armor Bypass:** Critical success with roll of (2 × Defender's Physical Defense + Physical Armor for worn armor) OR (2 × Defender's Physical Defense + 3 × Physical Armor for natural/innate armor)

**Critical Effects:**
- **Critical Hit:** Armor bypass, potentially extra damage or special effects for epic weapons
- **Critical Miss:** Drop weapon, fumble

**Epic Weapons:** May have special critical effects (disarm, knockdown, bleeding condition, etc.)

### Conditions

Conditions are tags that modify rolls, apply damage/regen over time, or unlock abilities.

**Example Conditions:**
- Burning
- Resting/Sleeping
- Grappled
- Encumbered (light, medium, heavy)
- Mutation (extra limbs, thick scales, etc.)
- Regen (from potions, healing spells, magical locations)
- Wanted (criminal status)
- Prone/Knocked Down
- Bleeding
- Poisoned
- Blinded

**Duration:** Most conditions have durations (number of rounds, game hours, etc.)

### Fleeing & Mercy

**Fleeing:**
- An action that requires a successful roll to escape
- Some enemies may pursue based on disposition
- Can set a "wimpy" threshold (% of health/stamina/conviction) to auto-attempt flee

**Mercy System:**
- When a combatant reaches 0 health/stamina/conviction, they become comatose
- Opponents can choose to finish them off (kill) or show mercy
- **Mercy Flag:** Toggle whether you finish off comatose opponents
- **Comatose State:** Drop out of combat; allies can apply healing (no instant healing—only accelerated regeneration via conditions)

### NPC Combat Behavior

- Some NPCs fight to the death
- Some NPCs flee when wounded or outmatched
- Behavior based on NPC personality, disposition, and circumstances

---

## Magic & Mutations

### The Four Spell Schools

Magic and mutations are manifestations of The Chrysalis infection—beliefs made real.

**Spell Schools (as tags, spells can belong to multiple):**

| School | Description | Examples |
|--------|-------------|----------|
| **Elemental** | Physical manifestations of natural forces | Fire, ice, lightning, earth, wind attacks and manipulation |
| **Enhancement** | Body modification and augmentation | Strength buffs, speed boosts, toughened skin, enhanced senses |
| **Mental** | Mind-affecting powers | Psionics, illusion, charm, fear, telepathy, influence |
| **Vital** | Life and health manipulation | Healing, poison, disease, life drain, energy transfer |

### Learning Spells

**Methods:**
1. **Trainers:** Learn from an NPC teacher (costs vary based on spell rarity/power)
2. **Other Players:** Learn from another player via teaching
3. **Observation:** High Perception allows learning by watching spells cast in the same room (fairly rare unless Perception is very high; multiple exposures may be needed)
4. **The First Awakening:** Completing the tutorial's main quest includes a ritual that grants the first mutation/power

**Spell Costs:** Vary by spell—may cost Health, Conviction, or induce temporary status effects. More powerful spells have higher costs.

**Shared Beliefs:** Multiple players using the same powers together may strengthen them through shared belief (passive buffs as conditions).

### Mutations

**Acquisition:**
- Random with weighted chances based on actions and observations
- More common early in a character's "career"
- Progressively harder to acquire additional mutations
- More powerful/desirable mutations are rarer (weighted)
- Having more than 3 mutations is rare but possible

**Mutation Examples by Tier (Tiers determine acquisition probability):**

**Tier 1 (Common):**
- Enhanced strength/bone density
- Increased speed
- Night vision/superior eyesight
- Toughened skin

**Tier 2 (Uncommon):**
- Claws/natural weapons
- Poison resistance
- Water breathing
- Natural camouflage

**Tier 3 (Rare):**
- Wings/flight
- Fire breath
- Electrical discharge
- Regeneration

**Tier 4 (Very Rare):**
- Multiple limbs (4-6 arms, etc.)
- Telekinesis (psi-like force manipulation)
- Shapeshifting abilities
- Telepathy

**Morphological Changes:**
- Mutations cause visible physical changes (wings, claws, dappled skin, larger muscles, altered bone structure, etc.)
- Characters remain mostly humanoid but can become quite exotic
- Dynamic descriptions: Base character description modified by acquired mutations

**Mutation Conflicts & Synergies:**
- Handled organically (wings + heavy armor = can't fly; wings + hollow bones = easy flight but fragility)
- Conflicting mutations discovered through play (fire breath vs. ice breath would conflict)
- Future implementation: spells or NPCs that can remove unwanted mutations (very expensive, rare, possibly requiring quests and rare ritual components)

### Belief Mechanics

**Core Principle:** Conviction and belief shape reality.

**Critical Success/Failure:** Can strengthen or weaken beliefs, potentially unlocking new abilities or losing confidence in existing ones.

**Seasons & Moons:** People believe these affect power, so they do (moon phase stat modifiers, seasonal effects).

**Collective Belief:** Large groups sharing a belief focus can create powerful effects. The most dangerous individuals are those who can convince many to share a belief system.

**Mind Control/Influence:** Player-achievable but very difficult and resistable (target resists both consciously and unconsciously). Even simple effects like Taunting can be profound if done well. Balance must be carefully monitored.

**Discovering the Truth:** Players who discover the truth about the ships/Earth/infection face a dilemma—NPCs won't believe or understand them, and they may be hunted by religious groups and the royal bloodline.

---

## Economy & Crafting

### Resources

**Categories:** Broad resource types with quality tiers. Descriptions are fluid, but mechanics are determined by type and quality.

**Resource Types (examples):**
- Metals (iron, copper, silver, gold, etc.)
- Wood (hardwood, softwood, exotic woods)
- Textiles (cotton, wool, silk, leather)
- Food (grains, meats, produce, preserved goods)
- Magical Reagents (herbs, crystals, rare components)
- Stone (granite, marble, sandstone)
- Animal Products (hides, bones, feathers, etc.)

**Quality Tiers (5 levels):**
1. Poor
2. Common
3. Fine
4. Exceptional
5. Masterwork

**Resource Gathering:**
- Part of crafting skills (Mining, Lumberjacking, Herbalism, Hunting/Skinning)
- Good for early progression and earning money
- Gathering has lower progression chance than actual crafting

### Crafting System

**Recipe Acquisition:**
- Purchase from vendors
- Find as loot or quest rewards
- Learn from NPCs (may require minimum skill threshold and/or fee)
- **Experimentation:** Players can combine materials to discover new recipes (critical failures burn materials and waste cooldown time)

**Crafting Process:**
- Requires appropriate skill, materials, and sometimes tools/facilities (forge, alchemy lab, etc.)
- Success chance based on: (Crafter Skill + Material Quality) vs. difficulty
- **Time Cost:** Cooldown based on item complexity (see Crafting Cooldowns section)
- **Failure:**
  - **Normal Failure:** Wasted time/cooldown, potential skill progression
  - **Critical Failure:** Junk produced, materials lost, potential skill progression

**Item Quality:**
- Determined by: Crafter Skill + Material Quality
- Crafter skill is equally or more important than material quality
- Higher quality items are more durable, effective, valuable

**Multi-Stage Crafting (Epic Items):**
- Very powerful items require multiple stages
- Each stage has a separate failure chance
- **Critical failure at any stage ruins the entire piece**

**Example: Masterwork Sword**
1. Smelt ore into ingots (1 game hour, ~1 real minute)
2. Forge blade (3 game hours)
3. Harden/temper blade (6 game hours)
4. Create hilt/guard (1 game hour)
5. Assemble and polish (2 game hours)
6. **Total:** 13 game hours (13 real minutes)

### Economy Mechanics

**Regional Scarcity:**
- Different regions have gluts and scarcities of specific resources
- Prices vary by region based on supply/demand
- Price formula: Base Value × Scarcity Multiplier

**NPC Caravans:**
- Move resources between cities (future implementation)
- Can be attacked/robbed by players or NPCs
- Players can also transport goods for profit

**Player Impact:**
- Gathering resources affects local availability
- Moving goods affects prices
- Disrupting caravans affects regional economy

**Gold:**
- Has weight (carrying large amounts is encumbering)
- Can be stolen by NPCs (pickpocketing, robbery)
- Depositing in banks is incentivized

### Banking System

**The Magical Banking Network:**
- Global bank system, magically connected (because everyone believes it is, it is)
- Deposit at any branch, withdraw at any branch
- One account per character

**Fees:**
- Deposit: Free
- Withdrawal: Small fee (<1% of withdrawal amount)

**Integration:**
- Auction house directly tied to bank for deposits/withdrawals

### Auction House

**Listings:**
- 5% listing fee per game month based on buyout price
- Sellers set a buyout price
- If buyout not met, item sells to highest bid at expiration
- Listing for multiple months requires paying for each month

**Bidding:**
- Players can bid or use "buy now" at buyout price
- NPC vendors can buy materials they need for crafting (future implementation)
- NPCs do not buy equippable items

**Communication:**
- Global Auction channel for announcements

### NPC Crafters

- NPCs craft autonomously based on available resources and demand (future implementation)
- NPCs eventually pay higher prices for material components they need
- Players can commission NPCs to craft specific items via quest-like system

---

## Death & Respawn

### Death Mechanics

**Death Occurs At:** -10 Health, Stamina, or Conviction

**Death Penalties:**

| Death Count (within 24 game hours) | Penalty |
|-------------------------------------|---------|
| **1st Death** | No penalty |
| **2nd+ Deaths** | Progressively worse random penalty to a primary stat or skill (permanent) |

**Penalty Details:**
- Random stat or skill is chosen
- Penalty worsens with each subsequent death in the 24-hour window
- **Floor:** Skills cannot drop below rank 1, stats cannot drop below ~50
- **Recovery:** Players must "relearn" the stat/skill via the progression system
- **Bounty Deaths:** Dying to a bounty hunter incurs a slightly worse penalty or accelerated worsening scale

### Respawn

**Location:** Last city visited, at the Temple of the Chrysalis

**Respawn Points:** Limited number of respawn locations (major cities/towns with temples)

**Goal:** Keep respawn locations fairly small in number to maintain world coherence

---

## Travel & Stamina

### Time Ratio

**60:1 Ratio:** 1 game hour = 1 real-world minute

**Implications:**
- 1 game day (25 hours) = 25 real minutes
- Crafting/cooldowns feel reasonable
- Long-distance travel requires planning

### Stamina Costs

**Average Character:** ~125 Stamina (Vitality + Willpower + Strength)

**Movement Costs:**

| Terrain / Condition | Stamina Cost Per Room |
|---------------------|----------------------|
| Flat terrain, unencumbered | 2 stamina |
| Rough terrain and/or encumbered | 5-10 stamina |
| Very rough terrain and/or heavily encumbered | 15-20 stamina |
| Uphill | 2-3x base cost |

**Examples:**
- Unencumbered on flat road: 2 stamina/room
- Heavily encumbered uphill in rough terrain: 20+ stamina/room

### Stamina Regeneration

**Baseline:** 2 game hours to fully regenerate stamina while resting (~2 real minutes)

**Modifiers:**
- Resting: Faster regen
- Moving/Crafting: Slower regen
- Conditions (buffs, potions, magical locations): Faster regen
- Injuries/Exhaustion: Slower regen
- Food/Drink: Faster regen (see Eating & Drinking)

**Background Tick:** Passive regeneration even while active, just slower than resting

### Mounts & Pack Animals

**Function:** Act like party members (not listed in party interface)

**Benefits:**
- Carry items for the player
- Reduce encumbrance penalties
- Allow moving large quantities of goods (profitable for traders)

**Mounted Travel:**
- While actively mounted, the mount's stamina is consumed instead of the rider's
- Mounts can tire and become annoyed if overworked
- Feeding mounts improves stamina regen and disposition

**Availability:** Pack animals, riding animals, and wagons/carriages can be crafted or purchased

### Travel Distances (Stamina Bars)

Travel between major locations measured in "stamina bars" (1 bar ≈ 33-42 stamina for average character):

| Destination | Stamina Bars | Approximate Distance |
|-------------|--------------|---------------------|
| Tutorial Area (Sanctum Basin) to nearby forest | 3 bars | ~100-125 stamina |
| Tutorial Area to Stillwater (Moses Lake analogue) | 3 bars | ~100-125 stamina |
| Tutorial Area to Irongate (Spokane analogue) | 6 bars | ~200-250 stamina |
| Tutorial Area to New Plymouth (Seattle analogue) | 6 bars | ~200-250 stamina |

**Note:** Distances are approximate and will be tuned during implementation based on feel.

---

## NPCs & AI

### NPC Equality

**Philosophy:** NPCs are functionally identical to players.
- Same stats, skills, progression system
- Can acquire mutations
- Can learn spells by observation or training
- Progress through use and critical successes/failures

### NPC Schedules

**Future Implementation:** NPCs follow schedules based on their role and disposition.

**Examples:**
- Shopkeeper opens shop at dawn, closes at dusk, goes home
- Guard patrols specific routes at specific times
- Farmer works fields during day, sleeps at night

**Dynamic Schedules:**
- NPCs can abandon roles based on life events (e.g., blacksmith's family killed → becomes bounty hunter)
- Major world events can shift NPC behaviors

### NPC Disposition & Memory

**Individual Disposition System:**
- Each NPC tracks a disposition score (-100 to +100) toward each player
- Disposition affects: prices, willingness to teach, hostility, quest availability

**Disposition Changes:**
- Positive actions: helping, gift-giving, completing quests, successful trading
- Negative actions: theft, violence, failed persuasion, betrayal

**Gossip & Reputation Spread:**
- NPCs gossip within their social networks
- Information spreads 2-3 degrees of separation (blacksmith tells close friends)
- Major events (murder, heroic deeds) spread further than minor events (successful trade)

**Complexity Management:**
- Gossip limited to prevent explosion of complexity
- Not every NPC knows everything about every player

### NPC Crafting

**Future Implementation:**
- NPCs craft autonomously based on available resources
- NPCs pay higher prices for components they need
- Players can commission specific crafts via quest system

### AI Agents

**Long-Term Vision:** AI agents could be integrated as "players" or assigned specific NPC roles to see emergent behaviors.

**Current Plan:** Build hooks for future AI integration, but this is a much later development goal.

---

## Social Systems

### Crime & Justice

**Crime Tiers:**

| Tier | Crime Types | Penalties |
|------|-------------|-----------|
| **Minor Offenses** | Petty theft, trespassing, public disturbance | Fine + temporary "Petty Criminal" tag |
| **Serious Crimes** | Assault, major theft, fraud | Beating + jail time + "Wanted" tag + bounty |
| **Capital Crimes** | Murder, treason, arson | Execution or "Outlaw" status (kill on sight, banned from towns) |
| **Exile Crimes** | Repeat offenses, crimes against the community | Permanent outcasting from specific city-state |

**Jail Mechanics:**
- Players sit in a cell
- Can attempt to escape (additional crime if caught)
- Jail time based on severity

**Guard Behavior:**
- Minor: Approach and fine or eject from area
- Serious: Arrest and jail
- Capital: Kill on sight or attempt capture for execution (depends on city/guards)

### Bounty System

**Bounties:** Posted for capital crimes only

**Claiming Bounties:**
- Both players and NPCs can claim
- Requires bringing back the target's head
- Bounty deaths incur worse stat penalties (or accelerated penalty scaling)

### Reputation

**City-Level Reputation:**
- Each city-state tracks player reputation as a whole
- Affects: guard hostility, prices, access to certain areas or quests

**Individual Disposition:**
- See NPC Disposition & Memory section

### Party System

**Party Mechanics:**
- Players can form parties (no shared XP since there is no XP)
- **Confidence Buff:** Being in a party provides a slight buff to all skills/stats
- **Observational Learning:** Chance to learn from party members' critical successes and failures
- **Follow Command:** Auto-move with party leader
- **Party Chat:** Private communication channel
- **Shared Vision:** See party member health/status (future implementation)

**Max Party Size:** TBD (probably 5-8 players)

### Multiplayer Restrictions

**General Rule:** Multiplayer (one player controlling multiple characters) is banned.

**Exception:** Limited multiplayer allowed in designated "Inn" areas for transferring items/money between a player's own characters.

**Enforcement:** Technical and policy-based enforcement mechanisms TBD.

---

## Communication Channels

Players can toggle which channels they listen to via flags.

| Channel | Scope | Purpose |
|---------|-------|---------|
| **Say** | Local room only | Normal conversation |
| **Shout** | Area-wide | Emergencies, guard calls, public announcements |
| **Whisper** | Private, one target | Private conversation |
| **Auction** | Global | Auction house announcements and bids |
| **Newbie** | Global | Help for new players |
| **OOC** | Global | Out-of-character talk, rules questions, general chat |
| **Dev/Bug** | Global | Reporting bugs, asking for developer help |
| **Party** | Party members only | Private party communication |
| **Guild/Faction** | Guild/faction members | (Future implementation) |

---

## Calendar & Festivals

### Calendar System

**Year:** 344.125 days, divided into seasons

**Seasons (Northern Hemisphere):**
- **Short Hot Summer** (~70 days): Intense heat, perihelion coincides with NH summer
- **Long Moderate Spring** (~90 days): Warming, transitional storms
- **Long Moderate Fall** (~90 days): Cooling, transitional storms
- **Short Hard Winter** (~90 days): Harsh cold, far from star

**Months:** People likely track time using moon cycles rather than fixed months.

**Weeks:** Possibly 5-day weeks based on Swiftmoon's ~5-day cycle.

### Festivals & Holidays

**Goal:** ~15 holidays per year

**Types of Holidays:**

**Lunar Festivals:**
- **The Convergence Festival** (Triple Full Moon, ~every 44 days): Major celebration when all three moons are full. Peak power, feasting, rituals. (~8 per year)

**Seasonal Festivals:**
- **Spring Festival** (Planting, renewal)
- **Summer Festival** (Peak of heat and light)
- **Harvest Festival** (Fall, gathering resources for winter)
- **Winter Festival** (Solstice, survival, endurance)

**Civic Holidays:**
- **Founder's Day** (New Plymouth): Celebrates the royal bloodline founder (actually the ship captain's descendant, but mythologized)
- **City Founding Days:** Each major city celebrates its founding

**Festival Mechanics:**
- Mostly flavor: NPCs take the day off, shops closed, special decorations/dialogue
- Bank holidays (can't access banking, auction house?)
- Buffs are separate, tied to moon system (not festivals themselves)
- Possible: Special quests, temporary NPCs, rare goods available

---

## Major Locations

### Geography Overview

**Continent:** Thera
**Region:** The Windward Marches (Pacific Northwest analogue)

**Biomes:**
- High desert plateau (starting area)
- Dense temperate rainforest (west)
- Mountain ranges (Cascade analogues)
- River gorges
- Coastal areas (far west)
- Dry steppe/desert (east)
- Farmland and river valleys

### Tutorial Area - Sanctum Basin

**Location:** High desert plateau near basalt rock formations (Feathers analogue). The tutorial area is built in a gigantic basalt plunge pool—a natural bowl formation with a small lake inside.

**Isolation:** Single exit from the tutorial area to maintain separation from the main world.

**Key Locations:**
- **Sanctuary of the Chrysalis** (The Academy): Religious school worshipping The Chrysalis infection
  - Teaches basic game mechanics
  - Performs ritual to awaken first mutation (main quest reward)
  - Visibly mutated teachers demonstrating different paths
- **Training Grounds:** Combat dummies, practice areas
- **Small Town:** Basic shops, NPCs for quests
- **Observatory:** Teaches moon phase mechanics and buffs (run by a trainer)
- **Gathering Areas:** Resource nodes for practicing gathering skills
- **Cave System / Dungeon:** Low-level challenge area

**Starting Gear:**
- Basic clothing
- Small amount of money
- Simple weapon obtainable in tutorial (quest, loot, or purchase)
- Quest rewards provide slightly better (but still poor quality) starting gear

**Graduation:**
- No specific graduation event
- Players leave when ready (exit the tutorial area)
- No stat reroll mechanic (removed from design)

### Major Cities

| City | Real-World Analogue | Description |
|------|---------------------|-------------|
| **New Plymouth** | Seattle | Largest city-state. Seat of the royal bloodline. Coastal. ~150 rooms initially. |
| **Tidemark** | Tacoma | Port city south of New Plymouth. Trade hub. |
| **Greenford** | Portland | Southern city on major river crossing. Green, forested surroundings. |
| **Irongate** | Spokane | Eastern city. Fortified. Mining and industry. |
| **The Confluence** | Tri-Cities | Where three rivers meet. Agricultural and trade center. |
| **Amber Valley** | Yakima | Sunny valley town. Farming community. |
| **Stillwater** | Moses Lake | Lake town. Fishing and agriculture. |

**Plus:** Smaller farmsteads, hamlets, and outposts scattered between major cities.

### New Plymouth (The Capital)

**Size:** ~150 rooms to start, expandable later

**Under Construction:** Some areas blocked off or marked as "under construction" for flavor as the game world expands.

**Districts (Walled Sections):**

**1. Trade District**
- Market square
- Auction house
- Bank (Temple of the Chrysalis branch)
- Merchant shops
- Warehouses

**2. Crafting Quarter**
- Forges, smithies
- Alchemy labs
- Workshops for various crafts
- Master trainers for advanced skills

**3. Noble Quarter**
- Royal palace (home of the bloodline and the energy weapon relic)
- Estates of wealthy citizens
- Guarded gates

**4. Docks**
- Ships, trade vessels
- Smugglers, black market
- Taverns, rougher establishments
- Ferry services to other coastal cities

**5. Slums / Common Quarter**
- Poor housing
- Thieves, outcasts
- Potential crime-related quests
- Cheap vendors

**6. Temple District**
- Grand Temple of the Chrysalis
- Respawn point
- Healing services (regen buffs)
- Religious quests and lore

**Access from Tutorial Area:** 6 stamina bars (~200-250 stamina for average character). Accessible from the start but requires planning and possibly a mount or high stamina pool.

### Political Landscape

**Current Simplicity:** Most settlements pay taxes/tribute to New Plymouth.

**Resentment:** Some resentment toward the royal bloodline exists but is not (yet) open rebellion.

**Future Complexity:** Could add:
- Different governments (councils, warlords, theocracies)
- Cultural differences (some cities embrace mutations, others fear them)
- Trade rivalries
- War or conflict between city-states

---

## The Royal Bloodline & Ancient Relics

### The Royal Bloodline

**Origin:** Descendants of the colony ship's captain. Uninfected due to genetic resistance (recessive on multiple alleles).

**Knowledge:**
- Know they are descendants of someone important
- Title has morphed into a king-equivalent (different name, TBD: "The Sovereign," "High Keeper," etc.)
- Have lost knowledge of Earth and the ships
- Possess techno-legends passed down, diluted, and corrupted over 10,000 years
- Know more than anyone else but mostly have myths with kernels of truth

**Family Size:** Very low number of members, possibly inbred due to small population and desire to maintain uninfected lineage.

**Rule:** Openly govern New Plymouth and demand tribute from other cities.

**Breeding:**
- Uninfected × Uninfected = Uninfected children
- Uninfected × Infected = Low chance of uninfected children (recessive on multiple alleles)
- Bloodline carefully manages marriages to preserve uninfected status

**Secret Purge:**
- Actively hunt other uninfected outside the bloodline
- **Adults:** Killed
- **Children:** Brainwashed and adopted into the bloodline
- Justification: Maintain monopoly on ancient relics, prevent challenges to power

### The Energy Weapon

**Description:**
- Semi-sentient relic from the colony ship
- Looks technological but described from non-techie POV: strange metal, glowing parts, ominous hum, dangerous aesthetic
- Visible in the palace (heavily guarded)

**Function:**
- Only the uninfected can wield, carry, or touch it
- Limited power supply or requires long recharge (exact mechanics unknown to the bloodline—only stories)

**Response to Infected:**
1. Audible reaction when infected are near: dangerous hum, lights intensify
2. If infected person touches it: violent energy discharge, blinding light
3. If they don't drop it / back off: escalates to deadly energy discharge

**Cultural Impact:**
- Public knowledge of the weapon
- Horrifying cautionary tales of its use keep the bloodline in power
- Stories exaggerate and mythologize past uses

**Potential Quests:**
- High-level players might attempt to steal or study it
- Could be a key to accessing the crash site or the moons

### The Crash Site

**Location:** A large, oddly-shaped hill with a single entrance (the ship's hull buried and overgrown).

**Knowledge:**
- Known about by a few scholars/adventurers
- Rarely penetrated (maybe once a generation in lore)
- Considered a cursed place, taboo to discuss

**Exterior:**
- Half-broken ancient defenses (automated turrets, energy barriers—degraded)
- Dangerous even to approach

**Interior:**
- Very high-level zone (endgame content)
- Tons of traps and automated defenses
- Technological relics: medical equipment, computer cores (oracle stones), fabrication tools, weapons
- Severe degradation from 10,000 years
- Described from non-techie POV (glowing runes, magic artifacts, ancient mysteries)

**Rewards for the Clever:**
- Actual historical records (the truth about Earth, the crash, The Chrysalis infection)
- Possibility of accessing a shuttle or teleporter to reach the orbiting ships (moons)

### The Moons (Orbiting Ships)

**Access:** Via the crash site (if players are clever and high-level enough).

**Content:**
- Functional ship areas: bridge, medical bay, armory, cryogenics, engineering
- Zero or low gravity mechanics (future implementation challenge)
- Hostile ship AI or automated defenses
- Powerful technological rewards (energy weapons, advanced armor, medical nanites)

**The Truth:**
- Players can discover Earth's history, the colony mission, the infection's origin
- NPCs will not believe or understand if players try to share this knowledge
- **Consequences:**
  - Religious groups may hunt the player as a heretic
  - The royal bloodline will hunt the player to silence them
  - Knowledge is dangerous but could unlock powerful rewards

**Infection on the Ships:**
- The ships are sterile; The Chrysalis infection is specific to Gaius
- Infected characters may suffer penalties or lose access to mutations while on the ships
- Uninfected characters (if players can eventually create them) have advantages in this area

### Other Uninfected Humans

**Population:** Very few scattered across the world.

**Self-Perception:** Think they are cursed or crippled (lack the "gift" everyone else has).

**Secret Societies:**
- Small, hidden groups of uninfected
- Hard to find
- May possess fragments of true knowledge passed down
- Potential quest-givers or allies for high-level content

**Bloodline's Response:**
- Adults: Killed if discovered
- Children: Brainwashed and adopted into the bloodline
- Secrecy is survival for the uninfected

---

## Implementation Notes

### Starting Point - Modifying GoMud Codebase

**Existing Codebase:** GoMud provides a solid foundation but requires significant modification.

**Major Changes Required:**
1. **Remove Levels:** Replace level-based progression with skill-use progression
2. **Replace Races with Species/Family:** Rename and refactor race system (fewer than 50-100 categories for animals, NPCs, etc.)
3. **Rework Stats:** Replace existing stats with the 6 primary stats + 3 secondary pools
4. **Implement Distribution-Based Rolling:** Replace existing dice mechanics with statistical distribution system
5. **Container System:** Ensure everything can contain everything (with weight/volume checks and circular containment prevention)
6. **Remove Incompatible Systems:** Strip out any GoMud features that conflict with DOG's design

**Development Philosophy:**
- Focus on core mechanics first
- Make everything extensible for future features
- Build hooks for later systems (AI agents, complex economy, etc.)
- Iterative development: get a working (if limited) MUD, then expand

### Development Phases (Suggested Priority)

**Phase 1: Core Engine**
- Rooms, movement, containers, object system
- Time system (game time, real-time ratio, day/night cycles)
- Basic commands (look, inventory, get, drop, etc.)
- Weight/encumbrance system

**Phase 2: Stats & Progression**
- Implement 6 primary stats + secondary stats
- Stat generation for new characters
- Skill system framework
- Distribution-based rolling system
- Progression triggers (critical success/failure, deeds)

**Phase 3: Basic Combat**
- Initiative and turn order
- Attack/defense rolls
- Basic damage calculation
- Health/stamina/conviction tracking
- Simple conditions (prone, wounded, etc.)
- Fleeing and mercy mechanics

**Phase 4: NPCs**
- NPC spawning and basic behaviors
- NPC stats and skills (mirror player system)
- Basic conversation system
- Disposition tracking
- Simple NPC AI (aggressive, friendly, neutral)

**Phase 5: Economy Basics**
- Gold with weight
- Basic shops (buy/sell)
- Simple crafting (single-stage, no failures yet)
- Resource gathering skills

**Phase 6: Tutorial Area**
- Build Sanctum Basin
- Sanctuary of the Chrysalis (Academy)
- Tutorial quests
- Starting town and NPCs
- Observatory
- Low-level content for testing all systems

**Phase 7: Magic & Mutations**
- Spell system (4 schools)
- Spell learning (trainers, observation)
- Mutation acquisition system
- Dynamic descriptions based on mutations
- Conviction costs and effects

**Phase 8: Advanced Combat**
- Multi-attack system
- Weapon properties (speed, reach, handedness)
- Armor and armor bypass mechanics
- Critical hit effects
- Status effects and conditions (burning, poison, etc.)

**Phase 9: Advanced Economy**
- Regional scarcity and price variation
- Multi-stage crafting
- Crafting failures and critical failures
- Quality tiers for items
- NPC crafters (basic)

**Phase 10: Social Systems**
- Crime and justice system
- Jail mechanics
- Bounty system
- City-level reputation
- Party system

**Phase 11: Communication & UI**
- All communication channels
- Channel flags/toggles
- Moon phase display
- Stat/skill display improvements

**Phase 12: World Building**
- Build major cities (New Plymouth first, ~150 rooms)
- Build other cities and towns
- Connecting wilderness areas
- Mounts and pack animals
- Travel system (stamina costs, distances)

**Phase 13: Banking & Auction**
- Global banking system
- Auction house
- Listing fees and bidding mechanics

**Phase 14: Calendar & Festivals**
- Implement full calendar
- Moon phase mechanics (stat buffs/debuffs)
- Seasonal effects
- Festival events and schedules

**Phase 15: Advanced NPC Features**
- NPC schedules
- Gossip and reputation spread
- Dynamic NPC behaviors (role abandonment based on events)
- NPC crafting autonomy

**Phase 16: High-Level Content**
- The Crash Site dungeon
- Energy weapon quest line
- Moon/ship content
- Uninfected secret societies

**Phase 17: Polish & Balance**
- Extensive testing and balance adjustments
- Mutation system tuning
- Economy balance (scarcity, prices, crafting)
- Combat balance
- Progression rate tuning

**Phase 18: Future/Experimental**
- AI agents as NPCs or "players"
- ASCII overhead combat map
- Advanced mutation synergies and conflicts
- Complex faction/guild systems

---

## Appendices

### Species/Family System

**Philosophy:** Keep categories small and manageable (under 50-100).

**Categories are for Animals and NPCs, not player races.** All players are human.

**Example Categories:**
- Humans (players and NPCs)
- Canines (dogs, wolves)
- Felines (cats, large cats)
- Avians (birds, various types)
- Equines (horses, mules, donkeys)
- Bovines (cattle, bison)
- Reptiles (snakes, lizards)
- Fish (various)
- Insects (giant insects for enemies?)
- Magical Creatures (future, if introduced)

**Purpose:** Primarily for NPC behavior, loot tables, interaction mechanics.

### Eating & Drinking

**Philosophy:** No penalties for hunger/thirst (avoids tedious survival mechanics).

**Benefits:**
- Eating and drinking provide accelerated stamina and health regeneration (temporary condition)
- Feeding pack animals improves their stamina regen and disposition toward the feeder

**Availability:**
- Food and drink available for purchase (prices based on scarcity)
- Can be crafted via Cooking skill
- Can be gathered (Foraging, Hunting)

### Object System

**Everything is an Object:**
- Players, NPCs, items, containers, rooms (technically), furniture, etc.

**Object Properties:**
- **Health Pool:** All objects have health; can be broken or killed
- **Weight/Volume:** Encumbrance value
- **Container:** All objects can contain other objects (unless specifically locked by builder)
- **Nested Containers:** Objects can contain objects can contain objects (with checks to prevent circular containment: sword in bag in chest in sword ❌)
- **Wearable/Wieldable/Usable:** Flags for how objects can be used
- **Consumable:** Some objects are consumed on use (potions, food)
- **Repairable/Healable:** Most objects can be repaired or healed

**Epic Objects:**
- Small chance for objects to gain powers through use in epic deeds (e.g., slaying a legendary foe)
- Belief in an object's power can make it more powerful (legendary sword vs. random sword with same kill count)

### Degradation & Repair

**Future Implementation:**
- Objects degrade over time with use
- Weapons dull, armor dents
- Repair skill needed to restore

**For Now:** Focus on core systems; degradation is a later polish feature.

---

## Glossary

- **The Chrysalis / The Awakening:** Cultural/religious name for the symbiotic infection that grants mutations and powers.
- **Gaius:** The planet where the game takes place.
- **Thera:** The continent on Gaius where play occurs.
- **The Windward Marches:** The region of Thera analogous to the Pacific Northwest.
- **Sanctum Basin:** The tutorial area, a basalt plunge pool formation.
- **Sanctuary of the Chrysalis:** The tutorial academy, a religious school worshipping The Chrysalis.
- **New Plymouth:** The largest city-state, seat of the royal bloodline, analogous to Seattle.
- **The Convergence:** Perfect alignment of all three moons; extremely rare.
- **Swiftmoon, The Wanderer, The Eye:** The three moons (secretly orbiting colony ships).
- **The Bloodline / The Royal Family:** Uninfected descendants of the colony ship captain, rulers of New Plymouth.
- **The Energy Weapon:** A semi-sentient relic from the colony ship, only usable by the uninfected.
- **The Crash Site:** The buried wreckage of the colony ship, a high-level dungeon.
- **Conviction:** A secondary stat representing faith and belief, used to power spells and mutations.
- **Species/Family:** Categories for animals and NPCs (not player races; all players are human).
- **Stamina Bar:** Informal measure of travel distance (~33-42 stamina for average character).

---

## Conclusion

**Delusions of Grandeur MUD** is an ambitious project blending traditional MUD mechanics with innovative belief-based magic, organic skill progression, and a rich hidden-truth narrative. The design prioritizes player agency, emergent gameplay, and a living world where NPCs are peers, not props.

The core loop:
1. **Explore** the diverse world of Gaius
2. **Fight** using statistical combat and unique abilities
3. **Progress** skills and stats through use, not grinding
4. **Mutate** based on playstyle and beliefs
5. **Craft** and trade in a dynamic economy
6. **Discover** the truth about the world (if you dare)

Development will be iterative, starting with core systems and expanding outward. The existing GoMud codebase provides a foundation, but significant refactoring is required to realize DOG's vision.

**Next Steps:**
1. Finalize any remaining design questions
2. Set up development branch structure (per git workflow rules)
3. Begin Phase 1: Core Engine modifications
4. Build tutorial area as first playable content
5. Iterate and expand

Welcome to Gaius. The Chrysalis awaits.
