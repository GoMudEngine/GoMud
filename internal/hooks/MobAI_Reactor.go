package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobai"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ─── Room adapter ────────────────────────────────────────────────────────────

// roomAdapter wraps *rooms.Room to satisfy the mobai.RoomAccessor interface.
type roomAdapter struct {
	room *rooms.Room
}

func (a *roomAdapter) GetAdjacentRoomIds() []int {
	ids := make([]int, 0, len(a.room.Exits))
	for _, ex := range a.room.Exits {
		if ex.RoomId > 0 {
			ids = append(ids, ex.RoomId)
		}
	}
	return ids
}

func (a *roomAdapter) GetPlayerIds() []int {
	return a.room.GetPlayers()
}

func (a *roomAdapter) GetMobInstanceIds() []int {
	return a.room.GetMobs()
}

func (a *roomAdapter) SendText(txt string, excludeUserIds ...int) {
	a.room.SendText(txt, excludeUserIds...)
}

// ─── Player adapter ───────────────────────────────────────────────────────────

// playerAdapter wraps *users.UserRecord to satisfy mobai.PlayerAccessor.
type playerAdapter struct {
	user *users.UserRecord
}

func (a *playerAdapter) GetUserId() int {
	return a.user.UserId
}

func (a *playerAdapter) GetHealth() int {
	return a.user.Character.Health
}

func (a *playerAdapter) GetStrengthAdj() int {
	return a.user.Character.Stats.Strength.ValueAdj
}

func (a *playerAdapter) GetDexterityAdj() int {
	return a.user.Character.Stats.Dexterity.ValueAdj
}

func (a *playerAdapter) GetWillpowerAdj() int {
	return a.user.Character.Stats.Willpower.ValueAdj
}

// ─── Context builder ──────────────────────────────────────────────────────────

// buildMobTriggerContext assembles the TriggerContext for a mob from live state.
func buildMobTriggerContext(mob *mobs.Mob) *mobai.TriggerContext {
	ctx := &mobai.TriggerContext{
		MobChar:   &mob.Character,
		HasAggro:  mob.Character.Aggro != nil,
		HasMemory: mob.CombatMemory != nil && mob.CombatMemory.Grudge,
	}

	// Check hidden state via adjective (buff 9 = Hidden permabuff)
	ctx.IsHidden = mob.Character.HasAdjective("hidden")

	// Collect active buff IDs from the buff list
	for _, b := range mob.Character.Buffs.GetBuffs() {
		ctx.ActiveBuffIds = append(ctx.ActiveBuffIds, b.BuffId)
	}

	// Resolve current aggro target
	if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				ctx.Target = u.Character
			}
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			if tm := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); tm != nil {
				ctx.Target = &tm.Character
			}
		}
	}

	// Count enemy players in the same room
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		ctx.EnemyCount = len(room.GetPlayers())
	}

	return ctx
}

// ─── Listener: HandleMobAISignal ─────────────────────────────────────────────

// HandleMobAISignal processes a MobAISignal event, evaluates tactics, and
// queues a reaction for the mob if appropriate.
func HandleMobAISignal(e events.Event) events.ListenerReturn {
	signal, ok := e.(events.MobAISignal)
	if !ok {
		return events.Continue
	}

	mob := mobs.GetInstance(signal.MobInstanceId)
	if mob == nil || mob.Character.Health <= 0 {
		return events.Continue
	}

	mudlog.Debug("MobAI", "signal", signal.SignalType, "mob", mob.Character.Name, "mobId", mob.InstanceId, "detail", signal.Detail)

	// Resolve tactics — skip mobs that have none configured
	tactics := mobai.ResolveTactics(mob.TacticPreset, mob.Tactics)
	if len(tactics) == 0 {
		mudlog.Debug("MobAI", "skip", "no tactics", "mob", mob.Character.Name)
		return events.Continue
	}

	mudlog.Debug("MobAI", "tactics_count", len(tactics), "mob", mob.Character.Name)

	bal := configs.GetBalanceConfig()
	cfg := mobai.ReactorConfig{
		Enabled:          bool(bal.MobAIEnabled),
		ReactionDelayMin: float64(bal.MobReactionDelayMin),
		ReactionDelayMax: float64(bal.MobReactionDelayMax),
		TurnsPerSecond:   configs.GetTimingConfig().TurnsPerSecond(),
	}

	ctx := buildMobTriggerContext(mob)

	sigData := mobai.SignalData{
		MobInstanceId: signal.MobInstanceId,
		SignalType:    signal.SignalType,
		Detail:        signal.Detail,
		RoomId:        signal.RoomId,
	}

	action := mobai.ProcessSignal(sigData, ctx, tactics,
		mob.GetEffectiveDiscipline(), mob.ReactionDelay,
		mob.GetLastReactionTurn(), cfg)

	if action != "" {
		mudlog.Debug("MobAI", "queued", action, "mob", mob.Character.Name, "discipline", mob.GetEffectiveDiscipline())
		mob.SetLastReactionTurn(util.GetTurnCount())
		mob.GrowDiscipline(0.01)
	} else {
		mudlog.Debug("MobAI", "no_action", mob.Character.Name, "signal", signal.SignalType)
	}

	return events.Continue
}

// ─── Listener: ProcessMobReactions ───────────────────────────────────────────

// ProcessMobReactions fires any pending mob AI reactions whose delay has
// elapsed. Called on every NewTurn tick.
func ProcessMobReactions(e events.Event) events.ListenerReturn {
	mobai.ProcessPendingReactions(
		util.GetTurnCount(),
		func(id int) mobai.MobActor {
			m := mobs.GetInstance(id)
			if m == nil {
				return nil
			}
			return m
		},
		func(roomId int) mobai.RoomAccessor {
			r := rooms.LoadRoom(roomId)
			if r == nil {
				return nil
			}
			return &roomAdapter{r}
		},
		func(id int) mobai.MobActor {
			m := mobs.GetInstance(id)
			if m == nil {
				return nil
			}
			return m
		},
		func(uid int) mobai.PlayerAccessor {
			u := users.GetByUserId(uid)
			if u == nil {
				return nil
			}
			return &playerAdapter{u}
		},
	)
	return events.Continue
}
