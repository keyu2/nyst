package mta

import (
	"math"
	"reflect"
	"testing"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func TestMapOnlyDrawsServicesWithUpcomingRealtimeTrains(t *testing.T) {
	trip := staticTrip{ID: "live-e", RouteID: "E", ShapeID: "east", ServiceID: "inactive"}
	data := &staticData{
		Trips:             []staticTrip{trip, {ID: "m", RouteID: "M", ShapeID: "east"}},
		TripsByRealtimeID: map[string]staticTrip{"live-e": trip},
		ShapesByID:        map[string][]shapePoint{"east": {{Longitude: -74, Latitude: 40.7}, {Longitude: -73.9, Latitude: 40.8}}},
	}
	entity := func(id, route string, timestamp int64) *gtfs.FeedEntity {
		return &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String(id), RouteId: proto.String(route)}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(timestamp)}}}}}
	}
	entities := []*gtfs.FeedEntity{entity("live-e", "E", 200), entity("m", "M", 100)}
	lines := currentRouteLines(data, entities, time.Unix(150, 0))
	if len(lines) != 1 || lines[0].Properties.RouteID != "E" {
		t.Fatalf("live matched route omitted or inactive route drawn: %v", lines)
	}
	if len(currentRouteLines(data, entities, time.Unix(250, 0))) != 0 {
		t.Fatal("expired service remained on the map")
	}
}

func TestMapOppositeDirectionsShareOneShapeWhileBranchesRemain(t *testing.T) {
	shapes := map[string][]shapePoint{
		"north":  {{Longitude: -74, Latitude: 40.7}, {Longitude: -73.9, Latitude: 40.8}},
		"south":  {{Longitude: -73.9, Latitude: 40.8}, {Longitude: -74, Latitude: 40.7}},
		"branch": {{Longitude: -74, Latitude: 40.7}, {Longitude: -73.8, Latitude: 40.9}},
	}
	trips := []staticTrip{{RouteID: "F", ShapeID: "north"}, {RouteID: "F", ShapeID: "south"}, {RouteID: "F", ShapeID: "branch"}, {RouteID: "E", ShapeID: "north"}}
	lines := routeLineFeatures(trips, shapes, nil)
	if len(lines) != 3 {
		t.Fatalf("opposite directions duplicated or a branch/service was removed: %v", lines)
	}
}

func TestRealtimeStationMapKeepsSameColorServicesAndSmoothsStopFallback(t *testing.T) {
	data := &staticData{Stops: map[string]staticStop{
		"A01": {Longitude: -74, Latitude: 40.70},
		"A02": {Longitude: -73.99, Latitude: 40.71},
		"A03": {Longitude: -73.98, Latitude: 40.72},
	}}
	entity := func(route string) *gtfs.FeedEntity {
		return &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{RouteId: proto.String(route)}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("A01N")}, {StopId: proto.String("A02N")}, {StopId: proto.String("A03N")}}}}
	}
	lines := realtimeStationLines([]*gtfs.FeedEntity{entity("C"), entity("E")}, data, []string{"C", "E"}, map[string]struct{}{"A02": {}})
	if len(lines) != 2 || lines[0].Properties.RouteID != "C" || lines[1].Properties.RouteID != "E" || lines[0].Properties.Offset == lines[1].Properties.Offset {
		t.Fatalf("same-color services were merged: %v", lines)
	}
	points := lines[0].Geometry.Coordinates
	if len(points) != 17 || math.Abs(points[8][0]+73.99) > 1e-10 || math.Abs(points[8][1]-40.71) > 1e-10 {
		t.Fatalf("smoothed path lost its station coordinate: %v", points)
	}
}

func TestStationMapUsesLiveComplexServicesBeforeSchedule(t *testing.T) {
	station := Station{Services: []StationService{
		{Routes: []string{"N", "W"}, ScheduledRoutes: []string{"N", "W"}},
		{Routes: []string{"7"}, RealtimeRoutes: []string{"7"}},
	}}
	if got := mapStationRoutes(station); !reflect.DeepEqual(got, []string{"7"}) {
		t.Fatalf("scheduled platforms overrode live station services: %v", got)
	}
}

func TestMapMergesOppositeTracksAndPreservesOtherColorsAndBranches(t *testing.T) {
	feature := func(color string, points [][]float64) MapFeature {
		line := MapFeature{Type: "Feature"}
		line.Properties.Color = color
		line.Geometry.Type = "LineString"
		line.Geometry.Coordinates = points
		return line
	}
	lines := mergeMapCorridors([]MapFeature{
		feature("#00933C", [][]float64{{-74, 40.7}, {-73.99, 40.71}, {-73.98, 40.72}}),
		feature("#00933C", [][]float64{{-73.9801, 40.7201}, {-73.9901, 40.7101}, {-74.0001, 40.7001}}),
		feature("#FF6319", [][]float64{{-73.98, 40.72}, {-73.99, 40.71}, {-74, 40.7}}),
		feature("#00933C", [][]float64{{-74, 40.7}, {-73.98, 40.7}, {-73.98, 40.72}}),
	})
	if len(lines) != 3 {
		t.Fatalf("got %d corridors, want 3", len(lines))
	}
	if lines[1].Geometry.Coordinates[0][0] != -74 {
		t.Fatal("shared corridors use opposite offset directions")
	}
}

func TestExpressMapMarkersKeepTheOriginalRouteFamilyBullet(t *testing.T) {
	data := &staticData{Stops: map[string]staticStop{"601": {Longitude: -73.99, Latitude: 40.71}, "602": {Longitude: -73.98, Latitude: 40.72}}}
	entity := &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String("express"), RouteId: proto.String("6X")}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("601N"), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(200)}}, {StopId: proto.String("602N"), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(300)}}}}}
	vehicles := mapVehicles([]*gtfs.FeedEntity{entity}, data, map[string]struct{}{"6": {}}, nil, time.Unix(150, 0))
	if len(vehicles) != 1 || vehicles[0].RouteID != "6" {
		t.Fatalf("marker bullet differs from original: %v", vehicles)
	}
}

func TestMapsHideTripsBeforeDepartureIncludingDelayedOrigins(t *testing.T) {
	now := time.Date(2026, 11, 4, 19, 0, 0, 0, newYorkLocation)
	data := &staticData{Stops: map[string]staticStop{"Q01": {Longitude: -74, Latitude: 40.7}, "Q02": {Longitude: -73.99, Latitude: 40.71}}, TripsByRealtimeID: map[string]staticTrip{"pending": {ID: "scheduled"}}, StopTimesByTrip: map[string][]stopTime{"scheduled": {{StopID: "Q01S", DepartureTime: "18:55:00"}}}}
	trip := &gtfs.TripDescriptor{TripId: proto.String("pending"), RouteId: proto.String("Q"), StartDate: proto.String("20261104"), StartTime: proto.String("19:06:00")}
	update := &gtfs.TripUpdate{Trip: trip, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("Q01S"), Departure: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(now.Unix() + 360)}}}}
	entities := []*gtfs.FeedEntity{{TripUpdate: update}, {Vehicle: &gtfs.VehiclePosition{Trip: trip, Position: &gtfs.Position{Longitude: proto.Float32(-74), Latitude: proto.Float32(40.7)}}}}
	displayed := map[string]struct{}{"Q": {}}
	if got := mapVehicles(entities, data, displayed, nil, now); len(got) != 0 {
		t.Fatal("pending train appeared", got)
	}
	trip.StartTime = proto.String("18:55:00")
	if got := mapVehicles(entities, data, displayed, nil, now); len(got) != 0 {
		t.Fatal("delayed origin appeared", got)
	}
	update.StopTimeUpdate[0].StopId = proto.String("Q02S")
	if got := mapVehicles(entities, data, displayed, nil, now); len(got) != 1 {
		t.Fatal("running train disappeared", got)
	}
	trip.ScheduleRelationship = gtfs.TripDescriptor_CANCELED.Enum()
	if got := mapVehicles(entities, data, displayed, nil, now); len(got) != 0 {
		t.Fatal("canceled train appeared", got)
	}
}
