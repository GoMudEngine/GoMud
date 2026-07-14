//go:build windows

package connections

import "github.com/GoMudEngine/GoMud/internal/copyover"

func (c *connectionsCopyoverContributor) CopyoverSave(enc *copyover.Encoder) error {
	return enc.WriteSection(c.CopyoverName(), connectionsState{ConnectCounter: connectCounter})
}

func (c *connectionsCopyoverContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var state connectionsState
	return dec.ReadSection(c.CopyoverName(), &state)
}
