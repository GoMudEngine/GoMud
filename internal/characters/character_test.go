package characters

import (
	"reflect"
	"strings"
	"testing"

	"maps"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
)

func TestCharacter_CanDualWield(t *testing.T) {
	tests := []struct {
		name       string
		skillLevel int
		want       bool
	}{
		{
			name:       "No DualWield skill",
			skillLevel: 0,
			want:       false,
		},
		{
			name:       "DualWield skill at level 1",
			skillLevel: 1,
			want:       true,
		},
		{
			name:       "DualWield skill at level 2",
			skillLevel: 2,
			want:       true,
		},
		{
			name:       "DualWield skill at level 4",
			skillLevel: 4,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Skills[string(skills.WeaponCombat)] = tt.skillLevel
			got := c.CanDualWield()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_GetMiscDataKeys(t *testing.T) {
	tests := []struct {
		name        string
		miscData    map[string]any
		prefixMatch []string
		want        []string
	}{
		{
			name:        "No misc data, no prefix",
			miscData:    nil,
			prefixMatch: nil,
			want:        []string{},
		},
		{
			name:        "No misc data, with prefix",
			miscData:    nil,
			prefixMatch: []string{"foo"},
			want:        []string{},
		},
		{
			name: "Misc data, no prefix",
			miscData: map[string]any{
				"foo1": 1,
				"bar2": 2,
				"baz3": 3,
			},
			prefixMatch: nil,
			want:        []string{"foo1", "bar2", "baz3"},
		},
		{
			name: "Misc data, prefix matches one key",
			miscData: map[string]any{
				"foo1": 1,
				"bar2": 2,
				"baz3": 3,
			},
			prefixMatch: []string{"foo"},
			want:        []string{"1"},
		},
		{
			name: "Misc data, prefix matches multiple keys",
			miscData: map[string]any{
				"foo1": 1,
				"foo2": 2,
				"bar3": 3,
			},
			prefixMatch: []string{"foo"},
			want:        []string{"1", "2"},
		},
		{
			name: "Misc data, prefix matches no keys",
			miscData: map[string]any{
				"foo1": 1,
				"bar2": 2,
			},
			prefixMatch: []string{"baz"},
			want:        []string{},
		},
		{
			name: "Misc data, multiple prefixes",
			miscData: map[string]any{
				"foo1": 1,
				"bar2": 2,
				"baz3": 3,
				"foo4": 4,
				"bar5": 5,
			},
			prefixMatch: []string{"foo", "bar"},
			want:        []string{"1", "4", "2", "5"},
		},
		{
			name: "Misc data, prefix is full key",
			miscData: map[string]any{
				"foo":    1,
				"foobar": 2,
			},
			prefixMatch: []string{"foo"},
			want:        []string{"", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.MiscData = tt.miscData
			got := c.GetMiscDataKeys(tt.prefixMatch...)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
func TestCharacter_CarryCapacity(t *testing.T) {
	// CarryCapacity() = float64(strengthAdj) * 3.0
	tests := []struct {
		name        string
		strengthAdj int
		expectedCap float64
	}{
		{"Strength 0", 0, 0.0},
		{"Strength 2", 2, 6.0},
		{"Strength 10", 10, 30.0},
		{"Strength 100", 100, 300.0},
		{"Negative Strength", -3, -9.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Stats.Strength.ValueAdj = tt.strengthAdj
			got := c.CarryCapacity()
			assert.Equal(t, tt.expectedCap, got)
		})
	}
}
func TestCharacter_GetDescription(t *testing.T) {
	// Test when Description does not start with "h:"
	t.Run("Plain description", func(t *testing.T) {
		c := New()
		c.Description = "A brave adventurer."
		got := c.GetDescription()
		assert.Equal(t, "A brave adventurer.", got)
	})

	// Test when Description starts with "h:" and hash is in descriptionCache
	t.Run("Description is hash and found in cache", func(t *testing.T) {
		c := New()
		hash := "abc123"
		want := "A mysterious stranger."
		descriptionCache[hash] = want
		c.Description = "h:" + hash
		got := c.GetDescription()
		assert.Equal(t, want, got)
	})

	// Test when Description starts with "h:" and hash is not in descriptionCache
	t.Run("Description is hash and not found in cache", func(t *testing.T) {
		c := New()
		hash := "notfound"
		delete(descriptionCache, hash)
		c.Description = "h:" + hash
		got := c.GetDescription()
		assert.Equal(t, "", got)
	})

	// Test when Description is exactly "h:" (empty hash)
	t.Run("Description is h: with empty hash", func(t *testing.T) {
		c := New()
		c.Description = "h:"
		got := c.GetDescription()
		assert.Equal(t, descriptionCache[""], got)
	})
}
func TestCharacter_DeductActionPoints(t *testing.T) {
	tests := []struct {
		name         string
		startAP      int
		deductAmount int
		wantResult   bool
		wantFinalAP  int
	}{
		{
			name:         "Enough action points",
			startAP:      10,
			deductAmount: 5,
			wantResult:   true,
			wantFinalAP:  5,
		},
		{
			name:         "Exact action points",
			startAP:      5,
			deductAmount: 5,
			wantResult:   true,
			wantFinalAP:  0,
		},
		{
			name:         "Not enough action points",
			startAP:      3,
			deductAmount: 5,
			wantResult:   false,
			wantFinalAP:  3,
		},
		{
			name:         "Deduct more than available, negative AP clamp",
			startAP:      2,
			deductAmount: 3,
			wantResult:   false,
			wantFinalAP:  2,
		},
		{
			name:         "Deduct zero action points",
			startAP:      7,
			deductAmount: 0,
			wantResult:   true,
			wantFinalAP:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.ActionPoints = tt.startAP
			got := c.DeductActionPoints(tt.deductAmount)
			assert.Equal(t, tt.wantResult, got)
			assert.Equal(t, tt.wantFinalAP, c.ActionPoints)
		})
	}
}
func TestCharacter_SetUserId(t *testing.T) {
	tests := []struct {
		name   string
		userId int
	}{
		{
			name:   "Set positive userId",
			userId: 42,
		},
		{
			name:   "Set zero userId",
			userId: 0,
		},
		{
			name:   "Set negative userId",
			userId: -7,
		},
		{
			name:   "Set large userId",
			userId: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.SetUserId(tt.userId)
			// userId is unexported, so we need to use reflection to check it
			userIdField := reflect.ValueOf(c).Elem().FieldByName("userId")
			assert.True(t, userIdField.IsValid(), "userId field should exist")
			assert.Equal(t, tt.userId, int(userIdField.Int()))
		})
	}
}
func TestCharacter_SetMiscData(t *testing.T) {
	tests := []struct {
		name         string
		initialData  map[string]any
		key          string
		value        any
		expectExists bool
		expectValue  any
	}{
		{
			name:         "Set new key-value pair",
			initialData:  nil,
			key:          "foo",
			value:        123,
			expectExists: true,
			expectValue:  123,
		},
		{
			name:         "Overwrite existing key",
			initialData:  map[string]any{"bar": "baz"},
			key:          "bar",
			value:        "qux",
			expectExists: true,
			expectValue:  "qux",
		},
		{
			name:         "Delete existing key by setting value to nil",
			initialData:  map[string]any{"del": 42},
			key:          "del",
			value:        nil,
			expectExists: false,
			expectValue:  nil,
		},
		{
			name:         "Delete non-existing key by setting value to nil",
			initialData:  map[string]any{"other": 1},
			key:          "missing",
			value:        nil,
			expectExists: false,
			expectValue:  nil,
		},
		{
			name:         "Set key with value zero",
			initialData:  nil,
			key:          "zero",
			value:        0,
			expectExists: true,
			expectValue:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialData != nil {
				c.MiscData = make(map[string]any)
				for k, v := range tt.initialData {
					c.MiscData[k] = v
				}
			}
			c.SetMiscData(tt.key, tt.value)
			val, ok := c.MiscData[tt.key]
			assert.Equal(t, tt.expectExists, ok)
			assert.Equal(t, tt.expectValue, val)
		})
	}
}
func TestCharacter_GetMiscData(t *testing.T) {
	tests := []struct {
		name        string
		initialData map[string]any
		key         string
		want        any
	}{
		{
			name:        "Get existing key",
			initialData: map[string]any{"foo": 123, "bar": "baz"},
			key:         "foo",
			want:        123,
		},
		{
			name:        "Get another existing key",
			initialData: map[string]any{"foo": 123, "bar": "baz"},
			key:         "bar",
			want:        "baz",
		},
		{
			name:        "Get non-existing key returns nil",
			initialData: map[string]any{"foo": 123},
			key:         "missing",
			want:        nil,
		},
		{
			name:        "Get key from nil map initializes map and returns nil",
			initialData: nil,
			key:         "anything",
			want:        nil,
		},
		{
			name:        "Get key with value zero",
			initialData: map[string]any{"zero": 0},
			key:         "zero",
			want:        0,
		},
		{
			name:        "Get key with value false",
			initialData: map[string]any{"flag": false},
			key:         "flag",
			want:        false,
		},
		{
			name:        "Get key with value nil explicitly set",
			initialData: map[string]any{"nilval": nil},
			key:         "nilval",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialData != nil {
				c.MiscData = make(map[string]any)
				for k, v := range tt.initialData {
					c.MiscData[k] = v
				}
			} else {
				c.MiscData = nil
			}
			got := c.GetMiscData(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_KeyCount(t *testing.T) {
	tests := []struct {
		name    string
		keyRing map[string]string
		want    int
	}{
		{
			name:    "Nil KeyRing",
			keyRing: nil,
			want:    0,
		},
		{
			name:    "Empty KeyRing",
			keyRing: map[string]string{},
			want:    0,
		},
		{
			name:    "One key in KeyRing",
			keyRing: map[string]string{"lock1": "SEQ1"},
			want:    1,
		},
		{
			name:    "Multiple keys in KeyRing",
			keyRing: map[string]string{"lock1": "SEQ1", "lock2": "SEQ2", "lock3": "SEQ3"},
			want:    3,
		},
		{
			name:    "KeyRing with empty string key",
			keyRing: map[string]string{"": "SEQ"},
			want:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.KeyRing = tt.keyRing
			got := c.KeyCount()
			assert.Equal(t, tt.want, got)
			// Also check that KeyRing is not nil after call
			assert.NotNil(t, c.KeyRing)
		})
	}
}
func TestCharacter_GetKey(t *testing.T) {
	tests := []struct {
		name      string
		keyRing   map[string]string
		lockId    string
		wantValue string
	}{
		{
			name:      "Nil KeyRing returns empty string",
			keyRing:   nil,
			lockId:    "lock1",
			wantValue: "",
		},
		{
			name:      "Empty KeyRing returns empty string",
			keyRing:   map[string]string{},
			lockId:    "lock1",
			wantValue: "",
		},
		{
			name:      "Key exists with exact case",
			keyRing:   map[string]string{"lock1": "SEQ1"},
			lockId:    "lock1",
			wantValue: "SEQ1",
		},
		{
			name:      "Key exists with different case",
			keyRing:   map[string]string{"lock1": "SEQ1"},
			lockId:    "LOCK1",
			wantValue: "SEQ1",
		},
		{
			name:      "Multiple keys, get correct one",
			keyRing:   map[string]string{"lock1": "SEQ1", "lock2": "SEQ2"},
			lockId:    "lock2",
			wantValue: "SEQ2",
		},
		{
			name:      "Key does not exist returns empty string",
			keyRing:   map[string]string{"lock1": "SEQ1"},
			lockId:    "lock3",
			wantValue: "",
		},
		{
			name:      "Key exists with empty string value",
			keyRing:   map[string]string{"lock1": ""},
			lockId:    "lock1",
			wantValue: "",
		},
		{
			name:      "Key exists with empty string key",
			keyRing:   map[string]string{"": "EMPTY"},
			lockId:    "",
			wantValue: "EMPTY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.KeyRing = tt.keyRing
			got := c.GetKey(tt.lockId)
			assert.Equal(t, tt.wantValue, got)
			assert.NotNil(t, c.KeyRing)
		})
	}
}
func TestCharacter_SetKey(t *testing.T) {
	tests := []struct {
		name         string
		initialKeys  map[string]string
		lockId       string
		sequence     string
		expectKey    string
		expectExists bool
	}{
		{
			name:         "Set new key with non-empty sequence",
			initialKeys:  nil,
			lockId:       "LockA",
			sequence:     "abc123",
			expectKey:    "ABC123",
			expectExists: true,
		},
		{
			name:         "Set new key with empty sequence (should not exist)",
			initialKeys:  nil,
			lockId:       "LockB",
			sequence:     "",
			expectKey:    "",
			expectExists: false,
		},
		{
			name:         "Overwrite existing key with new sequence",
			initialKeys:  map[string]string{"lockc": "OLDSEQ"},
			lockId:       "LockC",
			sequence:     "newseq",
			expectKey:    "NEWSEQ",
			expectExists: true,
		},
		{
			name:         "Delete existing key by setting empty sequence",
			initialKeys:  map[string]string{"lockd": "SOMETHING"},
			lockId:       "LockD",
			sequence:     "",
			expectKey:    "",
			expectExists: false,
		},
		{
			name:         "Set key with mixed case lockId, should store as lower",
			initialKeys:  nil,
			lockId:       "MiXeDcAsE",
			sequence:     "seq",
			expectKey:    "SEQ",
			expectExists: true,
		},
		{
			name:         "Set key with empty lockId",
			initialKeys:  nil,
			lockId:       "",
			sequence:     "emptykey",
			expectKey:    "EMPTYKEY",
			expectExists: true,
		},
		{
			name:         "Delete non-existing key with empty sequence",
			initialKeys:  map[string]string{"other": "VAL"},
			lockId:       "notfound",
			sequence:     "",
			expectKey:    "",
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialKeys != nil {
				c.KeyRing = make(map[string]string)
				for k, v := range tt.initialKeys {
					c.KeyRing[k] = v
				}
			} else {
				c.KeyRing = nil
			}
			c.SetKey(tt.lockId, tt.sequence)
			got, ok := c.KeyRing[strings.ToLower(tt.lockId)]
			assert.Equal(t, tt.expectExists, ok)
			assert.Equal(t, tt.expectKey, got)
		})
	}
}

func TestCharacter_GetSpells(t *testing.T) {
	tests := []struct {
		name      string
		spellBook map[string]int
		want      map[string]int
	}{
		{
			name:      "Nil SpellBook returns empty map",
			spellBook: nil,
			want:      map[string]int{},
		},
		{
			name:      "Empty SpellBook returns empty map",
			spellBook: map[string]int{},
			want:      map[string]int{},
		},
		{
			name:      "Single spell in SpellBook",
			spellBook: map[string]int{"fireball": 3},
			want:      map[string]int{"fireball": 3},
		},
		{
			name:      "Multiple spells in SpellBook",
			spellBook: map[string]int{"fireball": 3, "heal": 5, "ice": 0},
			want:      map[string]int{"fireball": 3, "heal": 5, "ice": 0},
		},
		{
			name:      "Negative value in SpellBook",
			spellBook: map[string]int{"curse": -2},
			want:      map[string]int{"curse": -2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.spellBook != nil {
				c.SpellBook = make(map[string]int)
				for k, v := range tt.spellBook {
					c.SpellBook[k] = v
				}
			} else {
				c.SpellBook = nil
			}
			got := c.GetSpells()
			assert.Equal(t, tt.want, got)
			// Ensure returned map is a copy, not the same reference
			if c.SpellBook != nil {
				got["newspell"] = 99
				_, exists := c.SpellBook["newspell"]
				assert.False(t, exists, "GetSpells should return a copy, not the original map")
			}
		})
	}
}
func TestCharacter_HasSpell(t *testing.T) {
	tests := []struct {
		name      string
		spellBook map[string]int
		spellName string
		want      bool
	}{
		{
			name:      "Nil SpellBook returns false",
			spellBook: nil,
			spellName: "fireball",
			want:      false,
		},
		{
			name:      "Empty SpellBook returns false",
			spellBook: map[string]int{},
			spellName: "fireball",
			want:      false,
		},
		{
			name:      "Spell exists with positive value",
			spellBook: map[string]int{"fireball": 2},
			spellName: "fireball",
			want:      true,
		},
		{
			name:      "Spell exists with value 1",
			spellBook: map[string]int{"heal": 1},
			spellName: "heal",
			want:      true,
		},
		{
			name:      "Spell exists with value 0",
			spellBook: map[string]int{"ice": 0},
			spellName: "ice",
			want:      false,
		},
		{
			name:      "Spell exists with negative value",
			spellBook: map[string]int{"curse": -3},
			spellName: "curse",
			want:      false,
		},
		{
			name:      "Spell does not exist",
			spellBook: map[string]int{"fireball": 2},
			spellName: "heal",
			want:      false,
		},
		{
			name:      "Multiple spells, check one present",
			spellBook: map[string]int{"fireball": 2, "heal": 1, "ice": 0},
			spellName: "heal",
			want:      true,
		},
		{
			name:      "Multiple spells, check one absent",
			spellBook: map[string]int{"fireball": 2, "heal": 1},
			spellName: "ice",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.spellBook != nil {
				c.SpellBook = make(map[string]int)
				for k, v := range tt.spellBook {
					c.SpellBook[k] = v
				}
			} else {
				c.SpellBook = nil
			}
			got := c.HasSpell(tt.spellName)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_DisableSpell(t *testing.T) {
	tests := []struct {
		name        string
		initialBook map[string]int
		spellName   string
		wantChanged bool
		wantValue   int
		wantExists  bool
	}{
		{
			name:        "Disable spell with positive value",
			initialBook: map[string]int{"fireball": 3},
			spellName:   "fireball",
			wantChanged: false,
			wantValue:   -3,
			wantExists:  true,
		},
		{
			name:        "Disable spell with value 1",
			initialBook: map[string]int{"heal": 1},
			spellName:   "heal",
			wantChanged: false,
			wantValue:   -1,
			wantExists:  true,
		},
		{
			name:        "Disable spell with value 0 (should not change)",
			initialBook: map[string]int{"ice": 0},
			spellName:   "ice",
			wantChanged: false,
			wantValue:   0,
			wantExists:  true,
		},
		{
			name:        "Disable spell with negative value (should not change)",
			initialBook: map[string]int{"curse": -2},
			spellName:   "curse",
			wantChanged: false,
			wantValue:   -2,
			wantExists:  true,
		},
		{
			name:        "Disable spell not in SpellBook (should not add)",
			initialBook: map[string]int{"fireball": 2},
			spellName:   "missing",
			wantChanged: false,
			wantValue:   0,
			wantExists:  false,
		},
		{
			name:        "Disable spell with value 100",
			initialBook: map[string]int{"bigspell": 100},
			spellName:   "bigspell",
			wantChanged: false,
			wantValue:   -100,
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialBook != nil {
				c.SpellBook = make(map[string]int)
				for k, v := range tt.initialBook {
					c.SpellBook[k] = v
				}
			} else {
				c.SpellBook = nil
			}
			_ = c.DisableSpell(tt.spellName)
			val, exists := c.SpellBook[tt.spellName]
			assert.Equal(t, tt.wantExists, exists)
			if exists {
				assert.Equal(t, tt.wantValue, val)
			}
		})
	}
}
func TestCharacter_EnableSpell(t *testing.T) {
	tests := []struct {
		name        string
		initialBook map[string]int
		spellName   string
		wantChanged bool
		wantValue   int
		wantExists  bool
	}{
		{
			name:        "Enable spell with negative value",
			initialBook: map[string]int{"fireball": -3},
			spellName:   "fireball",
			wantChanged: false,
			wantValue:   3,
			wantExists:  true,
		},
		{
			name:        "Enable spell with value -1",
			initialBook: map[string]int{"heal": -1},
			spellName:   "heal",
			wantChanged: false,
			wantValue:   1,
			wantExists:  true,
		},
		{
			name:        "Enable spell with value 0 (should not change)",
			initialBook: map[string]int{"ice": 0},
			spellName:   "ice",
			wantChanged: false,
			wantValue:   0,
			wantExists:  true,
		},
		{
			name:        "Enable spell with positive value (should not change)",
			initialBook: map[string]int{"bless": 2},
			spellName:   "bless",
			wantChanged: false,
			wantValue:   2,
			wantExists:  true,
		},
		{
			name:        "Enable spell not in SpellBook (should not add)",
			initialBook: map[string]int{"fireball": 2},
			spellName:   "missing",
			wantChanged: false,
			wantValue:   0,
			wantExists:  false,
		},
		{
			name:        "Enable spell with value -100",
			initialBook: map[string]int{"bigspell": -100},
			spellName:   "bigspell",
			wantChanged: false,
			wantValue:   100,
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialBook != nil {
				c.SpellBook = make(map[string]int)
				for k, v := range tt.initialBook {
					c.SpellBook[k] = v
				}
			} else {
				c.SpellBook = nil
			}
			_ = c.EnableSpell(tt.spellName)
			val, exists := c.SpellBook[tt.spellName]
			assert.Equal(t, tt.wantExists, exists)
			if exists {
				assert.Equal(t, tt.wantValue, val)
			}
		})
	}
}
func TestCharacter_TrackSpellCast(t *testing.T) {
	tests := []struct {
		name        string
		initialBook map[string]int
		spellName   string
		wantChanged bool
		wantValue   int
		wantExists  bool
	}{
		{
			name:        "Spell exists with positive value",
			initialBook: map[string]int{"fireball": 2},
			spellName:   "fireball",
			wantChanged: true,
			wantValue:   3,
			wantExists:  true,
		},
		{
			name:        "Spell exists with value 1",
			initialBook: map[string]int{"heal": 1},
			spellName:   "heal",
			wantChanged: true,
			wantValue:   2,
			wantExists:  true,
		},
		{
			name:        "Spell exists with value 0 (should not change)",
			initialBook: map[string]int{"ice": 0},
			spellName:   "ice",
			wantChanged: false,
			wantValue:   0,
			wantExists:  true,
		},
		{
			name:        "Spell exists with negative value (should not change)",
			initialBook: map[string]int{"curse": -3},
			spellName:   "curse",
			wantChanged: false,
			wantValue:   -3,
			wantExists:  true,
		},
		{
			name:        "Spell does not exist (should not add)",
			initialBook: map[string]int{"fireball": 2},
			spellName:   "missing",
			wantChanged: false,
			wantValue:   0,
			wantExists:  false,
		},
		{
			name:        "Nil SpellBook (should not panic, should not add)",
			initialBook: nil,
			spellName:   "anything",
			wantChanged: false,
			wantValue:   0,
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initialBook != nil {
				c.SpellBook = make(map[string]int)
				for k, v := range tt.initialBook {
					c.SpellBook[k] = v
				}
			} else {
				c.SpellBook = nil
			}
			_ = c.TrackSpellCast(tt.spellName)
			val, exists := c.SpellBook[tt.spellName]
			assert.Equal(t, tt.wantExists, exists)
			if exists {
				assert.Equal(t, tt.wantValue, val)
			}
		})
	}
}
func TestCharacter_LearnSpell(t *testing.T) {
	tests := []struct {
		name        string
		initialBook map[string]int
		spellName   string
		wantChanged bool
		wantValue   int
		wantExists  bool
	}{
		{
			name:        "Learn new spell when SpellBook is empty",
			initialBook: map[string]int{},
			spellName:   "heal",
			wantChanged: true,
			wantValue:   1,
			wantExists:  true,
		},
		{
			name:        "Learn spell already present with positive value",
			initialBook: map[string]int{"ice": 3},
			spellName:   "ice",
			wantChanged: false,
			wantValue:   3,
			wantExists:  true,
		},
		{
			name:        "Learn spell already present with zero value",
			initialBook: map[string]int{"curse": 0},
			spellName:   "curse",
			wantChanged: false,
			wantValue:   0,
			wantExists:  true,
		},
		{
			name:        "Learn spell already present with negative value",
			initialBook: map[string]int{"bless": -2},
			spellName:   "bless",
			wantChanged: false,
			wantValue:   -2,
			wantExists:  true,
		},
		{
			name:        "Learn another new spell with other spells present",
			initialBook: map[string]int{"fireball": 2, "heal": 1},
			spellName:   "lightning",
			wantChanged: true,
			wantValue:   1,
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			for k, v := range tt.initialBook {
				c.SpellBook[k] = v
			}
			got := c.LearnSpell(tt.spellName)
			assert.Equal(t, tt.wantChanged, got)
			val, exists := c.SpellBook[tt.spellName]
			assert.Equal(t, tt.wantExists, exists)
			if exists {
				assert.Equal(t, tt.wantValue, val)
			}
		})
	}
}
func TestCharacter_TrackCharmed(t *testing.T) {
	tests := []struct {
		name        string
		initial     []int
		mobId       int
		add         bool
		wantCharmed []int
	}{
		{
			name:        "Add mobId to empty list",
			initial:     nil,
			mobId:       101,
			add:         true,
			wantCharmed: []int{101},
		},
		{
			name:        "Add mobId to existing list",
			initial:     []int{201, 202},
			mobId:       203,
			add:         true,
			wantCharmed: []int{201, 202, 203},
		},
		{
			name:        "Add duplicate mobId (should not duplicate)",
			initial:     []int{301, 302},
			mobId:       301,
			add:         true,
			wantCharmed: []int{301, 302},
		},
		{
			name:        "Remove mobId from list",
			initial:     []int{401, 402, 403},
			mobId:       402,
			add:         false,
			wantCharmed: []int{401, 403},
		},
		{
			name:        "Remove mobId not in list (should do nothing)",
			initial:     []int{501, 502},
			mobId:       999,
			add:         false,
			wantCharmed: []int{501, 502, 999},
		},
		{
			name:        "Remove mobId from single-element list",
			initial:     []int{601},
			mobId:       601,
			add:         false,
			wantCharmed: []int{},
		},
		{
			name:        "Add mobId already present (should append again)",
			initial:     []int{701},
			mobId:       701,
			add:         true,
			wantCharmed: []int{701},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initial != nil {
				c.CharmedMobs = append([]int{}, tt.initial...)
			}
			c.TrackCharmed(tt.mobId, tt.add)
			assert.Equal(t, tt.wantCharmed, c.CharmedMobs)
		})
	}
}
func TestCharacter_GetCharmIds(t *testing.T) {
	tests := []struct {
		name        string
		charmedMobs []int
		want        []int
	}{
		{
			name:        "Nil CharmedMobs returns empty slice",
			charmedMobs: nil,
			want:        []int{},
		},
		{
			name:        "Empty CharmedMobs returns empty slice",
			charmedMobs: []int{},
			want:        []int{},
		},
		{
			name:        "Single mob in CharmedMobs",
			charmedMobs: []int{101},
			want:        []int{101},
		},
		{
			name:        "Multiple mobs in CharmedMobs",
			charmedMobs: []int{201, 202, 203},
			want:        []int{201, 202, 203},
		},
		{
			name:        "CharmedMobs with duplicate ids",
			charmedMobs: []int{301, 302, 301},
			want:        []int{301, 302, 301},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.charmedMobs != nil {
				c.CharmedMobs = append([]int{}, tt.charmedMobs...)
			} else {
				c.CharmedMobs = nil
			}
			got := c.GetCharmIds()
			assert.Equal(t, tt.want, got)
			// Ensure returned slice is a copy, not the same reference
			if c.CharmedMobs != nil {
				if len(got) > 0 {
					got[0] = -999
					assert.NotEqual(t, c.CharmedMobs[0], got[0], "GetCharmIds should return a copy, not the original slice")
				}
			}
		})
	}
}
func TestCharacter_GetCharmedUserId(t *testing.T) {
	tests := []struct {
		name       string
		charmed    *CharmInfo
		wantUserId int
	}{
		{
			name:       "Charmed is nil returns 0",
			charmed:    nil,
			wantUserId: 0,
		},
		{
			name:       "Charmed is set returns userId",
			charmed:    &CharmInfo{UserId: 42},
			wantUserId: 42,
		},
		{
			name:       "Charmed is set with negative userId",
			charmed:    &CharmInfo{UserId: -7},
			wantUserId: -7,
		},
		{
			name:       "Charmed is set with zero userId",
			charmed:    &CharmInfo{UserId: 0},
			wantUserId: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Charmed = tt.charmed
			got := c.GetCharmedUserId()
			assert.Equal(t, tt.wantUserId, got)
		})
	}
}
func TestCharacter_IsCharmed(t *testing.T) {
	tests := []struct {
		name    string
		charmed *CharmInfo
		userIds []int
		want    bool
	}{
		{
			name:    "No charmed info returns false",
			charmed: nil,
			userIds: nil,
			want:    false,
		},
		{
			name:    "Charmed info, no userId arg returns true",
			charmed: &CharmInfo{UserId: 42},
			userIds: nil,
			want:    true,
		},
		{
			name:    "Charmed info, userId matches",
			charmed: &CharmInfo{UserId: 99},
			userIds: []int{99},
			want:    true,
		},
		{
			name:    "Charmed info, userId does not match",
			charmed: &CharmInfo{UserId: 77},
			userIds: []int{88},
			want:    false,
		},
		{
			name:    "Charmed info, multiple userIds, one matches",
			charmed: &CharmInfo{UserId: 55},
			userIds: []int{11, 22, 55, 99},
			want:    true,
		},
		{
			name:    "Charmed info, multiple userIds, none match",
			charmed: &CharmInfo{UserId: 101},
			userIds: []int{1, 2, 3},
			want:    false,
		},
		{
			name:    "Charmed info, userId is zero, matches zero",
			charmed: &CharmInfo{UserId: 0},
			userIds: []int{0},
			want:    true,
		},
		{
			name:    "Charmed info, userId is negative, matches negative",
			charmed: &CharmInfo{UserId: -5},
			userIds: []int{-5},
			want:    true,
		},
		{
			name:    "Charmed info, userId is negative, does not match",
			charmed: &CharmInfo{UserId: -5},
			userIds: []int{5},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Charmed = tt.charmed
			got := c.IsCharmed(tt.userIds...)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_RemoveCharm(t *testing.T) {
	tests := []struct {
		name           string
		charmed        *CharmInfo
		adjectives     []string
		wantUserId     int
		wantCharmed    *CharmInfo
		wantAdjectives []string
	}{
		{
			name:           "No charmed info, no adjectives",
			charmed:        nil,
			adjectives:     nil,
			wantUserId:     0,
			wantCharmed:    nil,
			wantAdjectives: []string{},
		},
		{
			name:           "Charmed info present, no adjectives",
			charmed:        &CharmInfo{UserId: 42},
			adjectives:     nil,
			wantUserId:     42,
			wantCharmed:    nil,
			wantAdjectives: []string{},
		},
		{
			name:           "Charmed info present, 'charmed' adjective present",
			charmed:        &CharmInfo{UserId: 7},
			adjectives:     []string{"charmed", "sleepy"},
			wantUserId:     7,
			wantCharmed:    nil,
			wantAdjectives: []string{"sleepy"},
		},
		{
			name:           "Charmed info present, 'charmed' not in adjectives",
			charmed:        &CharmInfo{UserId: 99},
			adjectives:     []string{"sleepy"},
			wantUserId:     99,
			wantCharmed:    nil,
			wantAdjectives: []string{"sleepy"},
		},
		{
			name:           "Charmed info present, 'charmed' is only adjective",
			charmed:        &CharmInfo{UserId: 123},
			adjectives:     []string{"charmed"},
			wantUserId:     123,
			wantCharmed:    nil,
			wantAdjectives: []string{},
		},
		{
			name:           "No charmed info, 'charmed' adjective present",
			charmed:        nil,
			adjectives:     []string{"charmed", "alert"},
			wantUserId:     0,
			wantCharmed:    nil,
			wantAdjectives: []string{"alert"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Charmed = tt.charmed
			if tt.adjectives != nil {
				c.Adjectives = append([]string{}, tt.adjectives...)
			}
			got := c.RemoveCharm()
			assert.Equal(t, tt.wantUserId, got)
			assert.Equal(t, tt.wantCharmed, c.Charmed)
			assert.ElementsMatch(t, tt.wantAdjectives, c.Adjectives)
		})
	}
}
func TestCharacter_GetAllSkillRanks(t *testing.T) {
	tests := []struct {
		name   string
		skills map[string]int
		want   map[string]int
	}{
		{
			name:   "Nil Skills returns empty map",
			skills: nil,
			want:   map[string]int{},
		},
		{
			name:   "Empty Skills returns empty map",
			skills: map[string]int{},
			want:   map[string]int{},
		},
		{
			name:   "Single skill",
			skills: map[string]int{"sword": 2},
			want:   map[string]int{"sword": 2},
		},
		{
			name:   "Multiple skills",
			skills: map[string]int{"sword": 2, "archery": 3, "magic": 1},
			want:   map[string]int{"sword": 2, "archery": 3, "magic": 1},
		},
		{
			name:   "Skill with zero value",
			skills: map[string]int{"alchemy": 0},
			want:   map[string]int{"alchemy": 0},
		},
		{
			name:   "Skill with negative value",
			skills: map[string]int{"curse": -1},
			want:   map[string]int{"curse": -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.skills != nil {
				c.Skills = make(map[string]int)
				for k, v := range tt.skills {
					c.Skills[k] = v
				}
			} else {
				c.Skills = nil
			}
			got := c.GetAllSkillRanks()
			assert.Equal(t, tt.want, got)
			// Ensure returned map is a copy, not the same reference
			if c.Skills != nil {
				got["newskill"] = 99
				_, exists := c.Skills["newskill"]
				assert.False(t, exists, "GetAllSkillRanks should return a copy, not the original map")
			}
		})
	}
}
func TestCharacter_HasAdjective(t *testing.T) {
	tests := []struct {
		name       string
		adjectives []string
		checkAdj   string
		want       bool
	}{
		{
			name:       "No adjectives, check any",
			adjectives: nil,
			checkAdj:   "sleeping",
			want:       false,
		},
		{
			name:       "Empty adjectives slice, check any",
			adjectives: []string{},
			checkAdj:   "dead",
			want:       false,
		},
		{
			name:       "Single adjective, present",
			adjectives: []string{"wounded"},
			checkAdj:   "wounded",
			want:       true,
		},
		{
			name:       "Single adjective, not present",
			adjectives: []string{"wounded"},
			checkAdj:   "sleeping",
			want:       false,
		},
		{
			name:       "Multiple adjectives, present",
			adjectives: []string{"sleeping", "dead", "wounded"},
			checkAdj:   "dead",
			want:       true,
		},
		{
			name:       "Multiple adjectives, not present",
			adjectives: []string{"sleeping", "dead", "wounded"},
			checkAdj:   "awake",
			want:       false,
		},
		{
			name:       "Adjective is empty string, present",
			adjectives: []string{"", "foo"},
			checkAdj:   "",
			want:       true,
		},
		{
			name:       "Adjective is empty string, not present",
			adjectives: []string{"foo", "bar"},
			checkAdj:   "",
			want:       false,
		},
		{
			name:       "Case sensitive match",
			adjectives: []string{"Sleeping"},
			checkAdj:   "sleeping",
			want:       false,
		},
		{
			name:       "Case sensitive exact match",
			adjectives: []string{"sleeping"},
			checkAdj:   "sleeping",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Adjectives = tt.adjectives
			got := c.HasAdjective(tt.checkAdj)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_SetAdjective(t *testing.T) {
	tests := []struct {
		name      string
		initial   []string
		adj       string
		addToList bool
		want      []string
	}{
		{
			name:      "Add adjective to empty list",
			initial:   nil,
			adj:       "sleeping",
			addToList: true,
			want:      []string{"sleeping"},
		},
		{
			name:      "Add adjective to non-empty list",
			initial:   []string{"wounded"},
			adj:       "sleeping",
			addToList: true,
			want:      []string{"wounded", "sleeping"},
		},
		{
			name:      "Add duplicate adjective (should not duplicate)",
			initial:   []string{"sleeping"},
			adj:       "sleeping",
			addToList: true,
			want:      []string{"sleeping"},
		},
		{
			name:      "Remove adjective from list",
			initial:   []string{"sleeping", "wounded"},
			adj:       "sleeping",
			addToList: false,
			want:      []string{"wounded"},
		},
		{
			name:      "Remove adjective not present (should do nothing)",
			initial:   []string{"wounded"},
			adj:       "sleeping",
			addToList: false,
			want:      []string{"wounded"},
		},
		{
			name:      "Remove last adjective (should result in empty list)",
			initial:   []string{"sleeping"},
			adj:       "sleeping",
			addToList: false,
			want:      []string{},
		},
		{
			name:      "Add multiple adjectives sequentially",
			initial:   []string{},
			adj:       "sleeping",
			addToList: true,
			want:      []string{"sleeping"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initial != nil {
				c.Adjectives = append([]string{}, tt.initial...)
			} else {
				c.Adjectives = nil
			}
			c.SetAdjective(tt.adj, tt.addToList)
			assert.Equal(t, tt.want, c.Adjectives)
		})
	}

	t.Run("Add then remove adjective", func(t *testing.T) {
		c := New()
		c.SetAdjective("sleeping", true)
		assert.Equal(t, []string{"sleeping"}, c.Adjectives)
		c.SetAdjective("sleeping", false)
		assert.Equal(t, []string{}, c.Adjectives)
	})

	t.Run("Remove from nil adjectives", func(t *testing.T) {
		c := New()
		c.Adjectives = nil
		c.SetAdjective("sleeping", false)
		assert.Equal(t, []string{}, c.Adjectives)
	})

	t.Run("Add to nil adjectives", func(t *testing.T) {
		c := New()
		c.Adjectives = nil
		c.SetAdjective("sleeping", true)
		assert.Equal(t, []string{"sleeping"}, c.Adjectives)
	})
}
func TestCharacter_PruneCooldowns(t *testing.T) {
	type cooldownsCase struct {
		name         string
		initial      Cooldowns
		expectPruned Cooldowns
	}
	tests := []cooldownsCase{
		{
			name:         "Nil Cooldowns map does nothing",
			initial:      nil,
			expectPruned: nil,
		},
		{
			name:         "Empty Cooldowns map does nothing",
			initial:      Cooldowns{},
			expectPruned: Cooldowns{},
		},
		{
			name: "Cooldowns with zero and positive values",
			initial: Cooldowns{
				"foo": 0,
				"bar": 2,
				"baz": 0,
			},
			expectPruned: Cooldowns{
				"bar": 2,
			},
		},
		{
			name: "Cooldowns with all zero values",
			initial: Cooldowns{
				"a": 0,
				"b": 0,
			},
			expectPruned: Cooldowns{},
		},
		{
			name: "Cooldowns with all positive values",
			initial: Cooldowns{
				"x": 1,
				"y": 2,
			},
			expectPruned: Cooldowns{
				"x": 1,
				"y": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initial != nil {
				c.Cooldowns = make(Cooldowns)
				for k, v := range tt.initial {
					c.Cooldowns[k] = v
				}
			} else {
				c.Cooldowns = nil
			}
			c.PruneCooldowns()
			if tt.expectPruned == nil {
				assert.Nil(t, c.Cooldowns)
			} else {
				assert.Equal(t, tt.expectPruned, c.Cooldowns)
			}
		})
	}
}
func TestCharacter_GetCooldown(t *testing.T) {
	tests := []struct {
		name         string
		cooldowns    map[string]int
		trackingTag  string
		wantCooldown int
	}{
		{
			name:         "Nil Cooldowns map returns 0 and initializes map",
			cooldowns:    nil,
			trackingTag:  "foo",
			wantCooldown: 0,
		},
		{
			name:         "Empty Cooldowns map returns 0",
			cooldowns:    map[string]int{},
			trackingTag:  "bar",
			wantCooldown: 0,
		},
		{
			name:         "Cooldown exists for tag",
			cooldowns:    map[string]int{"baz": 7, "qux": 3},
			trackingTag:  "baz",
			wantCooldown: 7,
		},
		{
			name:         "Cooldown exists for another tag",
			cooldowns:    map[string]int{"baz": 7, "qux": 3},
			trackingTag:  "qux",
			wantCooldown: 3,
		},
		{
			name:         "Cooldown does not exist for tag returns 0",
			cooldowns:    map[string]int{"baz": 7},
			trackingTag:  "notfound",
			wantCooldown: 0,
		},
		{
			name:         "Cooldown with zero value",
			cooldowns:    map[string]int{"zero": 0},
			trackingTag:  "zero",
			wantCooldown: 0,
		},
		{
			name:         "Cooldown with negative value",
			cooldowns:    map[string]int{"neg": -5},
			trackingTag:  "neg",
			wantCooldown: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.cooldowns != nil {
				c.Cooldowns = make(Cooldowns)
				for k, v := range tt.cooldowns {
					c.Cooldowns[k] = v
				}
			} else {
				c.Cooldowns = nil
			}
			got := c.GetCooldown(tt.trackingTag)
			assert.Equal(t, tt.wantCooldown, got)
			assert.NotNil(t, c.Cooldowns, "Cooldowns map should be initialized")
		})
	}
}
func TestCharacter_GetAllCooldowns(t *testing.T) {
	tests := []struct {
		name      string
		cooldowns map[string]int
		want      map[string]int
	}{
		{
			name:      "Nil Cooldowns returns empty map",
			cooldowns: nil,
			want:      map[string]int{},
		},
		{
			name:      "Empty Cooldowns returns empty map",
			cooldowns: map[string]int{},
			want:      map[string]int{},
		},
		{
			name:      "Single cooldown",
			cooldowns: map[string]int{"attack": 3},
			want:      map[string]int{"attack": 3},
		},
		{
			name:      "Multiple cooldowns",
			cooldowns: map[string]int{"attack": 3, "cast": 5, "move": 1},
			want:      map[string]int{"attack": 3, "cast": 5, "move": 1},
		},
		{
			name:      "Cooldown with zero value",
			cooldowns: map[string]int{"wait": 0},
			want:      map[string]int{"wait": 0},
		},
		{
			name:      "Cooldown with negative value",
			cooldowns: map[string]int{"buggy": -2},
			want:      map[string]int{"buggy": -2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.cooldowns != nil {
				c.Cooldowns = make(Cooldowns)
				for k, v := range tt.cooldowns {
					c.Cooldowns[k] = v
				}
			} else {
				c.Cooldowns = nil
			}
			got := c.GetAllCooldowns()
			assert.Equal(t, tt.want, got)
			// Ensure returned map is a copy, not the same reference
			if c.Cooldowns != nil {
				got["newcd"] = 99
				_, exists := c.Cooldowns["newcd"]
				assert.False(t, exists, "GetAllCooldowns should return a copy, not the original map")
			}
		})
	}
}
func TestCharacter_SetSetting(t *testing.T) {
	tests := []struct {
		name         string
		initial      map[string]string
		settingName  string
		settingValue string
		want         map[string]string
	}{
		{
			name:         "Set new setting",
			initial:      nil,
			settingName:  "color",
			settingValue: "blue",
			want:         map[string]string{"color": "blue"},
		},
		{
			name:         "Overwrite existing setting",
			initial:      map[string]string{"color": "red"},
			settingName:  "color",
			settingValue: "green",
			want:         map[string]string{"color": "green"},
		},
		{
			name:         "Add another setting",
			initial:      map[string]string{"volume": "high"},
			settingName:  "brightness",
			settingValue: "low",
			want:         map[string]string{"volume": "high", "brightness": "low"},
		},
		{
			name:         "Delete existing setting by setting value to empty",
			initial:      map[string]string{"foo": "bar"},
			settingName:  "foo",
			settingValue: "",
			want:         map[string]string{},
		},
		{
			name:         "Delete non-existing setting by setting value to empty",
			initial:      map[string]string{"foo": "bar"},
			settingName:  "baz",
			settingValue: "",
			want:         map[string]string{"foo": "bar"},
		},
		{
			name:         "Set setting with empty name",
			initial:      nil,
			settingName:  "",
			settingValue: "someval",
			want:         map[string]string{"": "someval"},
		},
		{
			name:         "Delete setting with empty name",
			initial:      map[string]string{"": "someval", "other": "val"},
			settingName:  "",
			settingValue: "",
			want:         map[string]string{"other": "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initial != nil {
				c.Settings = make(map[string]string)
				for k, v := range tt.initial {
					c.Settings[k] = v
				}
			} else {
				c.Settings = nil
			}
			c.SetSetting(tt.settingName, tt.settingValue)
			assert.Equal(t, tt.want, c.Settings)
		})
	}
}
func TestCharacter_GetSetting(t *testing.T) {
	tests := []struct {
		name        string
		initial     map[string]string
		settingName string
		want        string
	}{
		{
			name:        "Nil Settings returns empty string",
			initial:     nil,
			settingName: "foo",
			want:        "",
		},
		{
			name:        "Empty Settings returns empty string",
			initial:     map[string]string{},
			settingName: "foo",
			want:        "",
		},
		{
			name:        "Setting exists returns value",
			initial:     map[string]string{"foo": "bar"},
			settingName: "foo",
			want:        "bar",
		},
		{
			name:        "Setting does not exist returns empty string",
			initial:     map[string]string{"foo": "bar"},
			settingName: "baz",
			want:        "",
		},
		{
			name:        "Multiple settings, get correct one",
			initial:     map[string]string{"foo": "bar", "baz": "qux"},
			settingName: "baz",
			want:        "qux",
		},
		{
			name:        "Setting exists with empty value",
			initial:     map[string]string{"empty": ""},
			settingName: "empty",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.initial != nil {
				c.Settings = make(map[string]string)
				for k, v := range tt.initial {
					c.Settings[k] = v
				}
			} else {
				c.Settings = nil
			}
			got := c.GetSetting(tt.settingName)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_IsDisabled(t *testing.T) {
	tests := []struct {
		name       string
		health     int
		stamina    int
		conviction int
		want       bool
	}{
		{
			name:       "All pools positive",
			health:     10,
			stamina:    10,
			conviction: 10,
			want:       false,
		},
		{
			name:       "Health is zero",
			health:     0,
			stamina:    10,
			conviction: 10,
			want:       true,
		},
		{
			name:       "Health is negative",
			health:     -1,
			stamina:    10,
			conviction: 10,
			want:       true,
		},
		{
			name:       "All pools large positive",
			health:     100,
			stamina:    100,
			conviction: 100,
			want:       false,
		},
		{
			name:       "Health is large negative",
			health:     -100,
			stamina:    10,
			conviction: 10,
			want:       true,
		},
		{
			name:       "Stamina is zero but health positive",
			health:     10,
			stamina:    0,
			conviction: 10,
			want:       false,
		},
		{
			name:       "Conviction is zero but health positive",
			health:     10,
			stamina:    10,
			conviction: 0,
			want:       false,
		},
		{
			name:       "Multiple pools depleted",
			health:     -5,
			stamina:    -3,
			conviction: 10,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Health = tt.health
			c.Stamina = tt.stamina
			c.Conviction = tt.conviction
			got := c.IsDisabled()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestCharacter_EndAggro(t *testing.T) {
	tests := []struct {
		name       string
		setupAggro bool
	}{
		{
			name:       "Aggro is nil, call EndAggro",
			setupAggro: false,
		},
		{
			name:       "Aggro is set, call EndAggro",
			setupAggro: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.setupAggro {
				c.Aggro = &Aggro{
					UserId:        1,
					MobInstanceId: 2,
					Type:          DefaultAttack,
					RoundsWaiting: 3,
				}
			} else {
				c.Aggro = nil
			}
			c.EndAggro()
			assert.Nil(t, c.Aggro)
		})
	}
}
func TestCharacter_GetSkillLevel(t *testing.T) {
	type args struct {
		skillsMap map[string]int
		skillTag  skills.SkillTag
	}
	tests := []struct {
		name     string
		args     args
		expected int
	}{
		{
			name: "Skill exists with positive value",
			args: args{
				skillsMap: map[string]int{string(skills.WeaponCombat): 3},
				skillTag:  skills.WeaponCombat,
			},
			expected: 3,
		},
		{
			name: "Skill exists with zero value",
			args: args{
				skillsMap: map[string]int{string(skills.Cast): 0},
				skillTag:  skills.Cast,
			},
			expected: 0,
		},
		{
			name: "Skill does not exist",
			args: args{
				skillsMap: map[string]int{string(skills.Spellcasting): 2},
				skillTag:  skills.Cast,
			},
			expected: 0,
		},
		{
			name: "Nil Skills map",
			args: args{
				skillsMap: nil,
				skillTag:  skills.Search,
			},
			expected: 0,
		},
		{
			name: "Multiple skills, get correct one",
			args: args{
				skillsMap: map[string]int{
					string(skills.Cast):      2,
					string(skills.WeaponCombat): 1,
					string(skills.Search):       4,
				},
				skillTag: skills.Search,
			},
			expected: 4,
		},
		{
			name: "Skill exists with negative value",
			args: args{
				skillsMap: map[string]int{string(skills.Cast): -2},
				skillTag:  skills.Cast,
			},
			expected: -2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.args.skillsMap != nil {
				c.Skills = make(map[string]int)
				maps.Copy(c.Skills, tt.args.skillsMap)
			} else {
				c.Skills = nil
			}
			got := c.GetSkillLevel(tt.args.skillTag)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCharacter_DeductStamina(t *testing.T) {
	tests := []struct {
		name            string
		initialStamina  int
		deductAmount    int
		expectedSuccess bool
		expectedStamina int
	}{
		{
			name:            "Sufficient stamina",
			initialStamina:  100,
			deductAmount:    20,
			expectedSuccess: true,
			expectedStamina: 80,
		},
		{
			name:            "Exactly enough stamina",
			initialStamina:  50,
			deductAmount:    50,
			expectedSuccess: true,
			expectedStamina: 0,
		},
		{
			name:            "Insufficient stamina",
			initialStamina:  10,
			deductAmount:    20,
			expectedSuccess: false,
			expectedStamina: 10,
		},
		{
			name:            "Zero stamina deduction",
			initialStamina:  50,
			deductAmount:    0,
			expectedSuccess: true,
			expectedStamina: 50,
		},
		{
			name:            "Deduct from zero stamina",
			initialStamina:  0,
			deductAmount:    10,
			expectedSuccess: false,
			expectedStamina: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Stamina = tt.initialStamina
			success := c.DeductStamina(tt.deductAmount)
			assert.Equal(t, tt.expectedSuccess, success, "DeductStamina return value")
			assert.Equal(t, tt.expectedStamina, c.Stamina, "Final stamina value")
		})
	}
}

func TestCharacter_GetMovementStaminaCost(t *testing.T) {
	// These tests use characters with no items (zero carried weight),
	// so only terrain multiplier affects cost. Encumbrance tests require
	// real item specs and are covered by integration tests.
	tests := []struct {
		name              string
		terrainMultiplier float64
		expectedCost      int
	}{
		{
			name:              "Normal terrain, no items",
			terrainMultiplier: 1.0,
			expectedCost:      2, // baseCost 2.0 * 1.0 = 2
		},
		{
			name:              "Easy terrain (road), no items",
			terrainMultiplier: 0.5,
			expectedCost:      1, // baseCost 2.0 * 0.5 = 1
		},
		{
			name:              "Rough terrain (mountains), no items",
			terrainMultiplier: 2.0,
			expectedCost:      4, // baseCost 2.0 * 2.0 = 4
		},
		{
			name:              "Very rough terrain, no items",
			terrainMultiplier: 3.0,
			expectedCost:      6, // baseCost 2.0 * 3.0 = 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Stats.Strength.Value = 100
			c.Stats.Strength.ValueAdj = 100

			cost := c.GetMovementStaminaCost(tt.terrainMultiplier)
			assert.Equal(t, tt.expectedCost, cost, "Movement stamina cost")
		})
	}
}

func TestCharacter_GetAttackStaminaCost(t *testing.T) {
	tests := []struct {
		name         string
		weaponId     int
		offhandId    int
		expectedCost int
	}{
		{
			name:         "Unarmed combat",
			weaponId:     0,
			offhandId:    0,
			expectedCost: 4,
		},
		// Note: Testing with actual weapons requires loading weapon data
		// which depends on item loading infrastructure. The weapon cost
		// logic is tested via integration tests with real weapons.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.weaponId > 0 {
				c.Equipment.Weapon = items.Item{ItemId: tt.weaponId}
			}
			if tt.offhandId > 0 {
				c.Equipment.Offhand = items.Item{ItemId: tt.offhandId}
			}

			cost := c.GetAttackStaminaCost()
			assert.Equal(t, tt.expectedCost, cost, "Attack stamina cost")
		})
	}
}

func TestCharacter_DeductAttackStamina(t *testing.T) {
	tests := []struct {
		name            string
		initialStamina  int
		weaponId        int
		expectedDeducted int
		expectedRemaining int
	}{
		{
			name:            "Sufficient stamina - unarmed",
			initialStamina:  100,
			weaponId:        0,
			expectedDeducted: 4,
			expectedRemaining: 96,
		},
		{
			name:            "Insufficient stamina",
			initialStamina:  2,
			weaponId:        0,
			expectedDeducted: 2, // Deducts what's available
			expectedRemaining: 0,
		},
		{
			name:            "Zero stamina",
			initialStamina:  0,
			weaponId:        0,
			expectedDeducted: 0,
			expectedRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Stamina = tt.initialStamina
			if tt.weaponId > 0 {
				c.Equipment.Weapon = items.Item{ItemId: tt.weaponId}
			}

			deducted := c.DeductAttackStamina()
			assert.Equal(t, tt.expectedDeducted, deducted, "Stamina deducted")
			assert.Equal(t, tt.expectedRemaining, c.Stamina, "Remaining stamina")
		})
	}
}

func TestCharacter_GetModifiedAttackCount(t *testing.T) {
	tests := []struct {
		name           string
		baseAttacks    int
		weaponSpeed    float64
		isOffhand      bool
		combatSkill    int
		dualWieldSkill int
		expected       int
	}{
		{
			name:           "Unarmed, no skills",
			baseAttacks:    3,
			weaponSpeed:    1.0,
			isOffhand:      false,
			combatSkill:    1,
			dualWieldSkill: 0,
			expected:       3, // 3 * 1.0 * 1.02 ≈ 3
		},
		{
			name:           "Unarmed, high combat skill",
			baseAttacks:    3,
			weaponSpeed:    1.0,
			isOffhand:      false,
			combatSkill:    50,
			dualWieldSkill: 0,
			expected:       3, // 3 * 1.0 * 1.1 = 3.3 ≈ 3
		},
		{
			name:           "Slow weapon (0.6x), no skills",
			baseAttacks:    3,
			weaponSpeed:    0.6,
			isOffhand:      false,
			combatSkill:    1,
			dualWieldSkill: 0,
			expected:       2, // 3 * 0.6 * 1.02 ≈ 1.8 ≈ 2
		},
		{
			name:           "Slow weapon (0.6x), high combat skill",
			baseAttacks:    3,
			weaponSpeed:    0.6,
			isOffhand:      false,
			combatSkill:    50,
			dualWieldSkill: 0,
			expected:       2, // 3 * 0.6 * 1.1 = 1.98 ≈ 2
		},
		{
			name:           "Fast weapon (1.2x), no skills",
			baseAttacks:    3,
			weaponSpeed:    1.2,
			isOffhand:      false,
			combatSkill:    1,
			dualWieldSkill: 0,
			expected:       4, // 3 * 1.2 * 1.02 ≈ 3.7 ≈ 4
		},
		{
			name:           "Offhand weapon, no dual wield skill",
			baseAttacks:    3,
			weaponSpeed:    0.8,
			isOffhand:      true,
			combatSkill:    1,
			dualWieldSkill: 0,
			expected:       1, // 3 * 0.8 * 1.02 * 0.5 ≈ 1.2 ≈ 1
		},
		{
			name:           "Offhand weapon, moderate dual wield skill",
			baseAttacks:    3,
			weaponSpeed:    0.8,
			isOffhand:      true,
			combatSkill:    1,
			dualWieldSkill: 25,
			expected:       2, // 3 * 0.8 * 1.02 * 0.85 ≈ 2.1 ≈ 2
		},
		{
			name:           "Offhand weapon, high dual wield skill",
			baseAttacks:    3,
			weaponSpeed:    0.8,
			isOffhand:      true,
			combatSkill:    1,
			dualWieldSkill: 50,
			expected:       3, // 3 * 0.8 * 1.02 * 1.2 ≈ 2.9 ≈ 3
		},
		{
			name:           "Minimum 1 attack",
			baseAttacks:    1,
			weaponSpeed:    0.3,
			isOffhand:      true,
			combatSkill:    1,
			dualWieldSkill: 0,
			expected:       1, // 1 * 0.3 * 1.02 * 0.5 < 1, min is 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Skills[string(skills.UnarmedCombat)] = tt.combatSkill
			c.Skills[string(skills.WeaponCombat)] = tt.dualWieldSkill

			result := c.GetModifiedAttackCount(tt.baseAttacks, tt.weaponSpeed, tt.isOffhand)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ─── Stage 40.2: Mitigation & Defense tests ─────────────────────────────────

func TestGetPhysicalMitigation(t *testing.T) {
	tests := []struct {
		name string
		body int // PhysicalMitigation on body armor
		legs int // PhysicalMitigation on legs armor
		want float64
	}{
		{"no equipment", 0, 0, 0.0},
		{"body only (10%)", 10, 0, 0.10},
		{"body + legs", 15, 10, 0.25},
		{"high values", 40, 35, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.body > 0 {
				c.Equipment.Body = items.Item{ItemId: 1, Spec: &items.ItemSpec{
					PhysicalMitigation: tt.body,
				}}
			}
			if tt.legs > 0 {
				c.Equipment.Legs = items.Item{ItemId: 2, Spec: &items.ItemSpec{
					PhysicalMitigation: tt.legs,
				}}
			}
			got := c.GetPhysicalMitigation()
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

func TestGetMagicalMitigation(t *testing.T) {
	tests := []struct {
		name string
		neck int // MagicalMitigation on neck item
		ring int // MagicalMitigation on ring item
		want float64
	}{
		{"no equipment", 0, 0, 0.0},
		{"neck only (20%)", 20, 0, 0.20},
		{"neck + ring", 15, 10, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.neck > 0 {
				c.Equipment.Neck = items.Item{ItemId: 1, Spec: &items.ItemSpec{
					MagicalMitigation: tt.neck,
				}}
			}
			if tt.ring > 0 {
				c.Equipment.Ring = items.Item{ItemId: 2, Spec: &items.ItemSpec{
					MagicalMitigation: tt.ring,
				}}
			}
			got := c.GetMagicalMitigation()
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

func TestGetConvictionMitigation(t *testing.T) {
	tests := []struct {
		name  string
		glove int // ConvictionMitigation on gloves
		belt  int // ConvictionMitigation on belt
		want  float64
	}{
		{"no equipment", 0, 0, 0.0},
		{"gloves only (12%)", 12, 0, 0.12},
		{"gloves + belt", 12, 8, 0.20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			if tt.glove > 0 {
				c.Equipment.Gloves = items.Item{ItemId: 1, Spec: &items.ItemSpec{
					ConvictionMitigation: tt.glove,
				}}
			}
			if tt.belt > 0 {
				c.Equipment.Belt = items.Item{ItemId: 2, Spec: &items.ItemSpec{
					ConvictionMitigation: tt.belt,
				}}
			}
			got := c.GetConvictionMitigation()
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

func TestGetDefenseScore(t *testing.T) {
	t.Run("dodge — dex + unarmed skill", func(t *testing.T) {
		c := New()
		c.Stats.Dexterity.ValueAdj = 100
		c.Skills[string(skills.UnarmedCombat)] = 20
		score := c.GetDefenseScore(DefenseDodge)
		assert.InDelta(t, 140.0, score, 1.0) // 100 + 20*2 (SkillWeight=2.0)
	})

	t.Run("parry — dex + weapon skill + weapon parry", func(t *testing.T) {
		c := New()
		c.Stats.Dexterity.ValueAdj = 80
		c.Skills[string(skills.WeaponCombat)] = 15
		c.Equipment.Weapon = items.Item{ItemId: 1, Spec: &items.ItemSpec{
			ParryRating: 10,
		}}
		score := c.GetDefenseScore(DefenseParry)
		assert.InDelta(t, 120.0, score, 1.0) // 80 + 15*2 + 10 (SkillWeight=2.0)
	})

	t.Run("block — (str+dex)/2 + weapon skill + shield block", func(t *testing.T) {
		c := New()
		c.Stats.Strength.ValueAdj = 120
		c.Stats.Dexterity.ValueAdj = 80
		c.Skills[string(skills.WeaponCombat)] = 10
		c.Equipment.Offhand = items.Item{ItemId: 1, Spec: &items.ItemSpec{
			Type:            items.Offhand,
			DamageReduction: 5,
			BlockRating:     15,
		}}
		score := c.GetDefenseScore(DefenseBlock)
		assert.InDelta(t, 135.0, score, 1.0) // (120+80)/2 + 10*2 + 15 (SkillWeight=2.0)
	})

	t.Run("unknown defense type → 0", func(t *testing.T) {
		c := New()
		assert.Equal(t, 0.0, c.GetDefenseScore("unknown"))
	})
}

func TestGetDefenseStaminaCost(t *testing.T) {
	// Default multipliers are 0.9 for all three
	tests := []struct {
		name string
		def  string
		want int // base * 0.9 truncated to int
	}{
		{"dodge: 2 * 0.9 = 1", DefenseDodge, 1},
		{"parry: 4 * 0.9 = 3", DefenseParry, 3},
		{"block: 5 * 0.9 = 4", DefenseBlock, 4},
		{"unknown → 0", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			got := c.GetDefenseStaminaCost(tt.def)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCharacter_Discoveries(t *testing.T) {
	c := New()

	// No discoveries initially
	assert.False(t, c.HasDiscovery(100, "compartment"))

	// Add a discovery
	c.AddDiscovery(100, "compartment")
	assert.True(t, c.HasDiscovery(100, "compartment"))

	// Idempotent — adding again doesn't duplicate
	c.AddDiscovery(100, "compartment")
	assert.Equal(t, 1, len(c.Discoveries[100]))

	// Different room, different namespace
	assert.False(t, c.HasDiscovery(200, "compartment"))

	// Multiple discoveries in same room
	c.AddDiscovery(100, "chest")
	assert.True(t, c.HasDiscovery(100, "chest"))
	assert.Equal(t, 2, len(c.Discoveries[100]))
}

func TestCharacter_SkillMigration_TrackingAndForaging(t *testing.T) {
	c := New()
	c.Skills["tracking"] = 5
	c.Skills["foraging"] = 3
	c.SkillUseCount["tracking"] = 100
	c.SkillUseCount["foraging"] = 50
	delete(c.Skills, "search")

	c.Validate()

	assert.Equal(t, 5, c.Skills["search"], "search should be max(tracking, foraging)")
	assert.Equal(t, 150, c.SkillUseCount["search"], "use counts should be summed")
	assert.Zero(t, c.Skills["tracking"], "tracking should be removed")
	assert.Zero(t, c.Skills["foraging"], "foraging should be removed")
	assert.Zero(t, c.SkillUseCount["tracking"])
	assert.Zero(t, c.SkillUseCount["foraging"])
}

func TestCharacter_SkillMigration_ForagingOnly(t *testing.T) {
	c := New()
	c.Skills["foraging"] = 7
	c.SkillUseCount["foraging"] = 200
	delete(c.Skills, "search")
	delete(c.Skills, "tracking")

	c.Validate()

	assert.Equal(t, 7, c.Skills["search"])
	assert.Equal(t, 200, c.SkillUseCount["search"])
	assert.Zero(t, c.Skills["foraging"])
}

func TestCharacter_SkillMigration_Idempotent(t *testing.T) {
	c := New()
	c.Skills["search"] = 10
	c.SkillUseCount["search"] = 500

	c.Validate()

	assert.Equal(t, 10, c.Skills["search"], "should not change existing search")
	assert.Equal(t, 500, c.SkillUseCount["search"])
}
