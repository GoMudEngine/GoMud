package knowledge

import "testing"

func TestSubjectEquality(t *testing.T) {
	a := Subject{Type: SubjectPlayer, Id: 17}
	b := Subject{Type: SubjectPlayer, Id: 17}
	c := Subject{Type: SubjectMob, Id: 17}
	if a != b {
		t.Errorf("expected equal subjects to compare equal")
	}
	if a == c {
		t.Errorf("player(17) should not equal mob(17)")
	}
}

func TestSubjectHelpers(t *testing.T) {
	if PlayerSubject(17) != (Subject{Type: SubjectPlayer, Id: 17}) {
		t.Errorf("PlayerSubject helper mismatch")
	}
	if MobSubject(99) != (Subject{Type: SubjectMob, Id: 99}) {
		t.Errorf("MobSubject helper mismatch")
	}
}
