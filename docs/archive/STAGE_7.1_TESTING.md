# Stage 7.1 Testing Guide: Layered Defense System

## Quick Start
```bash
go build && ./GoMud.exe
```

## Manual Test Cases

### 1. Unarmed Combat (Dodge Only)
**Setup:** Create/find two unarmed characters
**Expected:** Both characters see only dodge messages
**Test:**
```
remove weapon
remove offhand
kill <target>
```
**Look for:** "You dodge X's attack!" messages only

### 2. Single Weapon (Dodge → Parry)
**Setup:** Equip weapon, no offhand
**Test:** Engage in combat
**Expected:** See both dodge and parry messages
**Commands:**
```
wield sword
remove offhand
kill <target>
```

### 3. Weapon + Shield (Dodge → Parry → Block)
**Setup:** Equip weapon and shield
**Test:** Engage in combat
**Expected:** See dodge, parry, AND block messages
**Commands:**
```
wield sword
hold shield
kill <target>
```
**Verify:** Shield BlockRating bonus shows in combat effectiveness

### 4. Dual Wield (Dodge → Parry × 2)
**Setup:** Equip weapons in both hands
**Test:** Engage in combat
**Expected:** See dodge and two parry attempts
**Commands:**
```
wield sword
wield dagger (offhand)
kill <target>
```

### 5. Stamina Depletion
**Setup:** Fight until stamina runs low
**Expected:** Defenses stop working when stamina < cost
**Test:**
1. Start combat with full stamina
2. Monitor stamina bar during fight
3. Once stamina < 2, dodge should fail
4. Once stamina < 4, parry should fail
5. Once stamina < 5, block should fail
**Verify:** More hits land as stamina depletes

### 6. Equipment Mid-Combat
**Setup:** Change equipment during combat
**Test:**
```
# Start with weapon+shield
kill <target>
# During combat
remove shield
# Defense sequence should change immediately
```
**Expected:** Defense messages change when equipment changes

### 7. Critical Hits
**Setup:** Fight until crit triggers
**Expected:** Critical hits should bypass all defenses
**Look for:** "*** You strike X! ***" (crit marker)
**Verify:** Crit damage shows even when defender has defenses available

### 8. Weapon ParryRating Effectiveness
**Setup:** Test different weapons
**Weapons to compare:**
- Sharp stick (ParryRating: 6)
- Practice sword (ParryRating: 12)
- Arena blade (ParryRating: 15)

**Test:** Fight same opponent with each weapon
**Expected:** Higher ParryRating weapons should parry more successfully

### 9. Shield BlockRating Effectiveness
**Setup:** Test different shields
**Shields to compare:**
- Wooden shield (BlockRating: 12)
- Iron buckler (BlockRating: 15)

**Test:** Fight same opponent with each shield
**Expected:** Higher BlockRating shields should block more successfully

### 10. Skill Progression
**Setup:** Enable skill progression in config (should already be on)
**Test:** Fight repeatedly
**Expected:** See progression messages for:
- UnarmedCombat (when dodging)
- WeaponCombat (when parrying or blocking)
**Look for:** "*** Your weapon-combat skill improves to rank X! ***"

## Balance Verification

### Attack Success Rate
**Target:** ~50-70% of attacks should land after layered defense
**Method:** Count hits vs misses over 100 combat rounds
**Command:** Fight and track outcomes
**Acceptable range:** 50-70 hits per 100 attacks

### Stamina Duration
**Target:** Stamina should last 15-30 defensive actions
**Method:**
1. Note starting stamina
2. Count defense attempts until stamina runs out
3. Stamina costs: dodge (2×0.9=1.8), parry (4×0.9=3.6), block (5×0.9=4.5)
**Expected:** Defender can survive 15-30 attacks before exhaustion

### Defense Type Distribution
**Target:** See variety of defenses in combat log
**Method:** Fight with weapon+shield, review combat log
**Expected:** Mix of dodge, parry, and block messages (not just one type)

### Equipment Impact
**Target:** Shields provide noticeable defensive advantage
**Method:** Fight same opponent twice:
1. With weapon only
2. With weapon + shield
**Expected:** Survive significantly longer with shield

### Dual Wield Risk/Reward
**Target:** Dual wield should feel riskier but offer more offense
**Method:** Compare dual wield vs weapon+shield
**Expected:**
- Dual wield: More attacks, fewer blocks, higher risk
- Weapon+shield: Fewer attacks, better defense, lower risk

## Edge Cases

### Zero Stamina Defender
**Test:** Reduce defender stamina to 0
**Expected:** All defenses skipped, every attack hits

### Fumble on Attack Roll
**Test:** Fight until fumble occurs ("!!! You fumble! !!!")
**Expected:** Fumble = automatic miss, no stamina cost for defender

### Critical Hit
**Test:** Fight until critical hit occurs ("*** Critical! ***")
**Expected:** Crit hits land regardless of defenses

### Config Multiplier Changes
**Test:** Edit config.yaml, change multipliers to 0.5
**Expected:** Defense stamina costs halved (dodge: 1, parry: 2, block: 2.5)

## Known Issues to Check

### Room Instance Saves
**Issue:** Item changes might not apply if instance saves override
**Check:** After testing, delete `_datafiles/world/dogmud/rooms.instances/` if items seem wrong

### Defense Message Spam
**Issue:** Too many defense messages might clutter combat log
**Verify:** Combat log remains readable

### Balance Issues
**Watch for:**
- Defenses too strong (nothing hits)
- Defenses too weak (everything hits)
- Stamina drains too fast
- Skill progression too slow/fast

## Performance Check
**Monitor:** Server performance during large combats
**Test:** Multiple combatants with complex defense sequences
**Expected:** No noticeable lag

## Regression Testing
**Verify these still work:**
- Basic combat damage
- Critical hit damage bonuses
- Fumble detection
- Combat messages
- XP gain from combat
- Death mechanics

## Success Criteria
✅ All equipment configurations show correct defense sequences
✅ Defense messages display with appropriate colors
✅ Stamina costs apply correctly
✅ Skill progression triggers on successful defenses
✅ ParryRating/BlockRating affect defense success rates
✅ Combat feels tactical and equipment choices matter
✅ Balance feels good (not too easy, not impossible)

## Troubleshooting

### "Defense always fails"
- Check stamina is > 0
- Verify ParryRating/BlockRating in item files loaded correctly
- Check config multipliers are set

### "No defense messages"
- Verify defender has stamina
- Check that GetDefenseSequence returns non-empty array
- Ensure combat.go changes compiled

### "Items don't have ratings"
- Restart server to reload item files
- Check item YAML files have parryrating/blockrating fields
- Verify items loaded: `@item info <itemId>`

### "Skills not progressing"
- Check config: UseSkillProgression: true
- Verify DualProgressionMode: true for actual gains
- Check skill use tracking with debug logs
