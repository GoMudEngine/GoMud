package configs

import "testing"

func TestValidate_MailSendCooldownRoundsDefault(t *testing.T) {
	b := &Balance{}
	b.validateMisc()
	if int(b.MailSendCooldownRounds) != 10 {
		t.Errorf("MailSendCooldownRounds default = %d, want 10", int(b.MailSendCooldownRounds))
	}
}
