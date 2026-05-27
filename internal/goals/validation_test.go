package goals

import (
	"errors"
	"testing"
)

func TestValidateParams_NoSchema_AllParamsPass(t *testing.T) {
	g := &Goal{Type: "freeform", Params: map[string]any{"anything": 1, "goes": "here"}}
	if err := ValidateParams(g, nil); err != nil {
		t.Errorf("got err=%v, want nil (no schema = no validation)", err)
	}
}

func TestValidateParams_RequiredKeyPresent_Pass(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil", err)
	}
}

func TestValidateParams_RequiredKeyMissing_Fails(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{}}
	err := ValidateParams(g, schema)
	if err == nil {
		t.Fatalf("got nil, want ErrBadParams")
	}
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if bpe.Key != "target" {
		t.Errorf("bpe.Key=%q, want %q", bpe.Key, "target")
	}
}

func TestValidateParams_WrongType_Fails(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{"target": "five hundred"}}
	err := ValidateParams(g, schema)
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if bpe.ExpectedType != "int" {
		t.Errorf("bpe.ExpectedType=%q, want int", bpe.ExpectedType)
	}
	if bpe.GotType != "string" {
		t.Errorf("bpe.GotType=%q, want string", bpe.GotType)
	}
}

func TestValidateParams_OptionalKeyMissing_Pass(t *testing.T) {
	schema := []ParamSchema{
		{Key: "target", Required: true, GoType: "int"},
		{Key: "threshold", Required: false, GoType: "int"},
	}
	g := &Goal{Type: "x", Params: map[string]any{"target": 1}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil (optional key absent is fine)", err)
	}
}

func TestValidateParams_IntFromInt64_Pass(t *testing.T) {
	// YAML round-trips integers as int64; the validator should accept either.
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "x", Params: map[string]any{"target": int64(500)}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil (int64 should satisfy int schema)", err)
	}
}

func TestValidateParams_StringSlice_AcceptsBothShapes(t *testing.T) {
	// YAML can unmarshal a string list as []interface{} OR []string depending on path.
	schema := []ParamSchema{{Key: "tags", Required: true, GoType: "[]string"}}

	g1 := &Goal{Type: "x", Params: map[string]any{"tags": []any{"a", "b"}}}
	if err := ValidateParams(g1, schema); err != nil {
		t.Errorf("[]any{string}: got err=%v, want nil", err)
	}

	g2 := &Goal{Type: "x", Params: map[string]any{"tags": []string{"a", "b"}}}
	if err := ValidateParams(g2, schema); err != nil {
		t.Errorf("[]string: got err=%v, want nil", err)
	}

	g3 := &Goal{Type: "x", Params: map[string]any{"tags": []any{"a", 5}}}
	if err := ValidateParams(g3, schema); err == nil {
		t.Errorf("[]any with non-string element: want err, got nil")
	}
}

func TestValidateParams_FloatFromInt_Pass(t *testing.T) {
	schema := []ParamSchema{{Key: "ratio", Required: true, GoType: "float64"}}
	g := &Goal{Type: "x", Params: map[string]any{"ratio": 5}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("int → float64: got err=%v, want nil", err)
	}
}

func TestValidateParams_BoolStrict(t *testing.T) {
	schema := []ParamSchema{{Key: "flag", Required: true, GoType: "bool"}}
	if err := ValidateParams(&Goal{Type: "x", Params: map[string]any{"flag": true}}, schema); err != nil {
		t.Errorf("bool true: got err=%v, want nil", err)
	}
	if err := ValidateParams(&Goal{Type: "x", Params: map[string]any{"flag": 1}}, schema); err == nil {
		t.Errorf("int as bool: want err, got nil")
	}
}
