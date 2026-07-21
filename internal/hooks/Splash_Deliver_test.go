package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

func TestSplashTemplatePath(t *testing.T) {
	assert.Equal(t, "generic/splash/sunrise", splashTemplatePath("sunrise"))
	assert.Equal(t, "generic/splash/weather_storm", splashTemplatePath("weather_storm"))
}

type notASplashEvent struct{}

func (notASplashEvent) Type() string { return "dummy" }

func TestSplashDeliverWrongEventType(t *testing.T) {
	// A non-Splash event is ignored (Continue), no panic.
	assert.Equal(t, events.Continue, Splash_Deliver(notASplashEvent{}))
}
