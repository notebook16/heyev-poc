package ack

import "time"

type Ack struct {
	RequestID string    `json:"request_id"`
	DeviceID  string    `json:"device_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
