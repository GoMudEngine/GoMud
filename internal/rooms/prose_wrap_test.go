package rooms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// room5224Yaml is a verbatim copy of
// _datafiles/world/dogmud/rooms/pothole_coulee/5224.yaml as authored. It is the
// case that exposed the defect: a folded `description`, two long wrapped
// `nouns`, an `idlemessages` entry carrying ansi markup, a `mutators` block and
// a dead legacy `coord:` key. Copied into the test so the suite never touches
// the real world file.
const room5224Yaml = `roomid: 5224
zone: Pothole Coulee
title: Stargazer Cut
description: >
  A narrow cut climbs north through the cliffs toward the steppe rim,
  where a broken silhouette stands black against the sky -- a ruin, or the
  bones of one. The air in the cut sits strange and heavy, folded somehow,
  as though the distance had been pinched shorter than the eye allows.
  Loose stone has been cleared from the path lately, the way still raw
  where hands have widened it.
biome: cliffs
coord:
  x: 44
  y: -2
  z: 0
mutators:
- mutatorid: sanctuary
exits:
  south:
    roomid: 5218
  up:
    roomid: 5302
nouns:
  cut: The cut is a narrow cleft worked up through the cliff toward the
    rim, its walls close enough to touch on either hand. The stone here
    has the raw, pale look of recent clearing, where blocks have been
    levered out to widen the climb. The air in it hangs heavy and folded,
    pressing at you in a way plain air does not.
  ruin: Up on the steppe rim a broken shape stands against the sky, too
    far and too weathered to read -- a ruin, leaning walls and a fallen
    arch, the bones of something old. The fold in the air seems to gather
    around it, as though the place bent the distance to itself.
idlemessages:
- For a breath the air in the <ansi fg="itemname">cut</ansi> seems to pull taut, then eases.
x: 43
y: 17
`

// assertWithin80Columns pins the repo-wide 80 column convention on the saved
// file. Breaking it is the visible half of the defect.
func assertWithin80Columns(t *testing.T, data []byte) {
	t.Helper()
	for i, line := range strings.Split(string(data), "\n") {
		assert.LessOrEqualf(t, len(line), 80,
			"line %d is %d columns: %s", i+1, len(line), line)
	}
}

// TestSaveRoomTemplate_ProseRoundTripsAndStaysWrapped is the centrepiece.
//
// The hard requirement is round-trip fidelity: load(save(room)) must give back
// prose byte-for-byte identical to what went in. Folded (`>`) scalars are the
// only wrapping style that does this, because reading one rejoins the wrapped
// lines with single spaces and recovers the original string. A literal (`|`)
// scalar would bake hard newlines into the value and silently change what
// players see, which is why this test asserts equality of the VALUES and not
// merely the shape of the file.
func TestSaveRoomTemplate_ProseRoundTripsAndStaysWrapped(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	var authored Room
	require.NoError(t, yaml.Unmarshal([]byte(room5224Yaml), &authored))

	// Sanity: loading a folded scalar joins its lines, so the in-memory value
	// is one long line. That is the input the save path has to re-wrap.
	require.NotContains(t, strings.TrimSuffix(authored.Description, "\n"), "\n",
		"test premise: a folded description loads as a single joined line")
	require.Greater(t, len(authored.Description), 300)

	tempDir := useTempDataFiles(t, false)

	roomManager.zones[authored.Zone] = &ZoneConfig{
		Name:    authored.Zone,
		RoomId:  authored.RoomId,
		RoomIds: map[int]struct{}{},
	}

	zoneFolder := ZoneToFolder(authored.Zone)
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "rooms", zoneFolder), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "rooms.instances", zoneFolder), 0755))

	require.NoError(t, SaveRoomTemplate(authored))

	savedPath := filepath.Join(tempDir, "rooms", zoneFolder, "5224.yaml")
	saved, err := os.ReadFile(savedPath)
	require.NoError(t, err)

	// --- the shape on disk -------------------------------------------------
	assertWithin80Columns(t, saved)

	assert.Contains(t, string(saved), "description: >\n",
		"description carries a trailing newline, so it needs clip chomping")
	assert.Contains(t, string(saved), "cut: >-\n",
		"noun values carry no trailing newline, so they need strip chomping")
	assert.NotContains(t, string(saved), "description: |",
		"a literal block would bake hard newlines into the value")

	// --- the values, which is what actually matters -------------------------
	var reloaded Room
	require.NoError(t, yaml.Unmarshal(saved, &reloaded))

	assert.Equal(t, authored.Description, reloaded.Description,
		"description must survive save+load byte for byte")
	assert.Equal(t, authored.Nouns, reloaded.Nouns,
		"noun prose must survive save+load byte for byte")
	assert.Equal(t, authored.IdleMessages, reloaded.IdleMessages,
		"idle messages carry ansi markup and must survive byte for byte")

	// --- nothing else moved -------------------------------------------------
	assert.Equal(t, authored.Title, reloaded.Title)
	assert.Equal(t, authored.Biome, reloaded.Biome)
	assert.Equal(t, authored.X, reloaded.X)
	assert.Equal(t, authored.Y, reloaded.Y)
	assert.Equal(t, authored.Exits, reloaded.Exits)
	assert.Len(t, reloaded.Mutators, 1)
}

// TestMarshalRoomTemplate_OnlyProseFieldsDiffer pins that folding is surgical.
// A room saved with no prose change must differ from the plain yaml.v2 output
// only in the prose fields, so the 1386 committed room files do not churn.
func TestMarshalRoomTemplate_OnlyProseFieldsDiffer(t *testing.T) {
	var authored Room
	require.NoError(t, yaml.Unmarshal([]byte(room5224Yaml), &authored))

	plain, err := yaml.Marshal(&authored)
	require.NoError(t, err)

	folded, err := marshalRoomTemplate(authored)
	require.NoError(t, err)

	// Sequence style is the thing a wholesale switch to yaml.v3 would have
	// changed across every room file.
	assert.Contains(t, string(plain), "\nmutators:\n- mutatorid: sanctuary")
	assert.Contains(t, string(folded), "\nmutators:\n- mutatorid: sanctuary")

	for _, unchanged := range []string{
		"roomid: 5224\n",
		"zone: Pothole Coulee\n",
		"title: Stargazer Cut\n",
		"biome: cliffs\n",
		"x: 43\n",
		"\"y\": 17\n",
		"exits:\n  south:\n    roomid: 5218\n",
	} {
		assert.Contains(t, string(plain), unchanged, "test premise")
		assert.Contains(t, string(folded), unchanged, "non-prose field was reformatted")
	}
}

// TestMarshalRoomTemplate_RoundTripFidelity walks a spread of prose shapes and
// asserts every one comes back identical, whether it ended up folded or was
// refused. Refusing to wrap is always acceptable; changing the value never is.
func TestMarshalRoomTemplate_RoundTripFidelity(t *testing.T) {
	long := "Beyond the gate the road runs on between low walls of dressed stone, " +
		"each block set close enough that no mortar shows, and the dust lies " +
		"undisturbed in the joints where nothing has passed for a season."

	cases := []struct {
		name string
		body string
	}{
		{"short", "A small room."},
		{"empty", ""},
		{"long no trailing newline", long},
		{"long with trailing newline", long + "\n"},
		{"long with ansi markup", `The <ansi fg="itemname">lantern</ansi> ` + long},
		{"long with punctuation and dashes", long + " -- and then, at last, nothing."},
		{"interior newlines", "First paragraph line.\n\nSecond paragraph line. " + long},
		{"trailing newlines doubled", long + "\n\n"},
		{"double spaces", strings.ReplaceAll(long, ", ", ",  ")},
		{"leading space", " " + long},
		{"trailing space", long + " "},
		{"tab inside", strings.Replace(long, " ", "\t", 1)},
		{"single unbreakable token", strings.ReplaceAll(long, " ", "-")},
		{"one huge token among words", "Read the " + strings.Repeat("z", 120) + " sign carefully now."},
		{"exactly at the wrap boundary", strings.Repeat("ab ", 25) + "end"},
		{"unicode", "Le vent souffle sur la lande grise, et " + long},
		{"quotes and colons", `He said "north, then down" -- but the sign reads otherwise. ` + long},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := Room{
				RoomId:       900,
				Zone:         "TestZone",
				Title:        "Round Trip",
				Description:  tc.body,
				Nouns:        map[string]string{"stone": tc.body},
				HiddenNouns:  map[string]HiddenNoun{"seam": {Description: tc.body, HiddenDescription: tc.body}},
				IdleMessages: []string{tc.body},
			}

			out, err := marshalRoomTemplate(src)
			require.NoError(t, err)

			var back Room
			require.NoError(t, yaml.Unmarshal(out, &back), "output must still parse")

			assert.Equal(t, src.Description, back.Description, "description")
			assert.Equal(t, src.Nouns["stone"], back.Nouns["stone"], "noun")
			assert.Equal(t, src.HiddenNouns["seam"].Description, back.HiddenNouns["seam"].Description, "hidden noun description")
			assert.Equal(t, src.HiddenNouns["seam"].HiddenDescription, back.HiddenNouns["seam"].HiddenDescription, "hidden noun hidden_description")
			require.Len(t, back.IdleMessages, 1)
			assert.Equal(t, src.IdleMessages[0], back.IdleMessages[0], "idle message")
		})
	}
}

// TestMarshalRoomTemplate_DoesNotMutateCaller pins that building the sentinel
// copy never writes through the shared map and slice headers of the room the
// caller handed us. SaveRoomTemplate goes on to put that same room into the
// in-memory cache, so a leaked sentinel would be shown to players.
func TestMarshalRoomTemplate_DoesNotMutateCaller(t *testing.T) {
	long := strings.Repeat("the long grey wall of the coulee runs on ", 6)

	src := Room{
		RoomId:       901,
		Zone:         "TestZone",
		Description:  long,
		Nouns:        map[string]string{"wall": long},
		HiddenNouns:  map[string]HiddenNoun{"crack": {Description: long, HiddenDescription: long}},
		IdleMessages: []string{long},
	}

	_, err := marshalRoomTemplate(src)
	require.NoError(t, err)

	assert.Equal(t, long, src.Nouns["wall"])
	assert.Equal(t, long, src.HiddenNouns["crack"].Description)
	assert.Equal(t, long, src.HiddenNouns["crack"].HiddenDescription)
	assert.Equal(t, long, src.IdleMessages[0])
	assert.NotContains(t, src.Nouns["wall"], "prosefold")
}

// TestCanFoldProse enumerates the content-level refusals. Each of these is a
// case where a folded scalar would fail to give the value back unchanged.
func TestCanFoldProse(t *testing.T) {
	body := "A narrow cut climbs north through the cliffs toward the steppe rim."

	assert.True(t, canFoldProse(body), "plain prose")
	assert.True(t, canFoldProse(body+"\n"), "one trailing newline is clip chomping")

	assert.False(t, canFoldProse(""), "empty")
	assert.False(t, canFoldProse("\n"), "newline only")
	assert.False(t, canFoldProse(body+"\n\n"), "two trailing newlines")
	assert.False(t, canFoldProse("one\ntwo"), "interior newline")
	assert.False(t, canFoldProse("one\r\ntwo"), "carriage return")
	assert.False(t, canFoldProse("one\ttwo"), "tab")
	assert.False(t, canFoldProse("one  two"), "run of spaces")
	assert.False(t, canFoldProse(" "+body), "leading space")
	assert.False(t, canFoldProse(body+" "), "trailing space")
}

// TestWrapProse covers the breaking rule itself, including the guarantee that
// rejoining the produced lines with single spaces reproduces the input exactly
// (which is what YAML folding will do on the way back in).
func TestWrapProse(t *testing.T) {
	body := "A narrow cut climbs north through the cliffs toward the steppe rim, " +
		"where a broken silhouette stands black against the sky."

	lines, ok := wrapProse(body, 40)
	require.True(t, ok)
	require.Greater(t, len(lines), 1)
	for _, l := range lines {
		assert.LessOrEqual(t, len(l), 40)
		assert.NotEqual(t, " ", string(l[0]), "a folded line starting with space is read literally")
	}
	assert.Equal(t, body, strings.Join(lines, " "), "folding must rebuild the input exactly")

	_, ok = wrapProse("short enough", 40)
	assert.False(t, ok, "a single line needs no folding")

	_, ok = wrapProse("word "+strings.Repeat("x", 60)+" word", 40)
	assert.False(t, ok, "a token wider than the line cannot be broken safely")

	_, ok = wrapProse("double  space here and some more words to force a wrap", 20)
	assert.False(t, ok, "runs of spaces are refused")

	_, ok = wrapProse(body, 5)
	assert.False(t, ok, "an absurdly narrow width is refused")
}
