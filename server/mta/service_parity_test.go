package mta

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func TestStationShowsDepartureOnlyRealtimeArrival(t *testing.T) {
	now := time.Now()
	delay := int32(300)
	data := &staticData{
		Stops:               map[string]staticStop{"R01": {ID: "R01", Name: "Station"}},
		RegularRoutesByStop: map[string]map[string]struct{}{"R01": {"W": {}}},
	}
	update := &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String("departure-only"), RouteId: proto.String("W")}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("R01N"), Departure: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(now.Unix() + 300), Delay: &delay}}}}
	service := NewTransitService(
		&StaticService{data: data, loadedAt: now},
		&RealtimeService{entities: []*gtfs.FeedEntity{{TripUpdate: update}}, loadedAt: now},
		&MetadataService{loadedAt: now},
	)
	result, err := service.Stop(context.Background(), "R01", now)
	if err != nil || len(result.StopTimes) != 1 {
		t.Fatalf("departure-only train missing: %v, %v", result.StopTimes, err)
	}
	arrival := result.StopTimes[0]
	if arrival.Arrival != nil || arrival.Departure.Time != now.Unix()+300 || arrival.DelayMinutes == nil || *arrival.DelayMinutes != 5 {
		t.Fatalf("departure event or reported delay changed: %+v", arrival)
	}
	update.StopTimeUpdate[0].Departure.Delay = nil
	trip := staticTrip{ID: "static", RouteID: "W"}
	data.TripsByRealtimeID = map[string]staticTrip{"departure-only": trip}
	seconds := currentNewYorkSecond(now)
	scheduled := fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	data.StopTimesByTrip = map[string][]stopTime{"static": {{StopID: "R01N", ArrivalTime: scheduled}}}
	result, err = service.Stop(context.Background(), "R01", now)
	if err != nil || len(result.StopTimes) != 1 || result.StopTimes[0].DelayMinutes != nil {
		t.Fatalf("absent protobuf delay invented a lateness estimate: %+v, %v", result.StopTimes, err)
	}
}

func TestOverlappingFeedsKeepCompleteTripAndLatestVehicleInFeedOrder(t *testing.T) {
	update := func(id, route string, count int) *gtfs.FeedEntity {
		return &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String(id), RouteId: proto.String(route)}, StopTimeUpdate: make([]*gtfs.TripUpdate_StopTimeUpdate, count)}}
	}
	vehicle := func(id string) *gtfs.FeedEntity {
		return &gtfs.FeedEntity{Vehicle: &gtfs.VehiclePosition{Trip: &gtfs.TripDescriptor{TripId: proto.String(id)}}}
	}
	partial, complete := update("rerouted", "E", 2), update("rerouted", "E", 5)
	other, tied := update("other", "F", 3), update("other", "F", 3)
	oldVehicle, latestVehicle := vehicle("rerouted"), vehicle("rerouted")
	got := preferredRealtimeEntities([]*gtfs.FeedEntity{partial, other, oldVehicle, complete, tied, latestVehicle, {}})
	want := []*gtfs.FeedEntity{complete, other, latestVehicle}
	if len(got) != len(want) {
		t.Fatalf("duplicated or omitted realtime entities: %v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("feed preference or order changed at index %d", index)
		}
	}
}

func TestTemporaryReroutesDoNotBecomeRegularStationBullets(t *testing.T) {
	trips := []staticTrip{}
	times := map[string][]stopTime{}
	add := func(route string, total, stopping int) {
		for index := 0; index < total; index++ {
			id := fmt.Sprintf("%s-%d", route, index)
			trips = append(trips, staticTrip{ID: id, RouteID: route})
			if index < stopping {
				times[id] = []stopTime{{StopID: "251N"}}
			}
		}
	}
	add("2", 100, 1)
	add("3", 100, 80)
	add("4", 100, 20)
	add("5", 100, 2)
	actual := sortedRoutes(regularRoutesByStop(trips, times)["251"])
	if !reflect.DeepEqual(actual, []string{"3", "4"}) {
		t.Fatalf("regular routes = %v, want 3 and 4", actual)
	}
}

func TestStationDiscoveryRetainsExpressIDsAndNumericStationOrder(t *testing.T) {
	data := &staticData{
		StationIDs:          []string{"last", "second", "first"},
		Stops:               map[string]staticStop{"last": {ID: "last", Name: "103 St"}, "second": {ID: "second", Name: "18 St"}, "first": {ID: "first", Name: "2 Av"}},
		RegularRoutesByStop: map[string]map[string]struct{}{"last": {"6": {}, "6X": {}}, "second": {"F": {}, "FX": {}}, "first": {"7": {}, "7X": {}}},
	}
	result := buildBootstrap(data, nil, time.Now())
	if !reflect.DeepEqual(result.Routes, []string{"6", "6X", "7", "7X", "F", "FX"}) {
		t.Fatalf("express service was collapsed: %v", result.Routes)
	}
	ids := []string{}
	for _, station := range result.Stations {
		ids = append(ids, station.ID)
	}
	if !reflect.DeepEqual(ids, []string{"first", "second", "last"}) {
		t.Fatalf("station numbers sorted as plain text: %v", ids)
	}
}

func TestEqualArrivalTimesKeepOriginalFeedOrder(t *testing.T) {
	now := time.Now()
	data := &staticData{Stops: map[string]staticStop{"A32": {ID: "A32", Name: "Station"}}, RegularRoutesByStop: map[string]map[string]struct{}{"A32": {"E": {}}}}
	entities := []*gtfs.FeedEntity{}
	for index := 0; index < 40; index++ {
		entities = append(entities, &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String(fmt.Sprintf("train-%02d", index)), RouteId: proto.String("E")}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("A32N"), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(now.Unix() + 60 + int64(index%3)*60)}}}}})
	}
	service := NewTransitService(&StaticService{data: data, loadedAt: now}, &RealtimeService{entities: entities, loadedAt: now}, &MetadataService{loadedAt: now})
	result, err := service.Stop(context.Background(), "A32", now)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{}
	for group := 0; group < 3; group++ {
		for index := group; index < 40; index += 3 {
			expected = append(expected, fmt.Sprintf("train-%02d", index))
		}
	}
	actual := []string{}
	for _, arrival := range result.StopTimes {
		actual = append(actual, arrival.Trip.ID)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("equal-time arrivals changed order: %v", actual)
	}
}

func TestStationHeadingAndArrivalStopsKeepTheirOriginalAccessibilitySources(t *testing.T) {
	now := time.Now()
	data := &staticData{Stops: map[string]staticStop{"A32": {ID: "A32", Name: "Station", Accessible: true}}, RegularRoutesByStop: map[string]map[string]struct{}{"A32": {"E": {}}}}
	entity := &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: &gtfs.TripDescriptor{TripId: proto.String("live-train"), RouteId: proto.String("E")}, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("A32N"), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(now.Unix() + 60)}}}}}
	service := NewTransitService(&StaticService{data: data, loadedAt: now}, &RealtimeService{entities: []*gtfs.FeedEntity{entity}, loadedAt: now}, &MetadataService{loadedAt: now})
	result, err := service.Stop(context.Background(), "A32", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accessible {
		t.Fatal("station heading replaced metadata with static GTFS accessibility")
	}
	if len(result.StopTimes) != 1 || !result.StopTimes[0].Stop.Accessible || !result.StopTimes[0].Destination.Accessible {
		t.Fatal("arrival stop objects replaced GTFS accessibility with station metadata")
	}
}
