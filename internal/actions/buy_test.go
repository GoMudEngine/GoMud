package actions

import "testing"

func TestBuy_EmptyRequest(t *testing.T) {
	result := Buy(nil, BuyOptions{Request: ""})
	if result.Success {
		t.Errorf("expected Success=false on empty request")
	}
	if result.Reason != BuyReasonNoRequest {
		t.Errorf("expected Reason=%q, got %q", BuyReasonNoRequest, result.Reason)
	}
}
