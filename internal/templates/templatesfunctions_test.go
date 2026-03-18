package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPad(t *testing.T) {
	type args struct {
		width     int
		stringIn  string
		padString string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"Pad empty string with space", args{10, "", ""}, "          "},
		{"Pad empty string with minus", args{10, "", "-"}, "----------"},
		{"Pad with space", args{10, "test", ""}, "   test   "},
		{"Pad with space 2", args{10, "hello", ""}, "  hello   "},
		{"Pad with minus", args{10, "test", "-"}, "---test---"},
		{"Pad with space and zero width", args{0, "test", ""}, "test"},
		{"Pad with space and smaller width", args{3, "test", ""}, "test"},
		{"Pad with space and same width", args{4, "test", ""}, "test"},
		{"Pad wide charactors with space", args{10, "宽字符", ""}, "  宽字符  "},
		{"Pad wide charactors with space 2", args{10, "宽字符A", ""}, " 宽字符A  "},
		{"Pad wide charactors with space and zero width", args{0, "宽字符", ""}, "宽字符"},
		{"Pad wide charactors with space and smaller width", args{5, "宽字符", ""}, "宽字符"},
		{"Pad wide charactors with space and same width", args{6, "宽字符", ""}, "宽字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.args.stringIn == "" && tt.args.padString == "" {
				got = pad(tt.args.width)
			} else if tt.args.padString == "" {
				got = pad(tt.args.width, tt.args.stringIn)
			} else {
				got = pad(tt.args.width, tt.args.stringIn, tt.args.padString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPadLeft(t *testing.T) {
	type args struct {
		width     int
		stringIn  string
		padString string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"PadLeft empty string with space", args{10, "", ""}, "          "},
		{"PadLeft empty string with minus", args{10, "", "-"}, "----------"},
		{"PadLeft with space", args{10, "test", ""}, "      test"},
		{"PadLeft with space 2", args{10, "hello", ""}, "     hello"},
		{"PadLeft with minus", args{10, "test", "-"}, "------test"},
		{"PadLeft with space and zero width", args{0, "test", ""}, "test"},
		{"PadLeft with space and smaller width", args{3, "test", ""}, "test"},
		{"PadLeft with space and same width", args{4, "test", ""}, "test"},
		{"PadLeft wide charactors with space", args{10, "宽字符", ""}, "    宽字符"},
		{"PadLeft wide charactors with space 2", args{10, "宽字符A", ""}, "   宽字符A"},
		{"PadLeft wide charactors with space and zero width", args{0, "宽字符", ""}, "宽字符"},
		{"PadLeft wide charactors with space and smaller width", args{5, "宽字符", ""}, "宽字符"},
		{"PadLeft wide charactors with space and same width", args{6, "宽字符", ""}, "宽字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.args.stringIn == "" && tt.args.padString == "" {
				got = padLeft(tt.args.width)
			} else if tt.args.padString == "" {
				got = padLeft(tt.args.width, tt.args.stringIn)
			} else {
				got = padLeft(tt.args.width, tt.args.stringIn, tt.args.padString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPadRight(t *testing.T) {
	type args struct {
		width     int
		stringIn  string
		padString string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"PadRight empty string with space", args{10, "", ""}, "          "},
		{"PadRight empty string with minus", args{10, "", "-"}, "----------"},
		{"PadRight with space", args{10, "test", ""}, "test      "},
		{"PadRight with space 2", args{10, "hello", ""}, "hello     "},
		{"PadRight with minus", args{10, "test", "-"}, "test------"},
		{"PadRight with space and zero width", args{0, "test", ""}, "test"},
		{"PadRight with space and smaller width", args{3, "test", ""}, "test"},
		{"PadRight with space and same width", args{4, "test", ""}, "test"},
		{"PadRight wide charactors with space", args{10, "宽字符", ""}, "宽字符    "},
		{"PadRight wide charactors with space 2", args{10, "宽字符A", ""}, "宽字符A   "},
		{"PadRight wide charactors with space and zero width", args{0, "宽字符", ""}, "宽字符"},
		{"PadRight wide charactors with space and smaller width", args{5, "宽字符", ""}, "宽字符"},
		{"PadRight wide charactors with space and same width", args{6, "宽字符", ""}, "宽字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.args.stringIn == "" && tt.args.padString == "" {
				got = padRight(tt.args.width)
			} else if tt.args.padString == "" {
				got = padRight(tt.args.width, tt.args.stringIn)
			} else {
				got = padRight(tt.args.width, tt.args.stringIn, tt.args.padString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPadRightX(t *testing.T) {
	type args struct {
		width     int
		stringIn  string
		padString string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"PadRightX empty string with space", args{10, "", " "}, "          "},
		{"PadRightX empty string with minus", args{10, "", "-"}, "----------"},
		{"PadRightX empty string with minus and plus", args{10, "", "-+"}, "-+-+-+-+-+"},
		{"PadRightX empty string with minus plus and dot", args{10, "", "-+."}, "-+.-+.-+.-"},
		{"PadRightX with space", args{10, "test", " "}, "test      "},
		{"PadRightX with space 2", args{10, "hello", " "}, "hello     "},
		{"PadRightX with minus", args{10, "test", "-"}, "test------"},
		{"PadRightX with space and zero width", args{0, "test", " "}, "test"},
		{"PadRightX with space and smaller width", args{3, "test", " "}, "test"},
		{"PadRightX with space and same width", args{4, "test", " "}, "test"},
		{"PadRightX wide charactors with space", args{10, "宽字符", " "}, "宽字符    "},
		{"PadRightX wide charactors with space 2", args{10, "宽字符A", " "}, "宽字符A   "},
		{"PadRightX wide charactors with space and zero width", args{0, "宽字符", " "}, "宽字符"},
		{"PadRightX wide charactors with space and smaller width", args{5, "宽字符", " "}, "宽字符"},
		{"PadRightX wide charactors with space and same width", args{6, "宽字符", " "}, "宽字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRightX(tt.args.stringIn, tt.args.padString, tt.args.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringOr(t *testing.T) {
	type args struct {
		stringA string
		stringB string
		padding []int
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"StringOr", args{"a", "b", []int{10}}, "a         "},
		{"StringOr empty string a and b", args{"", "", []int{10}}, "          "},
		{"StringOr empty string a", args{"", "b", []int{10}}, "b         "},
		{"StringOr empty string b", args{"a", "", []int{10}}, "a         "},
		{"StringOr without padding", args{"a", "b", nil}, "a"},
		{"StringOr with zero width", args{"test", "b", []int{0}}, "test"},
		{"StringOr with smaller width", args{"test", "b", []int{3}}, "test"},
		{"StringOr with same width", args{"test", "b", []int{4}}, "test"},
		{"StringOr wide charactors", args{"宽字符", "", []int{10}}, "宽字符    "},
		{"StringOr wide charactors 2", args{"宽字符A", "", []int{10}}, "宽字符A   "},
		{"StringOr wide charactors with zero width", args{"宽字符", "", []int{0}}, "宽字符"},
		{"StringOr wide charactors with smaller width", args{"宽字符", "", []int{5}}, "宽字符"},
		{"StringOr wide charactors with same width", args{"宽字符", "", []int{6}}, "宽字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringOr(tt.args.stringA, tt.args.stringB, tt.args.padding...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDamageQuality(t *testing.T) {
	fn := funcMap["damageQuality"].(func(float64) string)

	tests := []struct {
		name string
		mult float64
		want string
	}{
		{"zero", 0.0, "negligible striking power"},
		{"feeble low", 0.3, "feeble striking power"},
		{"feeble mid", 0.5, "feeble striking power"},
		{"light low", 0.6, "light striking power"},
		{"light high", 0.99, "light striking power"},
		{"moderate low", 1.0, "moderate striking power"},
		{"moderate high", 1.49, "moderate striking power"},
		{"strong low", 1.5, "strong striking power"},
		{"strong high", 2.49, "strong striking power"},
		{"devastating low", 2.5, "devastating striking power"},
		{"devastating high", 3.99, "devastating striking power"},
		{"legendary", 4.0, "legendary striking power"},
		{"legendary high", 10.0, "legendary striking power"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.mult))
		})
	}
}

func TestSpellDamageQuality(t *testing.T) {
	fn := funcMap["spellDamageQuality"].(func(float64) string)

	tests := []struct {
		name string
		mult float64
		want string
	}{
		{"zero", 0.0, "negligible arcane resonance"},
		{"faint", 0.5, "faint arcane resonance"},
		{"mild", 0.8, "mild arcane resonance"},
		{"moderate", 1.2, "moderate arcane resonance"},
		{"strong", 1.6, "strong arcane resonance"},
		{"intense", 2.5, "intense arcane resonance"},
		{"legendary", 4.0, "legendary arcane resonance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.mult))
		})
	}
}

func TestStatModDescription(t *testing.T) {
	fn := funcMap["statModDescription"].(func(string, int) string)

	tests := []struct {
		name  string
		stat  string
		value int
		want  string
	}{
		{"large positive", "strength", 25, "greatly bolsters your strength"},
		{"medium positive", "dexterity", 15, "bolsters your dexterity"},
		{"small positive", "perception", 3, "slightly bolsters your perception"},
		{"large negative", "vitality", -25, "greatly saps your vitality"},
		{"medium negative", "willpower", -15, "saps your willpower"},
		{"small negative", "charisma", -3, "slightly saps your charisma"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.stat, tt.value))
		})
	}
}
