package ack

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eclipse/paho.golang/paho"

	"heyev-backend-poc/logger"
	"heyev-backend-poc/state"
	"heyev-backend-poc/topic"
)

type Handler struct {
	log         *logger.Logger
	idempotency *IdempotencyStore
	tracker     *state.Tracker
}

func NewHandler(log *logger.Logger, store *IdempotencyStore, tracker *state.Tracker) *Handler {
	return &Handler{
		log:         log,
		idempotency: store,
		tracker:     tracker,
	}
}

func (h *Handler) Handle(msg *paho.Publish) {
	receivedAt := time.Now().UTC()

	deviceID, err := topic.ParseDeviceID(msg.Topic)
	if err != nil {
		h.log.Error("Failed to parse device ID from topic %s: %v", msg.Topic, err)
		return
	}

	var ack Ack
	if err := json.Unmarshal(msg.Payload, &ack); err != nil {
		h.log.Error("Failed to parse ACK payload: %v", err)
		return
	}

	h.log.Ack("========================================")
	h.log.Ack("ACK RECEIVED")
	h.log.Ack("========================================")
	h.log.Ack("Topic: %s", msg.Topic)
	h.log.Ack("Device ID: %s", deviceID)
	h.log.Ack("QoS: %d", msg.QoS)
	h.log.Ack("Payload: %s", string(msg.Payload))
	h.log.Ack("Timestamp: %s", receivedAt.Format(time.RFC3339Nano))
	h.log.Ack("Request ID: %s", ack.RequestID)
	h.log.Ack("Status: %s", ack.Status)
	h.log.Ack("========================================")

	if ack.RequestID == "" {
		h.log.Error("ACK missing request_id")
		return
	}

	if !h.idempotency.IsNew(ack.RequestID) {
		h.log.Idempotency("DUPLICATE ACK - IGNORING request_id=%s", ack.RequestID)
		h.tracker.Set(ack.RequestID, state.StatusDuplicate)
		return
	}

	h.log.Ack("PROCESS ACK request_id=%s", ack.RequestID)
	h.idempotency.MarkProcessed(ack.RequestID)

	h.log.Ack("Stage 3: Device-level ACK received (simulator receipt confirmation)")
	h.log.Ack("Meaning: %s", ack.Message)

	switch ack.Status {
	case "SUCCESS":
		h.tracker.Set(ack.RequestID, state.StatusSuccess)
		h.log.State("request_id=%s status=%s", ack.RequestID, state.StatusSuccess)
		h.log.Ack("Stage 4: SUCCESS reported by device/simulator")
	default:
		if _, err := h.tracker.Transition(ack.RequestID, state.StatusAcknowledged); err != nil {
			h.tracker.Set(ack.RequestID, state.StatusAcknowledged)
		}
		h.log.State("request_id=%s status=%s", ack.RequestID, state.StatusAcknowledged)
		h.log.Ack("Stage 4: SUCCESS not available — ACKNOWLEDGED means command received by simulator only")
	}
}

func ParsePayload(data []byte) (*Ack, error) {
	var a Ack
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("unmarshal ack: %w", err)
	}
	return &a, nil
}
