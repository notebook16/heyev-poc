package command

import (
	"testing"

	"github.com/eclipse/paho.golang/paho"
)

func TestParseCommand(t *testing.T) {
	msg := &paho.Publish{
		Topic:   "heyev/v1/devices/6264/commands",
		Payload: []byte(`{"request_id":"cmd-1","device_id":"6264","command":"TEST","value":"hello"}`),
		QoS:     1,
	}

	cmd, err := Parse(msg)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.DeviceID != "6264" {
		t.Fatalf("device id = %q", cmd.DeviceID)
	}
	if cmd.RequestID != "cmd-1" {
		t.Fatalf("request id = %q", cmd.RequestID)
	}
}
