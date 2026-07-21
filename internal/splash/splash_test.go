package splash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplashEventType(t *testing.T) {
	assert.Equal(t, "Splash", Splash{}.Type())
}

func TestTargetConstants(t *testing.T) {
	// Distinct values; Global is the zero value so a bare Splash{} is global.
	assert.Equal(t, SplashTarget(0), TargetGlobal)
	assert.NotEqual(t, TargetGlobal, TargetZone)
	assert.NotEqual(t, TargetZone, TargetUser)
}
