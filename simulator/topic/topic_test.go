package topic

import "testing"

func TestParseDeviceID(t *testing.T) {
	got, err := ParseDeviceID("heyev/v1/devices/6264/commands")
	if err != nil {
		t.Fatal(err)
	}
	if got != "6264" {
		t.Fatalf("got %q", got)
	}
}

func TestAckTopic(t *testing.T) {
	if got := AckTopic("1234"); got != "heyev/v1/devices/1234/ack" {
		t.Fatalf("unexpected: %s", got)
	}
}
