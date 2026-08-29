package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultClientID = "heyev-backend-poc"
	CertDir         = "../certs/backend"
	CACertFile      = CertDir + "/AmazonRootCA1.pem"
	ClientCertFile  = CertDir + "/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-certificate.pem.crt"
	ClientKeyFile   = CertDir + "/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-private.pem.key"
)

type Config struct {
	MQTTVersion           string
	QoS                   byte
	Retain                bool
	Dup                   bool
	AllowDuplicatePublish bool
	AutoReconnect         bool
	Debug                 bool
	Endpoint              string
	ClientID              string
	DeviceID              string
	CommandDeviceID       string
	WildcardSubscribe     bool
	AckSubscribeTopic     string
	CommandPublishTopic   string
}

func Load() (*Config, error) {
	cfg := &Config{
		MQTTVersion: "5",
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

	if c.Dup {
		// Logged at startup; not an error.
	}

	return nil
}

func (c *Config) SetDeviceTopics(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device ID cannot be empty")
	}

	c.DeviceID = deviceID
	if deviceID == "+" {
		c.WildcardSubscribe = true
		c.AckSubscribeTopic = "heyev/v1/devices/+/ack"
		c.CommandPublishTopic = ""
		return nil
	}

	c.WildcardSubscribe = false
	c.AckSubscribeTopic = fmt.Sprintf("heyev/v1/devices/%s/ack", deviceID)
	c.CommandPublishTopic = fmt.Sprintf("heyev/v1/devices/%s/commands", deviceID)
	c.CommandDeviceID = deviceID
	return nil
}

func (c *Config) SetCommandDeviceID(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || deviceID == "+" {
		return errors.New("command device ID must be a concrete device identifier, not +")
	}
	c.CommandDeviceID = deviceID
	c.CommandPublishTopic = fmt.Sprintf("heyev/v1/devices/%s/commands", deviceID)
	return nil
}

func (c *Config) CommandTopicFor(deviceID string) string {
	return fmt.Sprintf("heyev/v1/devices/%s/commands", deviceID)
}
