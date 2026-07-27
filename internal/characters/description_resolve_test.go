package characters

import "testing"

func TestResolveDescriptionToken(t *testing.T) {
	prose := "A weathered ferryman with kind eyes and salt-stiff clothes."
	c := &Character{Description: prose}
	c.CacheDescription() // interns + replaces Description with h:<hash>

	got, ok := ResolveDescriptionToken(c.Description)
	if !ok || got != prose {
		t.Fatalf("interned token should resolve to the prose, got %q ok=%v", got, ok)
	}

	plain := "plain prose, not a token"
	got, ok = ResolveDescriptionToken(plain)
	if !ok || got != plain {
		t.Fatalf("non-token should pass through, got %q ok=%v", got, ok)
	}

	if _, ok := ResolveDescriptionToken("h:0000000000000000000000000000000000000000000000000000000000000000"); ok {
		t.Fatal("unknown token must not resolve")
	}
}
