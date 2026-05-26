package conversations

import (
	"strings"
	"testing"
)

func TestValidatePool_OK(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{
				{Speaker: "A", Text: "hello"},
				{Speaker: "B", Text: "hi back"},
			}},
		},
	}
	if err := validatePoolStandalone(p); err != nil {
		t.Errorf("valid pool should validate, got: %v", err)
	}
}

func TestValidatePool_UnknownTypeRejected(t *testing.T) {
	p := &Pool{
		Id: "not-a-real-type",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "x"}, {Speaker: "B", Text: "y"}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "relationship type") {
		t.Errorf("expected relationship-type error, got: %v", err)
	}
}

func TestValidatePool_EmptyExchangeRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: nil},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "no lines") {
		t.Errorf("expected no-lines error, got: %v", err)
	}
}

func TestValidatePool_BadSpeakerRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "C", Text: "x"}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "speaker") {
		t.Errorf("expected speaker error, got: %v", err)
	}
}

func TestValidatePool_EmptyTextRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: ""}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "empty text") {
		t.Errorf("expected empty-text error, got: %v", err)
	}
}

func TestValidatePairOverride_OK(t *testing.T) {
	po := &PairOverride{
		Id:   "test_pair",
		MobA: 100, MobB: 200,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "hi"}, {Speaker: "B", Text: "hi"}}},
		},
	}
	if err := validatePairOverrideStandalone(po); err != nil {
		t.Errorf("valid pair override should validate, got: %v", err)
	}
}

func TestValidatePairOverride_SelfPairRejected(t *testing.T) {
	po := &PairOverride{
		Id:   "self",
		MobA: 100, MobB: 100,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "x"}}},
		},
	}
	err := validatePairOverrideStandalone(po)
	if err == nil || !strings.Contains(err.Error(), "self") {
		t.Errorf("expected self-pair error, got: %v", err)
	}
}

// World-aware checks (mob existence, relationship edge presence) are
// covered by the boot smoke in T13; the integration setup is too heavy
// for unit tests.
func TestValidatePairOverrideAgainstWorld_Stub(t *testing.T) {
	t.Skip("requires loaded mobs registry + relationships graph — covered by T13 boot smoke")
}
