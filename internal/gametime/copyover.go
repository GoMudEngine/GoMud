package gametime

import "github.com/GoMudEngine/GoMud/internal/copyover"

type gametimeState struct {
	DayResetOffset int `json:"day_reset_offset"`
}

type gametimeCopyoverContributor struct{}

func (g *gametimeCopyoverContributor) CopyoverName() string {
	return "gametime"
}

func (g *gametimeCopyoverContributor) CopyoverSave(enc *copyover.Encoder) error {
	return enc.WriteSection(g.CopyoverName(), gametimeState{
		DayResetOffset: dayResetOffset,
	})
}

func (g *gametimeCopyoverContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var state gametimeState
	if err := dec.ReadSection(g.CopyoverName(), &state); err != nil {
		return err
	}
	dayResetOffset = state.DayResetOffset
	clear(roundDateCache)
	return nil
}

// CopyoverContributor returns the gametime contributor for registration.
func CopyoverContributor() copyover.Contributor {
	return &gametimeCopyoverContributor{}
}
