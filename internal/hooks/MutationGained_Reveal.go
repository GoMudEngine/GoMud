package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// deepenFlourishText is the rank-up announcement (no splash ceremony —
// spec 2026-07-29: deepenings get the lighter reveal).
func deepenFlourishText(name string, rank int, maxLevel int) string {
	levelTag := fmt.Sprintf(`Level %d`, rank)
	if rank >= maxLevel {
		levelTag = `fully matured`
	}
	return fmt.Sprintf(
		`<ansi fg="magenta">The Chrysalis deepens its hold. Your <ansi fg="yellow">%s</ansi> grows stronger (%s).</ansi>`,
		name, levelTag)
}

// revealCaption is the plain-text form: the splash caption for
// screen-reader users, and part of the degraded no-splash path.
func revealCaption(name, description string) string {
	return fmt.Sprintf(`Something stirs beneath your skin. A mutation emerges: %s. %s`, name, description)
}

// flattenDescription collapses the YAML description's authored hard-wraps
// into one paragraph so the splash template's splitstring is the only
// wrapping applied.
func flattenDescription(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// MutationGained_Reveal renders the terminal side of a mutation reveal:
// a per-user chrysalis splash ceremony for new acquisitions, a short
// flourish for deepenings. The web card is pushed separately by the gmcp
// module listening to the same event.
func MutationGained_Reveal(e events.Event) events.ListenerReturn {
	evt, ok := e.(mutations.Gained)
	if !ok {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	spec := mutations.GetMutation(evt.MutationId)
	if spec == nil {
		return events.Continue
	}

	if !evt.IsNew {
		user.SendText(messaging.CategoryMutation,
			deepenFlourishText(spec.Name, evt.Rank, int(configs.GetBalanceConfig().MutationMaxLevel)))
		return events.Continue
	}

	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		// Splash_Deliver drops everything when splashes are disabled, so
		// degrade here to the classic two-line announcement.
		user.SendText(messaging.CategoryMutation, fmt.Sprintf(
			`<ansi fg="magenta">Something stirs beneath your skin. A mutation emerges: <ansi fg="yellow">%s</ansi>.</ansi>`,
			spec.Name))
		user.SendText(messaging.CategoryMutation, fmt.Sprintf(`<ansi fg="magenta">%s</ansi>`, spec.Description))
		return events.Continue
	}

	events.AddToQueue(splash.Splash{
		SceneId: `mutation_reveal`,
		Caption: revealCaption(spec.Name, flattenDescription(spec.Description)),
		Target:  splash.TargetUser,
		UserId:  evt.UserId,
		Data: map[string]any{
			`name`:        spec.Name,
			`description`: flattenDescription(spec.Description),
		},
	})
	return events.Continue
}
