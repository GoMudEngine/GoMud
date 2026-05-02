package sealedcrate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v3"
)

type cratePayload struct {
	RoomId   int          `yaml:"roomid"`
	Capacity int          `yaml:"capacity"`
	Items    []items.Item `yaml:"items,omitempty"`
}

// SaveTo writes a crate's state to the given path. The parent
// directory will be created if missing.
func SaveTo(path string, c *Crate) error {
	payload := cratePayload{
		RoomId:   c.RoomId(),
		Capacity: c.Capacity(),
		Items:    c.Snapshot(),
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sealedcrate.SaveTo: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("sealedcrate.SaveTo: mkdir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("sealedcrate.SaveTo: write: %w", err)
	}
	return nil
}

// LoadFrom reads a crate from the given path. Returns nil + error
// if the file is missing or fails to unmarshal.
func LoadFrom(path string) (*Crate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload cratePayload
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("sealedcrate.LoadFrom: unmarshal: %w", err)
	}
	c := New(payload.RoomId, payload.Capacity)
	c.SetItemsForLoad(payload.Items)
	return c, nil
}
