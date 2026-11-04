package mta

import (
	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

func TestSIRExpressUsesSkippedLocalStopsInEitherDirection(t *testing.T) {
	data := &staticData{SIRLocalStopIDs: []string{"S01", "S02", "S03"}}
	pattern := func(ids ...string) []*gtfs.TripUpdate_StopTimeUpdate {
		result := []*gtfs.TripUpdate_StopTimeUpdate{}
		for _, id := range ids {
			result = append(result, &gtfs.TripUpdate_StopTimeUpdate{StopId: proto.String(id)})
		}
		return result
	}
	if displayTripRouteID("SI", staticTrip{}, pattern("S01N", "S02N", "S03N"), data) != "SI" {
		t.Fatal("local service got an express bullet")
	}
	if displayTripRouteID("SI", staticTrip{}, pattern("S03S", "S01S"), data) != "SIX" {
		t.Fatal("express service lost its express bullet")
	}
	if displayTripRouteID("F", staticTrip{}, pattern("S01N", "S03N"), data) != "F" {
		t.Fatal("SIR classifier changed another route")
	}
}

func TestRealtimeRerouteOverridesMatchingSequenceNumbers(t *testing.T) {
	static := []stopTime{{StopID: "A01N", Sequence: 1}, {StopID: "A02N", Sequence: 2}, {StopID: "A03N", Sequence: 3}}
	stop := func(id string, sequence uint32) *gtfs.TripUpdate_StopTimeUpdate {
		return &gtfs.TripUpdate_StopTimeUpdate{StopId: proto.String(id), StopSequence: proto.Uint32(sequence)}
	}
	if realtimePathDiffers([]*gtfs.TripUpdate_StopTimeUpdate{stop("A01N", 1), stop("A03N", 3)}, static) {
		t.Fatal("sparse updates should retain the scheduled path")
	}
	if !realtimePathDiffers([]*gtfs.TripUpdate_StopTimeUpdate{stop("A01N", 1), stop("X02N", 2), stop("A03N", 3)}, static) {
		t.Fatal("rerouted stop was overwritten by the schedule")
	}
	if !realtimePathDiffers([]*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("X02N")}}, static) {
		t.Fatal("reroute without sequence was ignored")
	}
	trimmed := trimStaticToTerminal([]*gtfs.TripUpdate_StopTimeUpdate{stop("A01N", 1), stop("A02N", 2)}, static)
	if len(trimmed) != 2 || trimmed[1].StopID != "A02N" {
		t.Fatal("scheduled stops remained beyond the realtime terminal")
	}
}

func TestApproximateTripMatchDoesNotAssignTimeBySequence(t *testing.T) {
	static := []stopTime{{StopID: "A01N", Sequence: 1, ArrivalTime: "10:00:00"}}
	realtime := []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String("X01N"), StopSequence: proto.Uint32(1), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(200)}}}
	result := mergeStaticRealtime(static, realtime, &gtfs.TripUpdate{}, &staticData{}, StationMetadata{}, time.Unix(150, 0), false)
	if len(result) != 1 || result[0].Stop.ID != "A01N" || result[0].Arrival != nil {
		t.Fatal("an approximate trip match reassigned an unrelated stop's time")
	}
}
