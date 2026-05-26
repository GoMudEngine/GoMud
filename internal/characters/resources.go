package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/statmods"
)

func (c *Character) DeductActionPoints(amount int) bool {

	if c.ActionPoints < amount {
		return false
	}
	c.ActionPoints -= amount
	if c.ActionPoints < 0 {
		c.ActionPoints = 0
	}
	return true
}

// DeductStamina attempts to deduct the specified amount of stamina.
// Returns false if the character doesn't have enough stamina.
func (c *Character) DeductStamina(amount int) bool {
	if c.Stamina < amount {
		return false
	}
	c.Stamina -= amount
	if c.Stamina < 0 {
		c.Stamina = 0
	}
	return true
}

// GetMovementStaminaCost calculates the stamina cost for movement based on
// terrain difficulty and encumbrance.
// terrainMultiplier: 1.0 = normal terrain, 2.0 = rough terrain, etc.
// Returns stamina cost (2-20 stamina range).
func (c *Character) GetMovementStaminaCost(terrainMultiplier float64) int {
	b := configs.GetBalanceConfig()
	baseCost := float64(b.MovementBaseStaminaCost)
	maxCost := float64(b.MovementMaxStaminaCost)

	// Apply terrain multiplier
	cost := baseCost * terrainMultiplier

	// Calculate encumbrance multiplier based on weight
	encumbranceMultiplier := 1.0
	carriedWeight := c.GetCarriedWeight()
	capacity := c.CarryCapacity()

	if carriedWeight > capacity {
		// Overencumbered: scale from 1.0 to 5.0 based on how much over capacity
		overAmount := carriedWeight - capacity
		overRatio := overAmount / capacity
		// Cap at 5x multiplier when carrying 2x capacity
		encumbranceMultiplier = 1.0 + math.Min(overRatio*4.0, 4.0)
	}

	// Apply encumbrance multiplier
	cost *= encumbranceMultiplier

	// Phase 24.2: Apply mutation movement speed modifier (Hasted, Extra Legs, etc.)
	if moveMod := mutations.GetMovementSpeedModifier(c.Mutations); moveMod != 0 {
		cost *= (1.0 - moveMod) // positive moveMod = faster = less cost
	}

	// Sneaking costs extra stamina — moving carefully is harder.
	// HiddenMoveStaminaMultiplier (default 3.0) stacks multiplicatively
	// with encumbrance: over-encumbered hidden movement is brutal (5×3 = 15× cost).
	if c.IsHidden() {
		cost *= float64(b.HiddenMoveStaminaMultiplier)
	}

	// Cap at maximum stamina cost
	if cost > maxCost {
		cost = maxCost
	}

	// Minimum 1 stamina
	if cost < 1.0 {
		cost = 1.0
	}

	return int(math.Ceil(cost))
}

// GetAttackStaminaCost calculates the stamina cost for making an attack.
// Cost is based on weapon type (or unarmed if no weapon).
func (c *Character) GetAttackStaminaCost() int {
	// Check main hand weapon
	if c.Equipment.Weapon.ItemId > 0 {
		weaponSpec := c.Equipment.Weapon.GetSpec()
		return weaponSpec.GetAttackStaminaCost()
	}

	// Check offhand weapon (dual wielding)
	if c.Equipment.Offhand.ItemId > 0 {
		offhandSpec := c.Equipment.Offhand.GetSpec()
		return offhandSpec.GetAttackStaminaCost()
	}

	// Unarmed combat costs less stamina
	return int(configs.GetBalanceConfig().UnarmedAttackStaminaCost)
}

// DeductAttackStamina deducts stamina for an attack and returns the actual cost deducted.
// If character doesn't have enough stamina, deducts what they have and returns that amount.
func (c *Character) DeductAttackStamina() int {
	cost := c.GetAttackStaminaCost()

	if c.Stamina >= cost {
		c.Stamina -= cost
		return cost
	}

	// Insufficient stamina - deduct what we have
	actualCost := c.Stamina
	c.Stamina = 0
	return actualCost
}

// GetDefenseStaminaCost returns stamina cost for a defense type (Stage 7.1)
func (c *Character) GetDefenseStaminaCost(defenseType string) int {
	bal := configs.GetBalanceConfig()

	baseCost := 0
	multiplier := 1.0

	switch defenseType {
	case DefenseDodge:
		baseCost = 2
		multiplier = float64(bal.DodgeMultiplier)
	case DefenseParry:
		baseCost = 4
		multiplier = float64(bal.ParryMultiplier)
	case DefenseBlock:
		baseCost = 5
		multiplier = float64(bal.BlockMultiplier)
	default:
		return 0
	}

	cost := int(float64(baseCost) * multiplier)
	if cost < 1 {
		cost = 1 // minimum 1 stamina
	}
	return cost
}

// DeductDefenseStamina deducts stamina for a defense and returns true if successful (Stage 7.1)
func (c *Character) DeductDefenseStamina(defenseType string) bool {
	cost := c.GetDefenseStaminaCost(defenseType)
	if c.Stamina >= cost {
		c.Stamina -= cost
		return true
	}
	return false
}

func (c *Character) ApplyHealthChange(healthChange int) int {
	oldHealth := c.Health
	newHealth := c.Health + healthChange
	if newHealth < 0 {
		// Any drop below 0 means dead; cancel combat-scoped buffs. Death
		// itself is processed by the per-round hooks (NewRound_DoCombat +
		// NewRound_AutoHeal); this function only applies the raw change.
		c.CancelCombatBuffs()
	} else if newHealth > c.HealthMax.Value {
		newHealth = c.HealthMax.Value
	}

	c.Health = newHealth

	return newHealth - oldHealth
}

func (c *Character) Heal(hp int) int {
	startHP := c.Health

	c.Health += hp
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}

	return c.Health - startHP
}

func (c *Character) HealthPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerHealthRegenPct)
	if c.IsMob {
		pct = float64(b.MobHealthRegenPct)
	}
	// StatMod reinterpreted as percentage bonus (e.g. 5 → +5%)
	pct += float64(c.StatMod(string(statmods.HealthRecovery))) / 100.0
	base := int(pct * float64(c.HealthMax.Value))
	if base < 1 {
		base = 1
	}
	// Chunk 3.3: 5× regen while sleeping.
	if c.HasBuffFlag(buffs.Sleeping) {
		if mult := float64(b.SleepRegenMultiplier); mult > 0 {
			base = int(float64(base) * mult)
		}
	}
	return base
}

func (c *Character) StaminaPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerStaminaRegenPct)
	if c.IsMob {
		pct = float64(b.MobStaminaRegenPct)
	}
	pct += float64(c.StatMod(string(statmods.StaminaRecovery))) / 100.0
	base := int(pct * float64(c.StaminaMax.Value))
	if base < 1 {
		base = 1
	}
	// Apply stamina_regen_multiplier mutations
	if mult := mutations.GetStaminaRegenMultiplier(c.Mutations); mult != 0 {
		base = int(float64(base) * (1.0 + mult))
		if base < 1 {
			base = 1
		}
	}
	// Chunk 3.3: 5× regen while sleeping (composes on top of mutation modifier).
	if c.HasBuffFlag(buffs.Sleeping) {
		if mult := float64(b.SleepRegenMultiplier); mult > 0 {
			base = int(float64(base) * mult)
		}
	}
	return base
}

func (c *Character) ConvictionPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerConvictionRegenPct)
	if c.IsMob {
		pct = float64(b.MobConvictionRegenPct)
	}
	pct += float64(c.StatMod(string(statmods.ConvictionRecovery))) / 100.0
	base := int(pct * float64(c.ConvictionMax.Value))
	if base < 1 {
		base = 1
	}
	// Chunk 3.3: 5× regen while sleeping.
	if c.HasBuffFlag(buffs.Sleeping) {
		if mult := float64(b.SleepRegenMultiplier); mult > 0 {
			base = int(float64(base) * mult)
		}
	}
	return base
}

// GetToxicityMax returns the maximum toxicity this character can handle.
// Formula: BaseMax + Vitality / VitalityScale
func (c *Character) GetToxicityMax() float64 {
	bal := configs.GetBalanceConfig()
	return float64(bal.ToxicityBaseMax) + float64(c.Stats.Vitality.ValueAdj)/float64(bal.ToxicityVitalityScale)
}

// AddToxicity attempts to add toxicity. Returns false if it would exceed max.
func (c *Character) AddToxicity(amount float64) bool {
	if c.Toxicity+amount > c.GetToxicityMax() {
		return false
	}
	c.Toxicity += amount
	return true
}

// GetToxicityPenalties returns stat multipliers based on toxicity threshold.
// Returns (regenMult, perceptionMult, dexterityMult) where 1.0 = no penalty.
func (c *Character) GetToxicityPenalties() (float64, float64, float64) {
	max := c.GetToxicityMax()
	if max <= 0 {
		return 1.0, 1.0, 1.0
	}
	ratio := c.Toxicity / max

	switch {
	case ratio >= 0.90:
		return 0.60, 0.80, 0.90 // -40% regen, -20% Per, -10% Dex
	case ratio >= 0.75:
		return 0.80, 0.90, 0.90 // -20% regen, -10% Per, -10% Dex
	case ratio >= 0.50:
		return 0.90, 0.90, 1.0  // -10% regen, -10% Per
	default:
		return 1.0, 1.0, 1.0   // no penalty
	}
}

// Where 1000 = a full round
func (c *Character) MovementCost() int {
	modifier := 3                                    // by default they should be able to move 3 times per round.
	modifier += int(c.Stats.Dexterity.ValueAdj / 15) // Every 15 dexterity, get an extra movement
	return int(1000 / modifier)
}
