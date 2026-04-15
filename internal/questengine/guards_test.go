package questengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalGuard_New(t *testing.T) {
	g := NewEvalGuard(10)
	assert.Equal(t, 0, g.Depth())
	assert.False(t, g.DepthExceeded())
}

func TestEvalGuard_DepthLimit(t *testing.T) {
	g := NewEvalGuard(3)
	assert.True(t, g.PushDepth())
	assert.True(t, g.PushDepth())
	assert.True(t, g.PushDepth())
	assert.False(t, g.PushDepth())
	assert.True(t, g.DepthExceeded())
}

func TestEvalGuard_PopDepth(t *testing.T) {
	g := NewEvalGuard(10)
	g.PushDepth()
	g.PushDepth()
	assert.Equal(t, 2, g.Depth())
	g.PopDepth()
	assert.Equal(t, 1, g.Depth())
	g.PopDepth()
	assert.Equal(t, 0, g.Depth())
	g.PopDepth() // should not go negative
	assert.Equal(t, 0, g.Depth())
}

func TestEvalGuard_VisitTracking(t *testing.T) {
	g := NewEvalGuard(10)
	assert.True(t, g.MarkVisited("q3-t0"))
	assert.False(t, g.MarkVisited("q3-t0"))
	assert.True(t, g.MarkVisited("q3-t1"))
}

func TestEvalGuard_DuplicateToken(t *testing.T) {
	g := NewEvalGuard(10)
	assert.True(t, g.MarkGranted("3-start"))
	assert.False(t, g.MarkGranted("3-start"))
	assert.True(t, g.MarkGranted("3-totem"))
}

func TestEvalGuard_ChainTrace(t *testing.T) {
	g := NewEvalGuard(10)
	g.PushDepth()
	g.AddToTrace("item_give → q3-t1 → grant 3-totem")
	g.PushDepth()
	g.AddToTrace("quest_granted → q3-t3 → grant 3-end")
	trace := g.GetTrace()
	assert.Len(t, trace, 2)
	assert.Equal(t, "item_give → q3-t1 → grant 3-totem", trace[0])
}
