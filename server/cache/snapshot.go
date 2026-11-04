package cache

import (
	"sync"
	"time"
)

type Snapshot struct {
	AlertCount          int       `json:"alertCount"`
	UpdatedAt           time.Time `json:"updatedAt"`
	LastAttemptAt       time.Time `json:"lastAttemptAt"`
	LastSuccessfulAt    time.Time `json:"lastSuccessfulAt"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	LastError           string    `json:"-"`
}

type Store struct {
	mu       sync.RWMutex
	current  Snapshot
	previous Snapshot
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Current() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *Store) Publish(alertCount int, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previous = s.current
	s.current = Snapshot{
		AlertCount:       alertCount,
		UpdatedAt:        now,
		LastAttemptAt:    now,
		LastSuccessfulAt: now,
	}
}

func (s *Store) Fail(err error, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previous = s.current
	s.current.LastAttemptAt = now
	s.current.ConsecutiveFailures++
	s.current.LastError = err.Error()
}

func (s *Store) Previous() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.previous
}
