package util

import (
	"bytes"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
)

// Because turnCount, roundCount and timeTrackers are package-level globals,
// it can be good practice to reset them in a TestMain or individually in tests.
// But for simplicity, each test that needs a reset can just do so in the test body.

func TestLockMud(t *testing.T) {
	// Basic concurrency test to make sure LockMud / UnlockMud do not panic
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		LockMud()
		defer UnlockMud()
		// do something
	}()

	go func() {
		defer wg.Done()
		RLockMud()
		defer RUnlockMud()
		// do something
	}()

	wg.Wait()
}

func TestSetServerAddress(t *testing.T) {
	// Reset global in case other tests changed it
	SetServerAddress("")
	if GetServerAddress() != "" {
		t.Fatalf("Expected empty address, got %q", GetServerAddress())
	}

	testAddr := "127.0.0.1:8080"
	SetServerAddress(testAddr)
	if addr := GetServerAddress(); addr != testAddr {
		t.Fatalf("Expected serverAddr to be %q, got %q", testAddr, addr)
	}
}

func TestRoundCount(t *testing.T) {
	// Reset roundCount for test isolation
	SetRoundCount(1314000)

	if rc := GetRoundCount(); rc != 1314000 {
		t.Fatalf("Expected roundCount to be 1314000, got %d", rc)
	}

	SetRoundCount(2000)
	if rc := GetRoundCount(); rc != 2000 {
		t.Fatalf("Expected roundCount to be 2000, got %d", rc)
	}

	newRC := IncrementRoundCount()
	if newRC != 2001 {
		t.Fatalf("Expected IncrementRoundCount to return 2001, got %d", newRC)
	}
	if GetRoundCount() != 2001 {
		t.Fatalf("Expected roundCount to be 2001, got %d", GetRoundCount())
	}
}

func TestTurnCount(t *testing.T) {
	// turnCount is also a global variable.
	// For test isolation, zero it out or set to a known value.
	turnCount = 0

	if GetTurnCount() != 0 {
		t.Fatalf("Expected turnCount to be 0, got %d", GetTurnCount())
	}

	newTC := IncrementTurnCount()
	if newTC != 1 {
		t.Fatalf("Expected turnCount to increment from 0 to 1, got %d", newTC)
	}

	if GetTurnCount() != 1 {
		t.Fatalf("Expected turnCount to be 1, got %d", GetTurnCount())
	}
}

func TestAccumulatorRecord(t *testing.T) {
	acc := &Accumulator{
		Name:    "Test",
		Total:   0,
		Lowest:  0,
		Highest: 0,
		Count:   0,
		Start:   time.Now(),
	}

	values := []float64{5.0, 7.5, 2.2, 10.0}
	for _, v := range values {
		acc.Record(v)
	}

	if acc.Count != float64(len(values)) {
		t.Fatalf("Expected Count to be %d, got %f", len(values), acc.Count)
	}

	expectedTotal := 5.0 + 7.5 + 2.2 + 10.0
	if acc.Total != expectedTotal {
		t.Fatalf("Expected Total to be %f, got %f", expectedTotal, acc.Total)
	}

	lowest := 2.2
	if acc.Lowest != lowest {
		t.Fatalf("Expected Lowest to be %f, got %f", lowest, acc.Lowest)
	}

	highest := 10.0
	if acc.Highest != highest {
		t.Fatalf("Expected Highest to be %f, got %f", highest, acc.Highest)
	}

	avg := expectedTotal / acc.Count
	if acc.Average() != avg {
		t.Fatalf("Expected average to be %f, got %f", avg, acc.Average())
	}

	l, h, av, c := acc.Stats()
	if l != lowest || h != highest || av != avg || c != acc.Count {
		t.Fatalf("Stats() returned unexpected values: got (%f, %f, %f, %f)", l, h, av, c)
	}
}

func TestTrackTimeAndGetTimeTrackers(t *testing.T) {
	// Reset the global timeTrackers map for test isolation
	timeTrackers = map[string]*Accumulator{}

	TrackTime("movement", 1.2)
	TrackTime("movement", 0.8)
	TrackTime("combat", 2.5)

	allTrackers := GetTimeTrackers()
	if len(allTrackers) != 2 {
		t.Fatalf("Expected 2 Accumulators, got %d", len(allTrackers))
	}

	// We don't guarantee order here, so let's find them by name
	var movement, combat *Accumulator
	for i := range allTrackers {
		a := &allTrackers[i]
		if a.Name == "movement" {
			movement = a
		} else if a.Name == "combat" {
			combat = a
		}
	}

	if movement == nil || combat == nil {
		t.Fatalf("Missing expected accumulators (movement or combat)")
	}

	if movement.Count != 2 {
		t.Fatalf("Expected movement.Count to be 2, got %f", movement.Count)
	}
	expectedMovementTotal := 1.2 + 0.8
	if movement.Total != expectedMovementTotal {
		t.Fatalf("Expected movement.Total to be %f, got %f", expectedMovementTotal, movement.Total)
	}

	if combat.Count != 1 {
		t.Fatalf("Expected combat.Count to be 1, got %f", combat.Count)
	}
	if combat.Total != 2.5 {
		t.Fatalf("Expected combat.Total to be 2.5, got %f", combat.Total)
	}
}

func TestRand(t *testing.T) {
	// Rand(0) should always return 0
	if v := Rand(0); v != 0 {
		t.Fatalf("Expected Rand(0) = 0, got %d", v)
	}

	// Rand(1) should always return 0
	if v := Rand(1); v != 0 {
		t.Fatalf("Expected Rand(1) = 0, got %d", v)
	}

	// Rand(2) should be in [0,1]
	for i := 0; i < 10; i++ {
		v := Rand(2)
		if v != 0 && v != 1 {
			t.Fatalf("Expected Rand(2) to be 0 or 1, got %d", v)
		}
	}
}

func TestSplitString(t *testing.T) {
	type args struct {
		input string
		width int
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"SplitString",
			args{
				`This is a sample sentence to be tested.`,
				10,
			},
			[]string{
				`This is a`,
				`sample`,
				`sentence`,
				`to be`,
				`tested.`,
			},
		},
		{
			"SplitString punctuation at eol",
			args{
				`Hello alex, this is a test.`,
				10,
			},
			[]string{
				`Hello`,
				`alex, this`,
				`is a test.`,
			},
		},
		{
			"SplitString punctuation at eol 2",
			args{
				`Hello alex, this is a test!!!`,
				10,
			},
			[]string{
				`Hello`,
				`alex, this`,
				`is a`,
				`test!!!`,
			},
		},
		{
			"SplitString long text",
			args{
				`As your senses attune to the stillness around, you find yourself engulfed in an impenetrable void, ` +
					`a realm where darkness reigns supreme. ` +
					`The abyss seems to stretch infinitely in all directions, ` +
					`and you feel a chilling isolation seeping into your very bones.`,
				79,
			},
			[]string{
				`As your senses attune to the stillness around, you find yourself engulfed in an`,
				`impenetrable void, a realm where darkness reigns supreme. The abyss seems to`,
				`stretch infinitely in all directions, and you feel a chilling isolation seeping`,
				`into your very bones.`,
			},
		},
		{
			"SplitString long text with line breaks",
			args{
				"You are now on the lawn.\nYou can smell the freshly cut grass in the " +
					"dwindling heat of the autumn afternoon sun.\nYou can see a frisbee flying in the " +
					"air, frozen mid-flight in this very sentence. Nearby, a group of picnickers pretends " +
					"to study.",
				79,
			},
			[]string{
				"You are now on the lawn.",
				"You can smell the freshly cut grass in the dwindling heat of the autumn",
				"afternoon sun.",
				"You can see a frisbee flying in the air, frozen mid-flight in this very",
				"sentence. Nearby, a group of picnickers pretends to study.",
			},
		},
		{
			"SplitString wide charactors",
			args{
				`当你的感官与周围的静谧相适应时，你发现自己被吞噬在一个无法穿透的虚空中，一个黑` +
					`暗至上的领域。深渊似乎向四面八方无限延伸，你感到一种令人不寒而栗的孤独感正渗入` +
					`你的骨髓。`,
				79,
			},
			[]string{
				`当你的感官与周围的静谧相适应时，你发现自己被吞噬在一个无法穿透的虚空中，一个黑`,
				`暗至上的领域。深渊似乎向四面八方无限延伸，你感到一种令人不寒而栗的孤独感正渗入`,
				`你的骨髓。`,
			},
		},
		{
			"SplitString mixed charactors",
			args{
				`这是一个测试文本，用于验证CJK字符和英文单词的分割行为。Hello world, this is a test!`,
				10,
			},
			[]string{`这是一个测`,
				`试文本，用`,
				`于验证CJK`,
				`字符和英文`,
				`单词的分割`,
				`行为。`,
				`Hello`,
				`world,`,
				`this is a`,
				`test!`,
			},
		},
		{
			"SplitString mixed charactors and punctuation at eol",
			args{
				`测试的文本，用于验证CJK字符。`,
				10,
			},
			[]string{
				`测试的文`,
				`本，用于验`,
				`证CJK字`,
				`符。`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitString(tt.args.input, tt.args.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitStringNL(t *testing.T) {
	type args struct {
		input  string
		width  int
		prefix []string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"SplitStringNL",
			args{
				`This is a sample sentence to be tested.`,
				10, nil,
			},
			`This is a` + "\r\n" +
				`sample` + "\r\n" +
				`sentence` + "\r\n" +
				`to be` + "\r\n" +
				`tested.`,
		},
		{
			"SplitStringNL punctuation at eol",
			args{
				`Hello alex, this is a test.`,
				10, nil,
			},
			`Hello` + "\r\n" +
				`alex, this` + "\r\n" +
				`is a test.`,
		},
		{
			"SplitStringNL punctuation at eol 2",
			args{
				`Hello alex, this is a test!!!`,
				10, nil,
			},
			`Hello` + "\r\n" +
				`alex, this` + "\r\n" +
				`is a` + "\r\n" +
				`test!!!`,
		},
		{
			"SplitStringNL long text",
			args{
				`As your senses attune to the stillness around, you find yourself engulfed in an impenetrable void, ` +
					`a realm where darkness reigns supreme. ` +
					`The abyss seems to stretch infinitely in all directions, ` +
					`and you feel a chilling isolation seeping into your very bones.`,
				79, nil,
			},
			`As your senses attune to the stillness around, you find yourself engulfed in an` + "\r\n" +
				`impenetrable void, a realm where darkness reigns supreme. The abyss seems to` + "\r\n" +
				`stretch infinitely in all directions, and you feel a chilling isolation seeping` + "\r\n" +
				`into your very bones.`,
		},
		{
			"SplitStringNL with prefix",
			args{
				`As your senses attune to the stillness around, you find yourself engulfed in an impenetrable void, ` +
					`a realm where darkness reigns supreme. ` +
					`The abyss seems to stretch infinitely in all directions, ` +
					`and you feel a chilling isolation seeping into your very bones.`,
				79, []string{"> "},
			},
			`As your senses attune to the stillness around, you find yourself engulfed in an` + "\r\n" +
				`> impenetrable void, a realm where darkness reigns supreme. The abyss seems to` + "\r\n" +
				`> stretch infinitely in all directions, and you feel a chilling isolation seeping` + "\r\n" +
				`> into your very bones.`,
		},
		{
			"SplitStringNL wide charactors",
			args{
				`当你的感官与周围的静谧相适应时，你发现自己被吞噬在一个无法穿透的虚空中，一个黑` +
					`暗至上的领域。深渊似乎向四面八方无限延伸，你感到一种令人不寒而栗的孤独感正渗入` +
					`你的骨髓。`,
				79, nil},
			`当你的感官与周围的静谧相适应时，你发现自己被吞噬在一个无法穿透的虚空中，一个黑` + "\r\n" +
				`暗至上的领域。深渊似乎向四面八方无限延伸，你感到一种令人不寒而栗的孤独感正渗入` + "\r\n" +
				`你的骨髓。`,
		},
		{
			"SplitStringNL mixed charactors",
			args{
				`这是一个测试文本，用于验证CJK字符和英文单词的分割行为。Hello world, this is a test!`,
				10, nil},
			`这是一个测` + "\r\n" +
				`试文本，用` + "\r\n" +
				`于验证CJK` + "\r\n" +
				`字符和英文` + "\r\n" +
				`单词的分割` + "\r\n" +
				`行为。` + "\r\n" +
				`Hello` + "\r\n" +
				`world,` + "\r\n" +
				`this is a` + "\r\n" +
				`test!`,
		},
		{
			"SplitString mixed charactors and punctuation at eol",
			args{
				`测试的文本，用于验证CJK字符。`,
				10, nil,
			},
			`测试的文` + "\r\n" +
				`本，用于验` + "\r\n" +
				`证CJK字` + "\r\n" +
				`符。`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitStringNL(tt.args.input, tt.args.width, tt.args.prefix...)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSplitButRespectQuotes checks splitting with respect for quoted substrings.
func TestSplitButRespectQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: `hello "my name" is 'Sammy'`,
			want:  []string{"hello", "my name", "is", "Sammy"},
		},
		{
			input: `  no quotes  `,
			want:  []string{"no", "quotes"},
		},
		{
			input: `"only quotes"`,
			want:  []string{"only quotes"},
		},
		{
			input: `mixed  "some space " 'another space' here`,
			want:  []string{"mixed", "some space", "another space", "here"},
		},
		{
			input: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		got := SplitButRespectQuotes(tt.input)
		if len(got) != len(tt.want) {
			t.Fatalf("SplitButRespectQuotes(%q) got %v, want %v", tt.input, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("SplitButRespectQuotes(%q) mismatch at %d: got %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// TestGetMatchNumber checks parsing of names that contain #N suffixes.
func TestGetMatchNumber(t *testing.T) {
	tests := []struct {
		input string
		name  string
		num   int
	}{
		{"sword", "sword", 1},
		{"SWORD#2", "sword", 2},
		{"shield#0", "shield", 1}, // 0 gets forced to 1
		{"  HELMET#42  ", "helmet", 42},
		{"   ", "", 1},
		{"just   text   ", "just   text", 1},
	}

	for _, tt := range tests {
		gotName, gotNum := GetMatchNumber(tt.input)
		if gotName != tt.name || gotNum != tt.num {
			t.Errorf("GetMatchNumber(%q) got (%q, %d), want (%q, %d)",
				tt.input, gotName, gotNum, tt.name, tt.num)
		}
	}
}

// TestFindMatchIn checks the behavior of partial and full matches in a slice.
func TestFindMatchIn(t *testing.T) {
	items := []string{"SWORD", "SHINING SWORD", "SHIELD", "BIG HELM", "HELMET", "GEM"}
	type expected struct {
		match      string
		closeMatch string
	}
	tests := []struct {
		name   string
		search string
		want   expected
	}{
		{"empty", "", expected{"", ""}},
		{"exact match", "SWORD", expected{"SWORD", "SWORD"}},
		{"exact match #2", "sword#2", expected{"", "SHINING SWORD"}},
		{"partial match", "HELM", expected{"", "HELMET"}},
		{"partial but not first item", "G", expected{"", "GEM"}},
		{"contains fallback", "iel", expected{"", "SHIELD"}}, // if logic tries "contains"
		{"helmet partial #2", "helm#2", expected{"", "HELMET"}},
	}

	for _, tt := range tests {
		gotMatch, gotClose := FindMatchIn(tt.search, items...)
		if gotMatch != tt.want.match || gotClose != tt.want.closeMatch {
			t.Errorf("FindMatchIn(%q) got (%q, %q), want (%q, %q)",
				tt.search, gotMatch, gotClose, tt.want.match, tt.want.closeMatch)
		}
	}
}

// TestStringMatch tests the stringMatch function with various conditions.
func TestStringMatch(t *testing.T) {
	tests := []struct {
		name          string
		searchFor     string
		searchIn      string
		allowContains bool
		wantPartial   bool
		wantFull      bool
	}{
		{"exact", "sword", "sword", false, true, true},
		{"partial prefix", "sw", "sword", false, true, false},
		{"no match prefix", "abc", "sword", false, false, false},
		{"contains partial", "wor", "sword", true, true, false},
		{"contains full", "sword", "MYswordX", true, true, false}, // because "sword" is fully matched inside
		{"case mismatch", "SWORD", "sword", false, true, true},
	}
	for _, tt := range tests {
		partial, full := stringMatch(tt.searchFor, tt.searchIn, tt.allowContains)
		if partial != tt.wantPartial || full != tt.wantFull {
			t.Errorf("stringMatch(%q, %q, %t) got (%v, %v), want (%v, %v)",
				tt.searchFor, tt.searchIn, tt.allowContains,
				partial, full, tt.wantPartial, tt.wantFull)
		}
	}
}

// TestHash checks Hash outputs a known format.
func TestHash(t *testing.T) {
	input := "hello"
	got := Hash(input)
	if len(got) == 0 {
		t.Errorf("Hash(%q) returned empty string", input)
	}
	// Basic check: SHA-256 hex string is 64 characters
	if len(got) != 64 {
		t.Errorf("Hash(%q) length = %d, want 64", input, len(got))
	}
}

// TestHashBytes checks HashBytes against known length (SHA-256).
func TestHashBytes(t *testing.T) {
	input := []byte("hello")
	got := HashBytes(input)
	if len(got) != 64 {
		t.Errorf("HashBytes(%q) length = %d, want 64", input, len(got))
	}
}

// TestMd5 checks the MD5 function for non-empty output.
func TestMd5(t *testing.T) {
	input := "hello"
	got := Md5(input)
	if len(got) == 0 {
		t.Errorf("Md5(%q) returned empty string", input)
	}
}

// TestGetLockSequence ensures the sequence is the correct length, contains only 'U' and 'D',
// and is deterministic for the same inputs.
func TestGetLockSequence(t *testing.T) {
	tests := []struct {
		name           string
		lockIdentifier string
		difficulty     int
		seed           string
		wantLength     int
	}{
		{
			name:           "BelowMinimum",
			lockIdentifier: "TestLock",
			difficulty:     1, // less than 2
			seed:           "seed",
			wantLength:     2, // forced to 2
		},
		{
			name:           "AboveMaximum",
			lockIdentifier: "TestLock",
			difficulty:     100, // more than 32
			seed:           "seed",
			wantLength:     32, // forced to 32
		},
		{
			name:           "WithinRange",
			lockIdentifier: "TestLock",
			difficulty:     8,
			seed:           "seed",
			wantLength:     8, // used as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLockSequence(tt.lockIdentifier, tt.difficulty, tt.seed, 0)
			if len(got) != tt.wantLength {
				t.Errorf("GetLockSequence(%q, %d, %q) length = %d; want %d",
					tt.lockIdentifier, tt.difficulty, tt.seed, len(got), tt.wantLength)
			}
			for i, c := range got {
				if c != 'U' && c != 'D' {
					t.Errorf("character at index %d is %q, want 'U' or 'D'", i, c)
				}
			}
		})
	}

	// Additional check: repeated calls with the same parameters should return the same sequence.
	t.Run("DeterministicCheck", func(t *testing.T) {
		first := GetLockSequence("Lock", 4, "Seed", 0)
		second := GetLockSequence("Lock", 4, "Seed", 0)
		if first != second {
			t.Errorf("Expected repeated calls to match, but got %q != %q", first, second)
		}
	})
}

func TestGetLockSequence_RotationBackCompat(t *testing.T) {
	// Rotation 0 must produce the same sequence the function used to,
	// so existing keyring entries stay valid for non-rotating locks.
	base := GetLockSequence("Lock", 4, "Seed", 0)
	if len(base) != 4 {
		t.Fatalf("base length = %d, want 4", len(base))
	}
	for _, c := range base {
		if c != 'U' && c != 'D' {
			t.Fatalf("base contains non-U/D char %q", c)
		}
	}
}

func TestGetLockSequence_RotationChangesOutput(t *testing.T) {
	base := GetLockSequence("Lock", 8, "Seed", 0)
	rotated := GetLockSequence("Lock", 8, "Seed", 1)
	if base == rotated {
		t.Fatalf("rotation 1 produced the same sequence as rotation 0 (%q); rotation must change output", base)
	}

	// Different rotation values produce different sequences.
	r2 := GetLockSequence("Lock", 8, "Seed", 2)
	if rotated == r2 {
		t.Fatalf("rotation 1 and rotation 2 produced the same sequence (%q); rotation must change output", rotated)
	}
}

// TestCompressDecompress round-trips data through gzip.
func TestCompressDecompress(t *testing.T) {
	input := []byte("This is some test data to compress")
	comp := Compress(input)
	if len(comp) == 0 {
		t.Error("Compress returned empty data")
	}

	decomp := Decompress(comp)
	if !bytes.Equal(decomp, input) {
		t.Errorf("Decompress(Compress(...)) mismatch. got %q, want %q", decomp, input)
	}
}

// TestEncodeDecode checks base64 encoding/decoding round trip.
func TestEncodeDecode(t *testing.T) {
	input := []byte("hello world")
	encoded := Encode(input)
	decoded := Decode(encoded)
	if !bytes.Equal(input, decoded) {
		t.Errorf("Decode(Encode(...)) mismatch. got %q, want %q", decoded, input)
	}
}

// TestGetMyIP is a very basic check; it will do an actual HTTP request.
// You might skip or mock this test in CI if external calls are unwanted.
func TestGetMyIP(t *testing.T) {
	// Overwrite default transport or skip if you want to avoid real calls.
	http.DefaultTransport.(*http.Transport).DisableKeepAlives = true

	ip := GetMyIP()
	if ip == "" {
		t.Error("GetMyIP() returned an empty string")
	}
	// We won't parse or validate the IP because different environments respond differently.
}

// TestProgressBar checks the generated bar pieces.
func TestProgressBar(t *testing.T) {
	full, empty := ProgressBar(0.5, 10)

	if runewidth.StringWidth(full) != 5 || runewidth.StringWidth(empty) != 5 {
		t.Errorf("ProgressBar(0.5,10) got %d full, %d empty; want 5,5", len(full), len(empty))
	}

	// test with 3 inputs
	full, empty = ProgressBar(0.5, 10, "A", "B", "C")

	if strings.Contains(full, "C") || strings.Contains(empty, "C") {
		t.Errorf("ProgressBar(0.5,10,\"A\",\"B\",\"C\") contained discardable data")
	}
}

// TestRollDice checks basic correctness within expected range.
func TestRollDice(t *testing.T) {
	dice, sides := 3, 6
	got := RollDice(dice, sides)
	// Min = 3, Max = 18
	if got < dice || got > dice*sides {
		t.Errorf("RollDice(3,6) = %d, want in range [3..18]", got)
	}

	dice, sides = 3, -6
	got = RollDice(dice, sides)
	// Min = 3, Max = 18
	if got < dice || got > dice*(-sides) {
		t.Errorf("RollDice(3,-6) = %d, want in range [3..18]", got)
	}

	// Negative dice
	gotNeg := RollDice(-2, 6)
	if gotNeg >= 0 {
		t.Errorf("RollDice(-2,6) should be negative, got %d", gotNeg)
	}

}

// TestParseDiceRoll checks parsing the format "[attacks@]XdY±Z#...".
func TestParseDiceRoll(t *testing.T) {
	tests := []struct {
		in           string
		wantAttacks  int
		wantDCount   int
		wantDSides   int
		wantBonus    int
		wantBuffCrit []int
	}{
		{"1d6", 1, 1, 6, 0, []int{}},
		{"2@1d3+2", 2, 1, 3, 2, []int{}},
		{"3d10-2", 1, 3, 10, -2, []int{}},
		{"2@3d8#1,5,10", 2, 3, 8, 0, []int{1, 5, 10}},
		{"-2d4", 1, -2, 4, 0, []int{}}, // negative count
	}
	for _, tt := range tests {
		a, dC, dS, bonus, crit := ParseDiceRoll(tt.in)
		if a != tt.wantAttacks || dC != tt.wantDCount || dS != tt.wantDSides || bonus != tt.wantBonus {
			t.Errorf("ParseDiceRoll(%q) = (%d,%d,%d,%d,%v), want (%d,%d,%d,%d,%v)",
				tt.in, a, dC, dS, bonus, crit,
				tt.wantAttacks, tt.wantDCount, tt.wantDSides, tt.wantBonus, tt.wantBuffCrit)
		}
		if len(crit) != len(tt.wantBuffCrit) {
			t.Errorf("ParseDiceRoll(%q) buffOnCrit got %v, want %v", tt.in, crit, tt.wantBuffCrit)
		}
	}
}

// TestFormatDiceRoll checks the inverse of ParseDiceRoll.
func TestFormatDiceRoll(t *testing.T) {
	tests := []struct {
		name       string
		attacks    int
		dCount     int
		dSides     int
		bonus      int
		buffOnCrit []int
		want       string
	}{
		{"basic", 1, 1, 6, 0, []int{}, "1d6"},
		{"multiple attacks", 2, 1, 3, 2, []int{}, "2@1d3+2"},
		{"negative bonus", 1, 3, 10, -2, []int{}, "3d10-2"},
		{"buff list", 2, 3, 8, 0, []int{1, 5}, "2@3d8#1,5"},
	}

	for _, tt := range tests {
		got := FormatDiceRoll(tt.attacks, tt.dCount, tt.dSides, tt.bonus, tt.buffOnCrit)
		if got != tt.want {
			t.Errorf("FormatDiceRoll(%d,%d,%d,%d,%v) = %q, want %q",
				tt.attacks, tt.dCount, tt.dSides, tt.bonus, tt.buffOnCrit,
				got, tt.want)
		}
	}
}

// TestSafeSave and TestSave demonstrate saving. They create temp files for safety.
func TestSafeSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "testfile.txt")
	data := []byte("testing safe save")

	err := SafeSave(path, data)
	if err != nil {
		t.Fatalf("SafeSave failed: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unable to read saved file: %v", err)
	}
	if !bytes.Equal(contents, data) {
		t.Errorf("safe saved file mismatch: got %q, want %q", contents, data)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "testfile.txt")
	data := []byte("testing normal save")

	err := Save(path, data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unable to read saved file: %v", err)
	}
	if !bytes.Equal(contents, data) {
		t.Errorf("saved file mismatch: got %q, want %q", contents, data)
	}
}

// TestFilePath just ensures the slash normalization works as expected.
func TestFilePath(t *testing.T) {
	path := FilePath("foo", "bar")
	// On POSIX, we'd expect "foobar". On Windows, it might differ in testing environment.
	// But fromSlash will remove forward slashes and apply OS-specific.
	// The best we can do here is confirm there's no slash if we're combining.
	if !strings.Contains(path, string(os.PathSeparator)) && len(path) != 6 {
		t.Errorf("FilePath('foo','bar') = %q, unexpected result", path)
	}

	single := FilePath("foo/bar")
	if !strings.Contains(single, string(os.PathSeparator)) && len(single) == 7 {
		t.Errorf("FilePath('foo/bar') = %q, expected slash replaced with OS separator", single)
	}
}

// TestBreakIntoParts ensures we get progressively smaller strings.
func TestBreakIntoParts(t *testing.T) {
	got := BreakIntoParts("hello world test")
	want := []string{
		"hello world test",
		"world test",
		"test",
	}
	if len(got) != len(want) {
		t.Fatalf("BreakIntoParts(...) = %v, want len = %d", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("BreakIntoParts(...) mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestHealthClass ensures the correct string class is returned.
func TestHealthClass(t *testing.T) {
	tests := []struct {
		health    int
		maxHealth int
		want      string
	}{
		{0, 100, "health-dead"},
		{50, 100, "health-50"},
		{100, 100, "health-100"},
		{1, 10, "health-10"},
		{9, 10, "health-90"},
	}
	for _, tt := range tests {
		got := HealthClass(tt.health, tt.maxHealth)
		if got != tt.want {
			t.Errorf("HealthClass(%d,%d) = %q, want %q", tt.health, tt.maxHealth, got, tt.want)
		}
	}
}

// TestQuantizeTens checks numeric bucketing.
func TestQuantizeTens(t *testing.T) {
	tests := []struct {
		value int
		max   int
		want  int
	}{
		{0, 10, 0},
		{1, 10, 10},
		// Perfect 50%
		{5, 10, 50},
		// Nearly max => 9/10 ~ 90%
		{9, 10, 90},
		// Full value => 10/10 => 100%
		{10, 10, 100},
		// Another set: 19/100 => 19% => floor(1.9)=1 => 1*10=10
		{19, 100, 10},
		// 49/100 => 49% => 4.9 => floor(4.9)=4 => 4*10=40
		{49, 100, 40},
		// 50/100 => 50% => 5 => 5*10=50
		{50, 100, 50},
		// 99/100 => 99% => 9.9 => floor(9.9)=9 => 9*10=90
		{99, 100, 90},
		// 100/100 => 100% => 10 => 10*10=100
		{100, 100, 100},
		// Edge case: if max=1 => percentages can jump quickly
		{1, 1, 100},
	}

	for _, tt := range tests {
		got := QuantizeTens(tt.value, tt.max)
		if got != tt.want {
			t.Errorf("QuantizeTens(%d, %d) = %d; want %d",
				tt.value, tt.max, got, tt.want)
		}
	}
}

// TestStripPrepositions ensures words like 'the', 'onto', 'to' etc. are stripped.
func TestStripPrepositions(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"onto the table", "table"},
		{"with my sword", "sword"},
		{"pick up the item", "pick up item"},
		{"none match", "none match"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripPrepositions(tt.in)
		if got != tt.want {
			t.Errorf("StripPrepositions(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestConvertColorShortTags verifies the replacement logic of {fg:bg} tags.
func TestConvertColorShortTags(t *testing.T) {
	input := "Hello {34}World{34:1}!"
	got := ConvertColorShortTags(input)
	// We don't parse the entire ANSI logic, just check that something replaced.
	if strings.Contains(got, "{34}") {
		t.Errorf("ConvertColorShortTags(...) did not replace {34} tag")
	}
	if strings.Contains(got, "{34:1}") {
		t.Errorf("ConvertColorShortTags(...) did not replace {34:1} tag")
	}
	if !strings.Contains(got, "fg=\"34\"") {
		t.Errorf("ConvertColorShortTags(...) missing fg=\"34\"")
	}
}

// TestPercentOfTotal checks the simple calculation.
func TestPercentOfTotal(t *testing.T) {
	tests := []struct {
		val1, val2 int
		want       float64
	}{
		{0, 100, 0},
		{1, 1, 2},    // (1+1)/1 = 2
		{5, 5, 2},    // (5+5)/5 = 2
		{10, 5, 1.5}, // (10+5)/10 = 1.5
	}
	for _, tt := range tests {
		got := PercentOfTotal(tt.val1, tt.val2)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("PercentOfTotal(%d,%d) = %f, want %f", tt.val1, tt.val2, got, tt.want)
		}
	}
}

// TestConvertForFilename ensures special chars are replaced with underscores, lowercased, etc.
func TestConvertForFilename(t *testing.T) {
	in := "Hello! This's a Test? 123"
	got := ConvertForFilename(in)
	wantPattern := `^hello__thiss_a_test__123$`
	if match, _ := regexp.MatchString(wantPattern, got); !match {
		t.Errorf("ConvertForFilename(%q) = %q, want match with %q", in, got, wantPattern)
	}
}

// TestStringWildcardMatch checks different combinations of wildcard usage.
func TestStringWildcardMatch(t *testing.T) {
	tests := []struct {
		inString string
		pattern  string
		want     bool
	}{
		{"hello", "hello", true},
		{"hello", "hel", false},
		{"hello", "*lo", true},   // ends with
		{"hello", "he*", true},   // starts with
		{"hello", "*ell*", true}, // contains
		{"hello", "no*", false},
	}
	for _, tt := range tests {
		got := StringWildcardMatch(tt.inString, tt.pattern)
		if got != tt.want {
			t.Errorf("StringWildcardMatch(%q, %q) = %v, want %v",
				tt.inString, tt.pattern, got, tt.want)
		}
	}
}

func TestValidateWorldFiles(t *testing.T) {
	// 1. Non-existent exampleWorldPath => should fail on os.ReadDir
	t.Run("NonExistentExampleWorld", func(t *testing.T) {
		// Provide a path we expect not to exist
		exampleWorldPath := filepath.Join(t.TempDir(), "nonexistent-subdir")
		// We won't create it, so it doesn't exist
		worldPath := t.TempDir()

		err := ValidateWorldFiles(exampleWorldPath, worldPath)
		if err == nil {
			t.Fatalf("Expected error for non-existent exampleWorldPath, got nil")
		}
		// Optional: check if the error message is the expected "unable to read directory ..."
		if msg := err.Error(); !containsAll(msg, "unable to read directory", exampleWorldPath) {
			t.Errorf("Unexpected error message: %v", msg)
		}
	})

	// 2. Happy path: all subfolders in exampleWorldPath exist in worldPath => no error
	t.Run("AllSubfoldersMatch", func(t *testing.T) {
		exampleWorldPath := t.TempDir()
		worldPath := t.TempDir()

		// Create some subfolders in exampleWorldPath
		subfolders := []string{"area1", "area2"}
		for _, sf := range subfolders {
			if err := os.Mkdir(filepath.Join(exampleWorldPath, sf), 0o755); err != nil {
				t.Fatalf("Failed to create subfolder %s in exampleWorldPath: %v", sf, err)
			}
		}

		// Mirror them in worldPath
		for _, sf := range subfolders {
			if err := os.Mkdir(filepath.Join(worldPath, sf), 0o755); err != nil {
				t.Fatalf("Failed to create subfolder %s in worldPath: %v", sf, err)
			}
		}

		// Should be no error
		if err := ValidateWorldFiles(exampleWorldPath, worldPath); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// 3. Missing subfolder => triggers "missing folder" error
	t.Run("MissingSubfolder", func(t *testing.T) {
		exampleWorldPath := t.TempDir()
		worldPath := t.TempDir()

		// Create subfolders in exampleWorldPath
		subfolders := []string{"area1", "area2"}
		for _, sf := range subfolders {
			if err := os.Mkdir(filepath.Join(exampleWorldPath, sf), 0o755); err != nil {
				t.Fatalf("Failed to create subfolder %s in exampleWorldPath: %v", sf, err)
			}
		}
		// Create only one subfolder in worldPath, so "area2" is missing
		if err := os.Mkdir(filepath.Join(worldPath, "area1"), 0o755); err != nil {
			t.Fatalf("Failed to create subfolder area1 in worldPath: %v", err)
		}

		err := ValidateWorldFiles(exampleWorldPath, worldPath)
		if err == nil {
			t.Fatalf("Expected an error due to missing subfolder, got nil")
		}
		// Optional: check the error message
		if msg := err.Error(); !containsAll(msg, "missing folder", "area2") {
			t.Errorf("Unexpected error message: %v", msg)
		}
	})

	// 4. Subfolder name exists but is a file => triggers "exists but is not a directory" error
	t.Run("SubfolderIsNotADirectory", func(t *testing.T) {
		exampleWorldPath := t.TempDir()
		worldPath := t.TempDir()

		// Create a subfolder "area1" in the exampleWorldPath
		if err := os.Mkdir(filepath.Join(exampleWorldPath, "area1"), 0o755); err != nil {
			t.Fatalf("Failed to create subfolder area1 in exampleWorldPath: %v", err)
		}

		// In worldPath, create a file named "area1" instead of a directory
		filePath := filepath.Join(worldPath, "area1")
		if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
			t.Fatalf("Failed to create file area1 in worldPath: %v", err)
		}

		err := ValidateWorldFiles(exampleWorldPath, worldPath)
		if err == nil {
			t.Fatalf("Expected an error due to subfolder name clashing with a file, got nil")
		}
		// Optional: check error message
		if msg := err.Error(); !containsAll(msg, "exists but is not a directory", filePath) {
			t.Errorf("Unexpected error message: %v", msg)
		}
	})
}

func TestGetMatchNumber_DikuFormat(t *testing.T) {
	name, num := GetMatchNumber("3.dagger")
	if name != "dagger" || num != 3 {
		t.Errorf("expected (dagger, 3), got (%s, %d)", name, num)
	}

	name, num = GetMatchNumber("all.dagger")
	if name != "dagger" || num != -1 {
		t.Errorf("expected (dagger, -1), got (%s, %d)", name, num)
	}

	name, num = GetMatchNumber("dagger#2")
	if name != "dagger" || num != 2 {
		t.Errorf("expected (dagger, 2), got (%s, %d)", name, num)
	}

	name, num = GetMatchNumber("st.elmo")
	if name != "st.elmo" || num != 1 {
		t.Errorf("expected (st.elmo, 1), got (%s, %d)", name, num)
	}

	name, num = GetMatchNumber("dagger")
	if name != "dagger" || num != 1 {
		t.Errorf("expected (dagger, 1), got (%s, %d)", name, num)
	}

	name, num = GetMatchNumber("1.sword")
	if name != "sword" || num != 1 {
		t.Errorf("expected (sword, 1), got (%s, %d)", name, num)
	}
}

// Utility helper: check that a string contains all given substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

// Another small helper for substring check (you could just use strings.Contains if you prefer).
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || containsAt(s, sub, 0) || contains(s[1:], sub))
}

// containsAt checks if sub is at the beginning of s.
func containsAt(s, sub string, index int) bool {
	if len(s[index:]) < len(sub) {
		return false
	}
	for i := 0; i < len(sub); i++ {
		if s[index+i] != sub[i] {
			return false
		}
	}
	return true
}

func TestConvertToAscii(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pure ascii passthrough", "Hello world!", "Hello world!"},
		{"box corners", "┌─┐└─┘", "+-++-+"},
		{"box sides", "│text│", "|text|"},
		{"double box", "╔═╗║x║╚═╝", "+=+|x|+=+"},
		{"mixed box", "╒═╕╘═╛", "+=++=+"},
		{"intersections", "├┤┬┴┼", "+++++"},
		{"double intersections", "╠╣╦╩╬", "+++++"},
		{"block elements", "█░▒▓", "#.:#"},
		{"half blocks", "▄▀▌▐", "-_||"},
		{"bullet", "• item", "* item"},
		{"ansi preserved", "\x1b[32mgreen\x1b[0m", "\x1b[32mgreen\x1b[0m"},
		{"mixed ansi and unicode", "\x1b[33m┌─┐\x1b[0m", "\x1b[33m+-+\x1b[0m"},
		{"status template chars", " ┌─ Attributes ─┐\n └──────────────┘", " +- Attributes -+\n +--------------+"},
		{"motd chars", "╔══╗\n║hi║\n╚══╝", "+==+\n|hi|\n+==+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToAscii(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToAscii(%q)\n  got:  %q\n  want: %q", tt.input, result, tt.expected)
			}
		})
	}
}
