package config

import "testing"

func TestValidateOptionBRequiresQoS1(t *testing.T) {
	cfg := &Config{MQTTVersion: "5", DeliveryMode: DeliveryModeB, QoS: 0, PersistentSession: true, SessionExpirySec: 900}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected Option B with QoS 0 to be rejected")
	}
}

func TestValidateOptionBRequiresPersistentSession(t *testing.T) {
	cfg := &Config{MQTTVersion: "5", DeliveryMode: DeliveryModeB, QoS: 1, PersistentSession: false, SessionExpirySec: 900}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected Option B without persistent session to be rejected")
	}
}

func TestValidateOptionBRequiresSessionExpiry(t *testing.T) {
	cfg := &Config{MQTTVersion: "5", DeliveryMode: DeliveryModeB, QoS: 1, PersistentSession: true, SessionExpirySec: 0}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected Option B with session expiry 0 to be rejected")
	}
}
