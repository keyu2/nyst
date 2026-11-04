package mta

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

type realtimeTransport func(*http.Request) (*http.Response, error)

func (transport realtimeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestRealtimeKeepsWorkingFeedsWhenOneFeedFails(t *testing.T) {
	var calls atomic.Int32
	entity := &gtfs.FeedEntity{Id: proto.String("entity-w"), TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String("running-w"), RouteId: proto.String("W")}}}
	body, err := proto.Marshal(&gtfs.FeedMessage{Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")}, Entity: []*gtfs.FeedEntity{entity}})
	if err != nil {
		t.Fatal(err)
	}
	service := NewRealtimeService()
	service.client = &http.Client{Transport: realtimeTransport(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		if strings.HasSuffix(request.URL.Path, "gtfs-ace") {
			return nil, errors.New("feed unavailable")
		}
		payload := []byte{}
		if strings.HasSuffix(request.URL.Path, "gtfs-nqrw") {
			payload = body
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	})}
	entities, err := service.Entities(context.Background())
	if err != nil || len(entities) != 1 || entities[0].GetTripUpdate().GetTrip().GetRouteId() != "W" {
		t.Fatalf("working service disappeared: %v %v", entities, err)
	}
	if calls.Load() != int32(len(realtimeFeedNames)) {
		t.Fatalf("not all original feeds requested: %d", calls.Load())
	}
	if _, err := service.Entities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != int32(len(realtimeFeedNames)) {
		t.Fatal("fresh snapshot fetched again before original 15 second TTL")
	}
}

func TestRealtimeRefreshFailureRetainsPreviousSnapshotLikeOriginal(t *testing.T) {
	previous := &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String("last-valid"), RouteId: proto.String("E")}}}
	service := NewRealtimeService()
	service.entities = []*gtfs.FeedEntity{previous}
	service.loadedAt = time.Now().Add(-time.Minute)
	service.client = &http.Client{Transport: realtimeTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("upstream offline") })}
	entities, err := service.Entities(context.Background())
	if err != nil || len(entities) != 1 || entities[0] != previous {
		t.Fatalf("last valid snapshot was discarded: %v %v", entities, err)
	}
	if time.Since(service.loadedAt) < 15*time.Second {
		t.Fatal("failed fetch was stamped as a successful refresh")
	}
}

func TestInitialRealtimeFailureRemainsUnavailable(t *testing.T) {
	service := NewRealtimeService()
	service.client = &http.Client{Transport: realtimeTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("upstream offline") })}
	entities, err := service.Entities(context.Background())
	if err == nil || len(entities) != 0 {
		t.Fatalf("failed feed became confirmed empty service: %v %v", entities, err)
	}
}
