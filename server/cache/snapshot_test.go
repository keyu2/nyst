package cache

import (
	"errors"
	"testing"
	"time"
)

func TestStoreKeepsOnlyCurrentAndPreviousSnapshots(t *testing.T) {
	store := NewStore()
	first := time.Unix(10, 0).UTC()
	second := time.Unix(20, 0).UTC()
	third := time.Unix(30, 0).UTC()

	store.Publish(1, first)
	store.Publish(2, second)
	store.Publish(3, third)

	if got := store.Current().AlertCount; got != 3 {
		t.Fatalf("current alert count = %d, want 3", got)
	}
	if got := store.Previous().AlertCount; got != 2 {
		t.Fatalf("previous alert count = %d, want 2", got)
	}
}

func TestFailurePreservesLastSuccessfulData(t *testing.T) {
	store := NewStore()
	success := time.Unix(10, 0).UTC()
	failure := time.Unix(20, 0).UTC()
	store.Publish(12, success)
	store.Fail(errors.New("feed unavailable"), failure)

	snapshot := store.Current()
	if snapshot.AlertCount != 12 {
		t.Fatalf("alert count = %d, want last successful value 12", snapshot.AlertCount)
	}
	if snapshot.LastSuccessfulAt != success {
		t.Fatalf("last successful time = %s, want %s", snapshot.LastSuccessfulAt, success)
	}
	if snapshot.ConsecutiveFailures != 1 {
		t.Fatalf("consecutive failures = %d, want 1", snapshot.ConsecutiveFailures)
	}
}
