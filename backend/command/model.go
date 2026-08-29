package command

import "time"

type Command struct {
	RequestID string    `json:"request_id"`
	DeviceID  string    `json:"device_id"`
	Command   string    `json:"command"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
