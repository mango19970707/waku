package contracts

import "testing"

func TestNormalizePlotlinePayload_Success(t *testing.T) {
	raw := `{"plotlines":[{"title":"主线","type":"main","description":"主角成长","status":"active"},{"id":"2","title":"支线","type":"sub","description":"宗门纷争"}]}`

	normalized, payload, err := NormalizePlotlinePayload(raw)
	if err != nil {
		t.Fatalf("NormalizePlotlinePayload() error = %v", err)
	}
	if normalized == "" {
		t.Fatalf("NormalizePlotlinePayload() returned empty normalized payload")
	}
	if len(payload.Plotlines) != 2 {
		t.Fatalf("expected 2 plotlines, got %d", len(payload.Plotlines))
	}
	if payload.Plotlines[1]["status"] != "active" {
		t.Fatalf("expected default status active, got %#v", payload.Plotlines[1]["status"])
	}
}

func TestNormalizePlotlinePayload_InvalidType(t *testing.T) {
	raw := `{"plotlines":[{"title":"主线","type":"foo","description":"主角成长"}]}`

	if _, _, err := NormalizePlotlinePayload(raw); err == nil {
		t.Fatalf("expected error for invalid type")
	}
}
