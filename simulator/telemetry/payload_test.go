package telemetry

import (
	"encoding/json"
	"testing"
)

func TestPayloadIsValidJSON(t *testing.T) {
	if !json.Valid(Bytes()) {
		t.Fatal("telemetry Payload is not valid JSON")
	}
}

func TestTopic(t *testing.T) {
	want := "heyev/v1/devices/866224084563153/telemetry"
	if got := Topic(); got != want {
		t.Fatalf("Topic() = %q, want %q", got, want)
	}
}
