package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultClientID = "iot-simulator-poc"
	CertDir         = "../certs/simulator"
	CACertFile      = CertDir + "/AmazonRootCA1.pem"
	ClientCertFile  = CertDir + "/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-certificate.pem.crt"
	ClientKeyFile   = CertDir + "/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-private.pem.key"
)

type Mode string

const (
	ModeAutonomous Mode = "autonomous"
	ModeControlled Mode = "controlled"
)

type Config struct {
	MQTTVersion   string
	QoS           byte
	Retain        bool
	Dup           bool
	AutoReconnect bool
	Debug         bool
	Mode          Mode
	Endpoint      string
	ClientID      string
	DeviceID      string
	CommandTopic  string
}

func Load() (*Config, error) {
	cfg := &Config{
		MQTTVersion: "5",
		Mode:        ModeAutonomous,
		ClientID:    DefaultClientID,
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := cfg.loadEnvironment(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) loadEnvironment() error {
	c.Endpoint = os.Getenv("IOT_ENDPOINT")
	if c.Endpoint == "" {
		return errors.New("IOT_ENDPOINT environment variable is required")
	}

	for _, path := range []string{CACertFile, ClientCertFile, ClientKeyFile} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("certificate file missing or unreadable: %s: %w", path, err)
		}
	}
	return nil
}

func (c *Config) validate() error {
	v := strings.ToLower(c.MQTTVersion)
	if v != "5" {
		return fmt.Errorf("mqtt-version %q is not supported: github.com/eclipse/paho.golang is MQTT 5 only", c.MQTTVersion)
	}
	if c.QoS > 1 {
		return fmt.Errorf("qos %d is not supported: AWS IoT Core supports QoS 0 and 1 only", c.QoS)
	}
	return nil
}

func (c *Config) SetCommandSubscribeTopic(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device ID cannot be empty")
	}
	c.DeviceID = deviceID
	c.CommandTopic = fmt.Sprintf("heyev/v1/devices/%s/commands", deviceID)
	return nil
}

func (c *Config) DefaultCommandTopic() string {
	return "heyev/v1/devices/+/commands"
}
