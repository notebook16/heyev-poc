package command

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"heyev-backend-poc/logger"
	"heyev-backend-poc/state"
)

type Service struct {
	log         *logger.Logger
	idempotency *IdempotencyStore
	tracker     *state.Tracker
	allowDup    bool
}

func NewService(log *logger.Logger, store *IdempotencyStore, tracker *state.Tracker, allowDup bool) *Service {
	return &Service{
		log:         log,
		idempotency: store,
		tracker:     tracker,
		allowDup:    allowDup,
	}
}

func (s *Service) Build(deviceID, commandName, value, requestID string) (*Command, error) {
	if requestID == "" {
		requestID = "cmd-" + uuid.NewString()
	}

	cmd := &Command{
		RequestID: requestID,
		DeviceID:  deviceID,
		Command:   commandName,
		Value:     value,
		Timestamp: time.Now().UTC(),
	}

	return cmd, nil
}

func (s *Service) CanPublish(requestID string) bool {
	if s.allowDup {
		s.log.Idempotency("Duplicate publish allowed by --allow-duplicate-publish for request_id=%s", requestID)
		return true
	}

	if s.idempotency.IsNew(requestID) {
		s.log.Idempotency("NEW COMMAND request_id=%s", requestID)
		return true
	}

	s.log.Idempotency("DUPLICATE COMMAND BLOCKED request_id=%s", requestID)
	s.tracker.Set(requestID, state.StatusDuplicate)
	return false
}

func (s *Service) MarkPublished(requestID string) {
	s.idempotency.MarkPublished(requestID)
	s.tracker.Set(requestID, state.StatusPublished)
	s.log.State("request_id=%s status=%s", requestID, state.StatusPublished)
}

func (s *Service) Marshal(cmd *Command) ([]byte, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}
	return data, nil
}
