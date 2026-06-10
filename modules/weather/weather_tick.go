package weather

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/engine"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// startSim initializes the simulation once a geography graph exists: load
// content, restore-or-seed state, reconcile the world's mutators to it, and
// schedule the first tick/emote. Safe to call again (no-ops when ready); a
// later successful 'weather rebuild' can start a sim that failed at boot.
// Degrades gracefully: with no graph the module logs once and stays idle
// (spec §2.3.5 / §10 "graceful degradation").
func (m *weatherModule) startSim(round uint64) {
	if m.simReady {
		return
	}
	if m.graph == nil {
		mudlog.Warn("Weather: no geography graph; simulation idle (fix the world and run 'weather rebuild')")
		return
	}
	m.simCfg = m.cfg.simConfig()
	m.loadContent()
	if !m.cfg.BuffsEnabled {
		n := engine.StripBuffs()
		mudlog.Info("Weather: buffs disabled by config", "specsStripped", n)
	}
	m.loadOrInitState(round)
	engine.Reconcile(m.state.Weather)
	m.nextTick = engine.NextTickRound(engine.TickPeriod(m.cfg.TickEveryGameHours))
	m.scheduleEmote(round)
	m.simReady = true
}

// loadContent loads climate overrides and emote tables from the module's
// embedded files. Both fail soft: defaults / silence plus a warning.
func (m *weatherModule) loadContent() {
	climate, err := content.LoadClimate(files, "files/datafiles/climate")
	if err != nil {
		mudlog.Warn("Weather: climate overrides failed to load; using defaults", "error", err)
	}
	m.climate = climate

	tables, err := content.LoadEmotes(files, "files/datafiles/emotes")
	if err != nil {
		mudlog.Warn("Weather: emote tables failed to load", "error", err)
	}
	m.tables = tables
}

// loadOrInitState restores persisted simulation state, or seeds a fresh run
// (configured Seed, else derived stably from the world's zone names).
func (m *weatherModule) loadOrInitState(round uint64) {
	if m.cfg.Persist {
		if b, err := m.plug.ReadBytes(engine.StateIdentifier); err == nil {
			if s, ok := engine.DecodeState(b); ok {
				m.state = s
				mudlog.Info("Weather: restored simulation state",
					"fronts", len(s.Fronts), "savedRound", s.Round)
				return
			}
		}
	}
	seed := m.cfg.Seed
	if seed == 0 {
		seed = sim.DeriveSeed(m.graph)
	}
	m.state = sim.NewState(seed)
	mudlog.Info("Weather: fresh simulation state", "seed", seed, "currentRound", round)
}

// tick advances the simulation one coarse step. Reconcile (rather than a bare
// diff-apply) re-asserts any mutator the specs' decayrate safety net dropped
// between ticks, so engine-side decay drift self-corrects within one tick.
func (m *weatherModule) tick(round uint64) {
	next, diff := sim.Step(m.state, m.graph, m.climate, m.simCfg, sim.Clock{Round: round})
	m.state = next
	_ = diff // per-zone changes are implied by the reconcile below
	engine.Reconcile(m.state.Weather)
	m.persistState()
	m.nextTick = engine.NextTickRound(engine.TickPeriod(m.cfg.TickEveryGameHours))
}

// persistState writes the current state to plugin storage (cheap: a few KB
// once per game hour). Also invoked from the engine's save callback.
func (m *weatherModule) persistState() {
	if !m.cfg.Persist {
		return
	}
	b, err := engine.EncodeState(m.state)
	if err != nil {
		mudlog.Error("Weather: state encode failed", "error", err)
		return
	}
	if err := m.plug.WriteBytes(engine.StateIdentifier, b); err != nil {
		mudlog.Error("Weather: state save failed", "error", err)
	}
}

// onSave is the plugins.Save() hook (autosave, shutdown, copyover).
func (m *weatherModule) onSave() {
	if m.simReady {
		m.persistState()
	}
}

// scheduleEmote picks the next ambient-emote round: the configured cadence
// jittered by ±25% so ambiance doesn't metronome.
func (m *weatherModule) scheduleEmote(round uint64) {
	every := m.cfg.EmoteEveryRounds
	delta := every
	if jitter := every / 4; jitter > 0 {
		delta += util.Rand(2*jitter+1) - jitter
	}
	if delta < 1 {
		delta = 1
	}
	m.nextEmote = round + uint64(delta)
}
