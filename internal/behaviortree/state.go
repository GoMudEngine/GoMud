package behaviortree

// BehaviorState is a per-mob-instance key/value store for behavior
// tree state. Persists for the mob's lifetime, reset on respawn.
type BehaviorState struct {
	data map[string]any
}

func NewBehaviorState() *BehaviorState {
	return &BehaviorState{data: make(map[string]any)}
}

func (s *BehaviorState) Get(key string) any {
	if s.data == nil {
		return nil
	}
	return s.data[key]
}

func (s *BehaviorState) GetString(key string) string {
	v, _ := s.Get(key).(string)
	return v
}

func (s *BehaviorState) GetInt(key string) int {
	switch v := s.Get(key).(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func (s *BehaviorState) Set(key string, value any) {
	if s.data == nil {
		s.data = make(map[string]any)
	}
	s.data[key] = value
}

func (s *BehaviorState) Delete(key string) {
	delete(s.data, key)
}
