package config

import "testing"

func TestSetDeviceTopicsConcrete(t *testing.T) {
	cfg := &Config{}
	if err := cfg.SetDeviceTopics("6264"); err != nil {
		t.Fatal(err)
	}
	if cfg.AckSubscribeTopic != "heyev/v1/devices/6264/ack" {
		t.Fatalf("ack topic = %q", cfg.AckSubscribeTopic)
	}
	if cfg.CommandPublishTopic != "heyev/v1/devices/6264/commands" {
		t.Fatalf("command topic = %q", cfg.CommandPublishTopic)
	}
}

func TestSetDeviceTopicsWildcard(t *testing.T) {
	cfg := &Config{}
	if err := cfg.SetDeviceTopics("+"); err != nil {
		t.Fatal(err)
	}
	if !cfg.WildcardSubscribe {
		t.Fatal("expected wildcard subscribe")
	}
	if cfg.CommandPublishTopic != "" {
		t.Fatalf("command topic should be empty for wildcard, got %q", cfg.CommandPublishTopic)
	}
}

func TestValidateRejectsMQTT311(t *testing.T) {
	cfg := &Config{MQTTVersion: "3.1.1", QoS: 0}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected mqtt 3.1.1 to be rejected")
	}
}

func TestValidateRejectsQoS2(t *testing.T) {
	cfg := &Config{MQTTVersion: "5", QoS: 2}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected qos 2 to be rejected")
	}
}
