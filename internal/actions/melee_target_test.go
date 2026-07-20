package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMeleeTargetOpts_MessagesMatchPreExtraction pins the exact player-facing
// strings each of the 11 melee verbs produced before AcquireMeleeTarget was
// extracted. These were copied verbatim from the command files; if a default
// here drifts, a player sees different text than they used to.
func TestMeleeTargetOpts_MessagesMatchPreExtraction(t *testing.T) {
	tests := []struct {
		name     string
		opts     MeleeTargetOpts
		crafting string
		prompt   string
		self     string
		charmed  string
	}{
		{
			name:     "bash",
			opts:     MeleeTargetOpts{Verb: "bash"},
			crafting: `<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Bash whom?",
			self:     "You can't bash yourself.",
			charmed:  "You can't bash a companion.",
		},
		{
			name:     "drain",
			opts:     MeleeTargetOpts{Verb: "drain"},
			crafting: `<ansi fg="red">You can't drain while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Drain whom?",
			self:     "You can't drain yourself.",
			charmed:  "You can't drain a companion.",
		},
		{
			name:     "gore",
			opts:     MeleeTargetOpts{Verb: "gore"},
			crafting: `<ansi fg="red">You can't gore while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Gore whom?",
			self:     "You can't gore yourself.",
			charmed:  "You can't gore a companion.",
		},
		{
			name:     "grapple",
			opts:     MeleeTargetOpts{Verb: "grapple"},
			crafting: `<ansi fg="red">You can't grapple while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Grapple whom?",
			self:     "You can't grapple yourself.",
			charmed:  "You can't grapple a companion.",
		},
		{
			name:     "kick",
			opts:     MeleeTargetOpts{Verb: "kick"},
			crafting: `<ansi fg="red">You can't kick while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Kick whom?",
			self:     "You can't kick yourself.",
			charmed:  "You can't kick a companion.",
		},
		{
			name:     "maul",
			opts:     MeleeTargetOpts{Verb: "maul"},
			crafting: `<ansi fg="red">You can't maul while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Maul whom?",
			self:     "You can't maul yourself.",
			charmed:  "You can't maul a companion.",
		},
		{
			// pounce differs on BOTH the prompt and the self message.
			name: "pounce",
			opts: MeleeTargetOpts{
				Verb:          "pounce",
				PromptMsg:     "Pounce on whom?",
				SelfTargetMsg: "You can't pounce on yourself.",
				CharmedMsg:    "You can't pounce on a companion.",
			},
			crafting: `<ansi fg="red">You can't pounce while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Pounce on whom?",
			self:     "You can't pounce on yourself.",
			charmed:  "You can't pounce on a companion.",
		},
		{
			name:     "rake",
			opts:     MeleeTargetOpts{Verb: "rake"},
			crafting: `<ansi fg="red">You can't rake while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Rake whom?",
			self:     "You can't rake yourself.",
			charmed:  "You can't rake a companion.",
		},
		{
			name:     "taunt",
			opts:     MeleeTargetOpts{Verb: "taunt"},
			crafting: `<ansi fg="red">You can't taunt while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Taunt whom?",
			self:     "You can't taunt yourself.",
			charmed:  "You can't taunt a companion.",
		},
		{
			name:     "throttle",
			opts:     MeleeTargetOpts{Verb: "throttle"},
			crafting: `<ansi fg="red">You can't throttle while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Throttle whom?",
			self:     "You can't throttle yourself.",
			charmed:  "You can't throttle a companion.",
		},
		{
			// trip's crafting message says "trip someone", not just "trip".
			name:     "trip",
			opts:     MeleeTargetOpts{Verb: "trip", CraftingVerb: "trip someone"},
			crafting: `<ansi fg="red">You can't trip someone while focused on your work. Finish or be interrupted first.</ansi>`,
			prompt:   "Trip whom?",
			self:     "You can't trip yourself.",
			charmed:  "You can't trip a companion.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.crafting, tt.opts.craftingMsg(), "crafting-guard message changed")
			assert.Equal(t, tt.prompt, tt.opts.promptMsg(), "empty-target prompt changed")
			assert.Equal(t, tt.self, tt.opts.selfTargetMsg(), "self-target message changed")
			assert.Equal(t, tt.charmed, tt.opts.charmedMsg(), "charmed-target message changed")
		})
	}
}
