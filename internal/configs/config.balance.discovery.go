package configs

// validateDiscovery sets defaults for the shared spell/recipe
// discovery offset knobs.
func (b *Balance) validateDiscovery() {
	if b.DiscoveryPerceptionScale <= 0 {
		b.DiscoveryPerceptionScale = 200.0
	}
	if b.DiscoverySkillScale <= 0 {
		b.DiscoverySkillScale = 100.0
	}
	if b.DiscoveryMaxDecayOffset <= 0 {
		b.DiscoveryMaxDecayOffset = 0.8
	}
}
