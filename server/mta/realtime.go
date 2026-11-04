package mta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

const realtimeBaseURL = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds"

var realtimeFeedNames = []string{
	"gtfs", "gtfs-ace", "gtfs-bdfm", "gtfs-g", "gtfs-jz",
	"gtfs-l", "gtfs-nqrw", "gtfs-si",
}

type RealtimeService struct {
	client   *http.Client
	mu       sync.Mutex
	entities []*gtfs.FeedEntity
	loadedAt time.Time
	loading  chan struct{}
	loadErr  error
}

func NewRealtimeService() *RealtimeService {
	return &RealtimeService{client: &http.Client{Timeout: 6 * time.Second}}
}

func (s *RealtimeService) Entities(ctx context.Context) ([]*gtfs.FeedEntity, error) {
	s.mu.Lock()
	if len(s.entities) > 0 && time.Since(s.loadedAt) < 15*time.Second {
		entities := s.entities
		s.mu.Unlock()
		return entities, nil
	}
	if s.loading != nil {
		waiting := s.loading
		s.mu.Unlock()
		select {
		case <-waiting:
			s.mu.Lock()
			entities, err := s.entities, s.loadErr
			s.mu.Unlock()
			return entities, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.loading = make(chan struct{})
	waiting := s.loading
	previous := s.entities
	s.mu.Unlock()

	entities, err := s.download(ctx)
	s.mu.Lock()
	if err == nil {
		s.entities, s.loadedAt = entities, time.Now()
	} else if len(previous) > 0 {
		entities, err = previous, nil
	}
	s.loadErr = err
	close(waiting)
	s.loading = nil
	s.mu.Unlock()
	return entities, err
}

func (s *RealtimeService) download(ctx context.Context) ([]*gtfs.FeedEntity, error) {
	type result struct {
		entities []*gtfs.FeedEntity
		err      error
	}
	results := make([]result, len(realtimeFeedNames))
	var group sync.WaitGroup
	for index, name := range realtimeFeedNames {
		group.Add(1)
		go func() {
			defer group.Done()
			entities, err := s.downloadFeed(ctx, name)
			results[index] = result{entities: entities, err: err}
		}()
	}
	group.Wait()

	var all []*gtfs.FeedEntity
	var failures []error
	for _, result := range results {
		if result.err != nil {
			failures = append(failures, result.err)
			continue
		}
		all = append(all, result.entities...)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("all MTA realtime feeds failed: %w", errors.Join(failures...))
	}

	return preferredRealtimeEntities(all), nil
}

func preferredRealtimeEntities(all []*gtfs.FeedEntity) []*gtfs.FeedEntity {
	preferred := map[string]*gtfs.FeedEntity{}
	vehicles := map[string]*gtfs.FeedEntity{}
	tripOrder := []string{}
	vehicleOrder := []string{}
	for _, entity := range all {
		update := entity.GetTripUpdate()
		tripID := update.GetTrip().GetTripId()
		if tripID != "" {
			current := preferred[tripID]
			if current == nil {
				tripOrder = append(tripOrder, tripID)
			}
			if current == nil || len(update.GetStopTimeUpdate()) > len(current.GetTripUpdate().GetStopTimeUpdate()) {
				preferred[tripID] = entity
			}
		}
		if vehicleID := entity.GetVehicle().GetTrip().GetTripId(); vehicleID != "" {
			if vehicles[vehicleID] == nil {
				vehicleOrder = append(vehicleOrder, vehicleID)
			}
			vehicles[vehicleID] = entity
		}
	}
	entities := make([]*gtfs.FeedEntity, 0, len(preferred)+len(vehicles))
	for _, id := range tripOrder {
		entities = append(entities, preferred[id])
	}
	for _, id := range vehicleOrder {
		entities = append(entities, vehicles[id])
	}
	return entities
}

func (s *RealtimeService) downloadFeed(ctx context.Context, name string) ([]*gtfs.FeedEntity, error) {
	endpoint := realtimeBaseURL + "/nyct%2F" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", name, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	feed := &gtfs.FeedMessage{}
	if err := proto.Unmarshal(body, feed); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return feed.GetEntity(), nil
}

func realtimeRoutesByStop(entities []*gtfs.FeedEntity, now time.Time) map[string]map[string]struct{} {
	routes := map[string]map[string]struct{}{}
	threshold := now.Unix()
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		route := normalizeRouteID(update.GetTrip().GetRouteId())
		if route == "" {
			continue
		}
		for _, stopTime := range update.GetStopTimeUpdate() {
			timestamp := stopTime.GetArrival().GetTime()
			if timestamp == 0 {
				timestamp = stopTime.GetDeparture().GetTime()
			}
			if timestamp == 0 || timestamp < threshold {
				continue
			}
			stop := stationStopID(stopTime.GetStopId())
			if stop == "" {
				continue
			}
			if routes[stop] == nil {
				routes[stop] = map[string]struct{}{}
			}
			routes[stop][route] = struct{}{}
		}
	}
	return routes
}
