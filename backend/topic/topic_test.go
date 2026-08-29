package topic

import "testing"

func TestParseDeviceID(t *testing.T) {
	tests := []struct {
		topic   string
		want    string
		wantErr bool
	}{
		{"heyev/v1/devices/6264/ack", "6264", false},
		{"heyev/v1/devices/TEST-DEVICE-01/commands", "TEST-DEVICE-01", false},
		{"heyev/v1/devices/+/ack", "", true},
		{"invalid/topic", "", true},
	}

	for _, tt := range tests {
		got, err := ParseDeviceID(tt.topic)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for topic %q", tt.topic)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.topic, err)
		}
		if got != tt.want {
			t.Fatalf("ParseDeviceID(%q) = %q, want %q", tt.topic, got, tt.want)
		}
	}
}

func TestAckTopic(t *testing.T) {
	if got := AckTopic("6264"); got != "heyev/v1/devices/6264/ack" {
		t.Fatalf("unexpected topic: %s", got)
	}
}
