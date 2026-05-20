package messaging

// Channel discriminates the broadcast path. Audio bypasses the sight
// gate and the anonymizer; visual runs the full per-recipient
// pipeline. Both channels run style normalization, color, and wrap.
type Channel int

const (
	ChannelAudio Channel = iota
	ChannelVisual
)

// RenderInput bundles the parameters for one recipient's pipeline
// pass. The caller (Room/UserRecord Send helpers) constructs one of
// these per recipient and invokes RenderForRecipient.
//
// SightDecision is computed by the caller using the predicates in
// predicates.go BEFORE entering the pipeline; the pipeline trusts it.
// CanSeeClearly + ChannelVisual → full visual text.
// CanSeeShapes only + ChannelVisual → anonymized text.
// Neither + ChannelVisual → empty return ("don't deliver").
// ChannelAudio ignores SightDecision entirely.
type RenderInput struct {
	Category      Category
	Text          string
	Channel       Channel
	SightDecision SightDecision
	LineWidth     int
}

// SightDecision is the precomputed visibility verdict for one recipient.
type SightDecision int

const (
	SightFull   SightDecision = iota // CanSeeClearly
	SightShapes                      // !CanSeeClearly && CanSeeShapes (infrared in dark)
	SightNone                        // can't see at all
)

// RenderForRecipient runs the pipeline for one recipient and returns
// the final delivery string. Empty return means "don't deliver".
//
// Stage order:
//   1. Compose (caller-provided in.Text)
//   2. Normalize (normalize.go — wired in T8)
//   3. Sight gate (visual channel only)
//   4. Anonymize (infrared-only path; anonymize.go — wired in T6)
//   5. Color (color.go — wired in T2 alongside this stub)
//   6. Wrap (wrap.go — wired in T5)
//   7. Deliver (caller does this; pipeline returns the string)
//
// Each stub stage is a no-op until its task lands. Order is locked.
func RenderForRecipient(in RenderInput) string {
	text := in.Text

	// Stage 2: normalize (stubbed; T8 lands the implementation).
	text = normalize(in.Category, text)

	// Stage 3: sight gate (visual channel only).
	if in.Channel == ChannelVisual {
		switch in.SightDecision {
		case SightNone:
			return ""
		case SightShapes:
			// Stage 4: anonymize (stubbed; T6 lands the implementation).
			text = anonymize(text)
		}
	}

	// Stage 5: color (stubbed; T2 lands a no-op, T4 wires data).
	text = applyCategoryColor(in.Category, text)

	// Stage 6: wrap (stubbed; T5 lands the implementation).
	text = wrap(text, in.LineWidth)

	return text
}

// Stub implementations — each task replaces its stub.
// Keeping them in pipeline.go for now; T5/T6/T8 move them to their
// own files.

func normalize(cat Category, text string) string {
	return Normalize(cat, text)
}
func anonymize(text string) string {
	return Anonymize(text)
}
func applyCategoryColor(cat Category, text string) string {
	if cat == CategoryDefault || text == "" {
		return text
	}
	return `<ansi fg="` + cat.String() + `">` + text + `</ansi>`
}
func wrap(text string, maxWidth int) string {
	return WrapAnsi(text, maxWidth)
}
