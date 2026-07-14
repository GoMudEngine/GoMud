package util

import "github.com/GoMudEngine/GoMud/internal/copyover"

type countsState struct {
	RoundCount uint64 `json:"round_count"`
	TurnCount  uint64 `json:"turn_count"`
}

type utilCopyoverContributor struct{}

func (u *utilCopyoverContributor) CopyoverName() string {
	return "util"
}

func (u *utilCopyoverContributor) CopyoverSave(enc *copyover.Encoder) error {
	return enc.WriteSection(u.CopyoverName(), countsState{
		RoundCount: roundCount,
		TurnCount:  turnCount,
	})
}

func (u *utilCopyoverContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var state countsState
	if err := dec.ReadSection(u.CopyoverName(), &state); err != nil {
		return err
	}
	roundCount = state.RoundCount
	turnCount = state.TurnCount
	return nil
}

// CopyoverContributor returns the util contributor for registration.
func CopyoverContributor() copyover.Contributor {
	return &utilCopyoverContributor{}
}
