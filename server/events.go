package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) refreshLiveData(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.liveMu.Lock()
			active := len(s.listeners) > 0
			s.liveMu.Unlock()
			if !active {
				continue
			}
			if _, err := s.realtime.Entities(ctx); err != nil {
				continue
			}
			s.broadcastLiveUpdate(time.Now().UTC())
		}
	}
}

func (s *Server) broadcastLiveUpdate(updatedAt time.Time) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	for listener := range s.listeners {
		select {
		case listener <- updatedAt:
		default:
		}
	}
}

func (s *Server) events(response http.ResponseWriter, request *http.Request) {
	if _, ok := response.(http.Flusher); !ok {
		http.Error(response, "Streaming is unavailable", http.StatusNotImplemented)
		return
	}
	updates := make(chan time.Time, 1)
	s.liveMu.Lock()
	s.listeners[updates] = struct{}{}
	s.liveMu.Unlock()
	defer func() {
		s.liveMu.Lock()
		delete(s.listeners, updates)
		s.liveMu.Unlock()
	}()
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache, no-transform")
	response.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(response)
	write := func(message string) error {
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := fmt.Fprint(response, message); err != nil {
			return err
		}
		if err := controller.Flush(); err != nil {
			return err
		}
		_ = controller.SetWriteDeadline(time.Time{})
		return nil
	}
	if err := write("retry: 3000\nevent: ready\ndata: {}\n\n"); err != nil {
		return
	}
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case updatedAt := <-updates:
			if err := write(fmt.Sprintf("event: refresh\ndata: {\"updatedAt\":%q}\n\n", updatedAt.Format(time.RFC3339Nano))); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := write(": heartbeat\n\n"); err != nil {
				return
			}
		}
	}
}
