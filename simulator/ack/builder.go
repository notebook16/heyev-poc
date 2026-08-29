package ack

import (
	"encoding/json"
	"time"
)

type Payload struct {
	RequestID string `json:"request_id"`
	DeviceID  string `json:"device_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func Build(deviceID, requestID string) ([]byte, error) {
	payload := Payload{
		RequestID: requestID,
		DeviceID:  deviceID,
		Status:    "ACKNOWLEDGED",
		Message:   "Command received by simulator",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return json.Marshal(payload)
}
