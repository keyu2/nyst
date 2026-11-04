package server

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRealtimeStreamSendsRefreshAndReleasesDisconnectedClient(t *testing.T) {
	s := &Server{listeners: map[chan time.Time]struct{}{}}
	host := httptest.NewServer(http.HandlerFunc(s.events))
	defer host.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(host.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal("response is not an event stream")
	}
	reader := bufio.NewReader(response.Body)
	readEvent := func() string {
		var result strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			result.WriteString(line)
			if line == "\n" {
				return result.String()
			}
		}
	}
	if !strings.Contains(readEvent(), "event: ready") {
		t.Fatal("stream did not open")
	}
	updatedAt := time.Date(2026, 11, 3, 14, 30, 0, 0, time.UTC)
	s.broadcastLiveUpdate(updatedAt)
	message := readEvent()
	if !strings.Contains(message, "event: refresh") || !strings.Contains(message, updatedAt.Format(time.RFC3339)) {
		t.Fatalf("missing refresh notification: %s", message)
	}
	response.Body.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.liveMu.Lock()
		count := len(s.listeners)
		s.liveMu.Unlock()
		if count == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("disconnected client was retained")
}

func TestSlowRealtimeClientDoesNotAccumulateNotifications(t *testing.T) {
	updates := make(chan time.Time, 1)
	s := &Server{listeners: map[chan time.Time]struct{}{updates: {}}}
	for index := range 1000 {
		s.broadcastLiveUpdate(time.Unix(int64(index), 0))
	}
	if len(updates) != 1 {
		t.Fatal("slow client accumulated updates")
	}
}
