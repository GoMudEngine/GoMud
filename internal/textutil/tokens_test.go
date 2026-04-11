package textutil

import "testing"

func TestSubstituteTokens_AllTokens(t *testing.T) {
	ctx := TokenContext{
		SourceName:      `<ansi fg="yellow">Kael</ansi>`,
		SourcePlainName: `Kael`,
		TargetName:      `<ansi fg="red">Goblin</ansi>`,
		TargetPlainName: `Goblin`,
	}
	input := `{source} hurls a bolt at {target}. {source_plain}'s eyes glow. {target_plain} staggers.`
	expected := `<ansi fg="yellow">Kael</ansi> hurls a bolt at <ansi fg="red">Goblin</ansi>. Kael's eyes glow. Goblin staggers.`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_EmptyTarget(t *testing.T) {
	ctx := TokenContext{
		SourceName:      `Kael`,
		SourcePlainName: `Kael`,
	}
	input := `{source} channels energy at {target}.`
	expected := `Kael channels energy at .`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_NoTokens(t *testing.T) {
	ctx := TokenContext{SourceName: `Kael`}
	input := `Energy crackles in the air.`
	result := SubstituteTokens(input, ctx)
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestSubstituteTokens_EmptyString(t *testing.T) {
	ctx := TokenContext{}
	result := SubstituteTokens("", ctx)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestValidateTokens_KnownTokens(t *testing.T) {
	warnings := ValidateTokens(`{source} attacks {target}`)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestValidateTokens_UnknownToken(t *testing.T) {
	warnings := ValidateTokens(`{source} attacks {targat}`)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if warnings[0] != `unknown token: {targat}` {
		t.Errorf("got %q", warnings[0])
	}
}

func TestValidateTokens_EmptyString(t *testing.T) {
	warnings := ValidateTokens("")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}
