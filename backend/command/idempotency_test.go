package command

import (
	"testing"

	"heyev-backend-poc/logger"
	"heyev-backend-poc/state"
)

func TestIdempotencyBlocksDuplicate(t *testing.T) {
	log := logger.New()
	store := NewIdempotencyStore()
	tracker := state.NewTracker()
	svc := NewService(log, store, tracker, false)

	if !svc.CanPublish("cmd-1") {
		t.Fatal("first publish should be allowed")
	}
	svc.MarkPublished("cmd-1")

	if svc.CanPublish("cmd-1") {
		t.Fatal("duplicate publish should be blocked")
	}
}

func TestAllowDuplicatePublish(t *testing.T) {
	log := logger.New()
	store := NewIdempotencyStore()
	tracker := state.NewTracker()
	svc := NewService(log, store, tracker, true)

	svc.MarkPublished("cmd-1")
	if !svc.CanPublish("cmd-1") {
		t.Fatal("duplicate publish should be allowed with flag")
	}
}
