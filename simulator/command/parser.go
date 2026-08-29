package command

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

type IncomingCommand struct {
	Topic            string
	DeviceID         string
	RequestID        string
	Command          string
	Value            string
	Payload          string
	QoS              byte
	Retain           bool
	MQTTDuplicate    bool
	DuplicateRequest bool
	ReceivedAt       time.Time
	Raw              *paho.Publish
}

type Payload struct {
	RequestID string `json:"request_id"`
	DeviceID  string `json:"device_id"`
	Command   string `json:"command"`
	Value     string `json:"value"`
	Timestamp string `json:"timestamp"`
}

func Parse(msg *paho.Publish) (*IncomingCommand, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	deviceID, err := parseDeviceFromTopic(msg.Topic)
	if err != nil {
		return nil, err
	}

	var payload Payload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("parse command JSON: %w", err)
	}

	if payload.RequestID == "" {
		return nil, fmt.Errorf("command missing request_id")
	}

	return &IncomingCommand{
		Topic:         msg.Topic,
		DeviceID:      deviceID,
		RequestID:     payload.RequestID,
		Command:       payload.Command,
		Value:         payload.Value,
		Payload:       string(msg.Payload),
		QoS:           msg.QoS,
		Retain:        msg.Retain,
		MQTTDuplicate: msg.Duplicate(),
		ReceivedAt:    time.Now().UTC(),
		Raw:           msg,
	}, nil
}

func parseDeviceFromTopic(topic string) (string, error) {
	const prefix = "heyev/v1/devices/"
	if len(topic) <= len(prefix) {
		return "", fmt.Errorf("invalid topic %q", topic)
	}
	rest := topic[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			id := rest[:i]
			if id == "" || id == "+" {
				return "", fmt.Errorf("invalid device_id in topic %q", topic)
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("invalid topic %q", topic)
}
