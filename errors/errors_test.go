package errors

import "testing"

func TestNewBadGatewayError_IsA502NotAServerError(t *testing.T) {
	e := NewBadGatewayError("", nil)
	if e.Status != 502 {
		t.Fatalf("status = %d, want 502", e.Status)
	}
	if e.Message == "" {
		t.Fatal("expected a default message")
	}
}
