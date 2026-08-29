package topic

import (
	"fmt"
	"strings"
)

const prefix = "heyev/v1/devices/"

func AckTopic(deviceID string) string {
	return fmt.Sprintf("%s%s/ack", prefix, deviceID)
}

func CommandTopic(deviceID string) string {
	return fmt.Sprintf("%s%s/commands", prefix, deviceID)
}

func ParseDeviceID(topic string) (string, error) {
	if !strings.HasPrefix(topic, prefix) {
		return "", fmt.Errorf("topic %q does not match heyev/v1/devices/{device_id}/...", topic)
	}

	rest := strings.TrimPrefix(topic, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("topic %q is missing suffix after device_id", topic)
	}

	deviceID := parts[0]
	if deviceID == "" || deviceID == "+" {
		return "", fmt.Errorf("topic %q has invalid device_id %q", topic, deviceID)
	}

	return deviceID, nil
}
