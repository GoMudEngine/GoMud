package questengine

// ResetEngineForTest replaces the global engine with a fresh empty Engine
// and returns a cleanup function that restores the original. Test-only.
func ResetEngineForTest() func() {
	orig := globalEngine
	globalEngine = NewEngine()
	return func() {
		globalEngine = orig
	}
}
