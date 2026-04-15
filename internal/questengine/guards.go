package questengine

// EvalGuard provides multiple layers of safety during trigger evaluation.
type EvalGuard struct {
	maxDepth     int
	currentDepth int
	exceeded     bool
	visited      map[string]bool // trigger IDs visited in this chain
	granted      map[string]bool // tokens granted in this chain
	trace        []string        // chain trace for debugging
}

func NewEvalGuard(maxDepth int) *EvalGuard {
	return &EvalGuard{
		maxDepth: maxDepth,
		visited:  make(map[string]bool),
		granted:  make(map[string]bool),
	}
}

func (g *EvalGuard) PushDepth() bool {
	g.currentDepth++
	if g.currentDepth > g.maxDepth {
		g.exceeded = true
		return false
	}
	return true
}

func (g *EvalGuard) PopDepth() {
	if g.currentDepth > 0 {
		g.currentDepth--
	}
}

func (g *EvalGuard) Depth() int {
	return g.currentDepth
}

func (g *EvalGuard) DepthExceeded() bool {
	return g.exceeded
}

func (g *EvalGuard) MarkVisited(trigId string) bool {
	if g.visited[trigId] {
		return false
	}
	g.visited[trigId] = true
	return true
}

func (g *EvalGuard) MarkGranted(token string) bool {
	if g.granted[token] {
		return false
	}
	g.granted[token] = true
	return true
}

func (g *EvalGuard) AddToTrace(desc string) {
	g.trace = append(g.trace, desc)
}

func (g *EvalGuard) GetTrace() []string {
	return g.trace
}
