package ack

import "testing"

func TestIdempotencyDuplicateACK(t *testing.T) {
	store := NewIdempotencyStore()
	if !store.IsNew("cmd-1") {
		t.Fatal("first ack should be new")
	}
	store.MarkProcessed("cmd-1")
	if store.IsNew("cmd-1") {
		t.Fatal("duplicate ack should not be new")
	}
}
