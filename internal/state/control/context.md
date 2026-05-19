# Package: internal/state/control

Per-character ControlLevel state machine for grapple dominance tracking.
Chunk 4b-fixup-2 (2026-05-18).

## States

- **Controlling** (stable): you have positional dominance.
- **LosingControl** (transient, auto-resolves same-tick): traversing
  Controlling↔Neutral boundary; fires gradient flavor messaging.
- **Neutral** (stable): neither side dominant; symmetric or in-flux.
- **BecomingControlled** (transient, auto-resolves same-tick): traversing
  Neutral↔Controlled boundary; fires gradient flavor messaging.
- **Controlled** (stable): you're dominated.

## Pattern

Mirrors `internal/state/awareness/` (the `Hidden → Revealing → Visible`
same-tick traversal). Transient states are real FSM states but
auto-resolve to the target stable state in the same call to
`TransitionTo*`. Boundary-cross callbacks fire during the brief in-state
moment.

Direct jumps cross both boundaries in sequence:
- `Controlling → Controlled`: LosingControl → Neutral →
  BecomingControlled → Controlled
- `Controlled → Controlling`: BecomingControlled → Neutral →
  LosingControl → Controlling

## Triggers

- `TriggerGrappleEnter`: initial setup at pair transition.
- `TriggerDriftWin`: drift roll favored this side; shift toward
  Controlling.
- `TriggerDriftLoss`: drift roll opposed this side; shift toward
  Controlled.
- `TriggerGrappleExit`: grapple breaks; reset to Neutral.

## Boundary-cross callbacks

Registered via `RegisterBoundaryCrossCallback`. Receives
`(self, transient, from, to, reason)`. Fires once per boundary crossed
during a same-tick transition. Used by
`internal/hooks/Position_GrappleTick.go` to emit gradient messaging
via `internal/grapplemessaging`.

## Initial states per position (set by `internal/state/position/pair.go`)

- Symmetric (Clinch, HalfGuard, Turtle): both sides Neutral.
- Asymmetric: controller arg → Controlling; controlled arg → Controlled.

## Validation

Per-pair invariant (in `internal/state/position/validation.go`):
- Not both at Controlling.
- Not both at Controlled.

## Not currently used for

- Position-change outcome gating. Position changes still come from the
  z-bucket outcome resolver (`internal/state/position/outcomes.go`,
  chunk 4b-fixup).
- Sub gate magnitude. Sub windows still gate on `|z| >= 1.5`
  (chunk 4d). ControlLevel state DOES gate sub eligibility per-position
  (must be Controlling for top subs; Controlled for bottom subs).
