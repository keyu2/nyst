package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/keyu2/nyst/server/cache"
	"github.com/keyu2/nyst/server/mta"
)

type Server struct {
	assets    fs.FS
	cache     *cache.Store
	static    *mta.StaticService
	transit   *mta.TransitService
	realtime  *mta.RealtimeService
	alerts    *mta.Client
	http      *http.Server
	startedAt time.Time
	liveMu    sync.Mutex
	listeners map[chan time.Time]struct{}
}

func New(assets fs.FS, store *cache.Store, static *mta.StaticService, transit *mta.TransitService, realtime *mta.RealtimeService, alerts *mta.Client) *Server {
	return &Server{assets: assets, cache: store, static: static, transit: transit, realtime: realtime, alerts: alerts, startedAt: time.Now().UTC(), listeners: map[chan time.Time]struct{}{}}
}

func (s *Server) Run(ctx context.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /api/bootstrap", s.bootstrap)
	mux.HandleFunc("GET /api/map", s.subwayMap)
	mux.HandleFunc("GET /api/maps/stations/{id}", s.stationMap)
	mux.HandleFunc("GET /api/stops/{id}", s.stop)
	mux.HandleFunc("GET /api/routes/{id}", s.route)
	mux.HandleFunc("GET /api/routes/{id}/alerts", s.routeAlerts)
	mux.HandleFunc("GET /api/trips/{routeId}/{tripId}", s.trip)
	mux.Handle("/", s.webApp())

	s.http = &http.Server{
		Addr:              ":" + port,
		Handler:           requestLog(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      75 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	result := make(chan error, 1)
	go s.refreshLiveData(ctx)
	go func() {
		slog.Info("NYST server listening", "port", port)
		result <- s.http.ListenAndServe()
	}()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	}
}

func (s *Server) stationMap(response http.ResponseWriter, request *http.Request) {
	data, err := s.static.StationMap(request.Context(), request.PathValue("id"), time.Now())
	if err != nil {
		writeJSON(response, http.StatusBadGateway, map[string]string{"error": "MTA station map is temporarily unavailable"})
		return
	}
	writeJSON(response, http.StatusOK, data)
}

func (s *Server) subwayMap(response http.ResponseWriter, request *http.Request) {
	data, err := s.static.Map(request.Context(), time.Now())
	if err != nil {
		slog.Error("load MTA map", "error", err)
		writeJSON(response, http.StatusBadGateway, map[string]string{"error": "MTA map data is temporarily unavailable"})
		return
	}
	writeJSON(response, http.StatusOK, data)
}

func (s *Server) trip(response http.ResponseWriter, request *http.Request) {
	trip, err := s.transit.Trip(request.Context(), request.PathValue("routeId"), request.PathValue("tripId"), time.Now())
	if err != nil {
		slog.Error("load MTA trip", "trip", request.PathValue("tripId"), "error", err)
		writeJSON(response, http.StatusNotFound, map[string]string{"error": "MTA realtime trip not found"})
		return
	}
	writeJSON(response, http.StatusOK, trip)
}

func (s *Server) route(response http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	var routeAlerts []mta.RouteAlert
	if snapshot, err := s.alerts.RouteAlertSnapshot(request.Context(), []string{id}, time.Now()); err == nil {
		routeAlerts = snapshot.RouteAlerts[id]
	}
	route, err := s.transit.Route(request.Context(), id, time.Now(), routeAlerts...)
	if err != nil {
		slog.Error("load MTA route", "id", request.PathValue("id"), "error", err)
		writeJSON(response, http.StatusBadGateway, map[string]string{"error": "MTA route data is temporarily unavailable"})
		return
	}
	writeJSON(response, http.StatusOK, route)
}

func (s *Server) routeAlerts(response http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	snapshot, err := s.alerts.RouteAlertSnapshot(request.Context(), []string{id}, time.Now())
	if err != nil {
		slog.Error("load MTA route alerts", "id", id, "error", err)
		writeJSON(response, http.StatusBadGateway, map[string]string{"error": "MTA service alerts are temporarily unavailable"})
		return
	}
	writeJSON(response, http.StatusOK, snapshot.RouteAlerts[id])
}

func (s *Server) stop(response http.ResponseWriter, request *http.Request) {
	stop, err := s.transit.Stop(request.Context(), request.PathValue("id"), time.Now())
	if err != nil {
		slog.Error("load MTA stop", "id", request.PathValue("id"), "error", err)
		writeJSON(response, http.StatusBadGateway, map[string]string{"error": "MTA realtime stop data is temporarily unavailable"})
		return
	}
	writeJSON(response, http.StatusOK, stop)
}

func (s *Server) bootstrap(response http.ResponseWriter, request *http.Request) {
	data, err := s.static.Bootstrap(request.Context(), time.Now())
	if err != nil {
		slog.Error("load MTA static data", "error", err)
		writeJSON(response, http.StatusBadGateway, map[string]string{
			"error": "MTA static data is temporarily unavailable",
		})
		return
	}
	if snapshot, alertErr := s.alerts.RouteAlertSnapshot(request.Context(), data.Routes, time.Now()); alertErr == nil {
		data.RouteAlerts = snapshot.RouteAlerts
		data.RouteStatuses = snapshot.RouteStatuses
	}
	writeJSON(response, http.StatusOK, data)
}

func (s *Server) health(response http.ResponseWriter, _ *http.Request) {
	snapshot := s.cache.Current()
	health := "ok"
	status := http.StatusOK
	if snapshot.LastSuccessfulAt.IsZero() || time.Since(snapshot.LastSuccessfulAt) > 2*time.Minute {
		health = "degraded"
		status = http.StatusServiceUnavailable
	}
	writeJSON(response, status, map[string]any{
		"status":           health,
		"startedAt":        s.startedAt,
		"lastSuccessfulAt": snapshot.LastSuccessfulAt,
	})
}

func (s *Server) status(response http.ResponseWriter, _ *http.Request) {
	snapshot := s.cache.Current()
	writeJSON(response, http.StatusOK, map[string]any{
		"alertCount":          snapshot.AlertCount,
		"updatedAt":           snapshot.UpdatedAt,
		"lastSuccessfulAt":    snapshot.LastSuccessfulAt,
		"consecutiveFailures": snapshot.ConsecutiveFailures,
		"stale":               snapshot.LastSuccessfulAt.IsZero() || time.Since(snapshot.LastSuccessfulAt) > 90*time.Second,
	})
}

func (s *Server) webApp() http.Handler {
	files := http.FileServer(http.FS(s.assets))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		path := strings.TrimPrefix(request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(s.assets, path); err != nil {
			request.URL.Path = "/"
		}
		if strings.HasPrefix(path, "assets/") || strings.HasPrefix(path, "_app/immutable/") {
			response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(response, request)
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(response, request)
		if strings.HasPrefix(request.URL.Path, "/api/") {
			slog.Info("request", "method", request.Method, "path", request.URL.Path, "duration", time.Since(started))
		}
	})
}
