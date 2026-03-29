package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func TestWorn_EnableAll(t *testing.T) {
	tests := []struct {
		name     string
		input    Worn
		expected Worn
	}{
		{
			name: "All slots have negative ItemId, should reset to zero value",
			input: Worn{
				Weapon:  items.Item{ItemId: -1},
				Offhand: items.Item{ItemId: -2},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: -4},
				Body:    items.Item{ItemId: -5},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: -7},
				Ring:    items.Item{ItemId: -8},
				Legs:    items.Item{ItemId: -9},
				Feet:    items.Item{ItemId: -10},
			},
			expected: Worn{
				Weapon:  items.Item{},
				Offhand: items.Item{},
				Head:    items.Item{},
				Neck:    items.Item{},
				Body:    items.Item{},
				Belt:    items.Item{},
				Gloves:  items.Item{},
				Ring:    items.Item{},
				Legs:    items.Item{},
				Feet:    items.Item{},
			},
		},
		{
			name: "All slots have positive ItemId, should remain unchanged",
			input: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
		},
		{
			name: "Mixed ItemIds, only negative should reset",
			input: Worn{
				Weapon:  items.Item{ItemId: -1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: -5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: -7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: -9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: Worn{
				Weapon:  items.Item{},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{},
				Feet:    items.Item{ItemId: 10},
			},
		},
		{
			name: "All slots zero ItemId, should remain unchanged",
			input: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: 0},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 0},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 0},
			},
			expected: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: 0},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 0},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.input
			w.EnableAll()
			assert.Equal(t, tt.expected, w)
		})
	}
}
func TestWorn_GetAllItems(t *testing.T) {
	tests := []struct {
		name     string
		worn     Worn
		expected []items.Item
	}{
		{
			name:     "All slots empty",
			worn:     Worn{},
			expected: []items.Item{},
		},
		{
			name: "All slots have positive ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 2},
				{ItemId: 3},
				{ItemId: 4},
				{ItemId: 5},
				{ItemId: 6},
				{ItemId: 7},
				{ItemId: 8},
				{ItemId: 9},
				{ItemId: 10},
			},
		},
		{
			name: "Some slots have positive ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: -1},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 3},
				{ItemId: 6},
				{ItemId: 8},
				{ItemId: 10},
			},
		},
		{
			name: "All slots zero or negative ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: -2},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: -4},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: -8},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: -10},
			},
			expected: []items.Item{},
		},
		{
			name: "Mixed positive, zero, and negative ItemIds",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: -10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 4},
				{ItemId: 7},
				{ItemId: 9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.worn.GetAllItems()
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestGetAllSlotTypes(t *testing.T) {
	expected := []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Shoulders),
		string(items.Body),
		string(items.Back),
		string(items.Belt),
		string(items.Wrist),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
		string(items.ComponentBag),
	}

	got := GetAllSlotTypes()
	assert.Equal(t, expected, got, "GetAllSlotTypes should return all slot types in correct order")
}
