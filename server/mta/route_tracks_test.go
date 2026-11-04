package mta

import (
	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
	"reflect"
	"testing"
	"time"
)

func TestGunHillExpressColorsTrackButNotSkippedStopsOrTrackBeyondTerminal(t *testing.T) {
	for _, ids := range [][]string{{"214", "213", "208"}, {"208", "213", "214"}} {
		main := []RouteStop{{StopObject: StopObject{ID: "501"}}, {StopObject: StopObject{ID: "505"}}, {StopObject: StopObject{ID: "213"}}, {StopObject: StopObject{ID: "214"}}}
		branch := RouteBranch{ID: "nereid"}
		for _, id := range []string{"204", "205", "206", "207", "208", "209", "210", "211", "212"} {
			branch.Stops = append(branch.Stops, RouteStop{StopObject: StopObject{ID: id}, HasService: proto.Bool(id == "208")})
		}
		applyTrackService("5", main, []RouteBranch{branch}, []stopPattern{{StopIDs: ids}}, true)
		for i, stop := range branch.Stops {
			above, below := i > 4, i >= 4
			if stop.TrackServiceAbove == nil || *stop.TrackServiceAbove != above || stop.TrackServiceBelow == nil || *stop.TrackServiceBelow != below {
				t.Fatalf("%s track above=%v below=%v", stop.ID, stop.TrackServiceAbove, stop.TrackServiceBelow)
			}
			if *stop.HasService != (i == 4) {
				t.Fatalf("skipped station %s was marked served", stop.ID)
			}
		}
	}
}

func TestSeparateLiveSectionsLeaveGapInactive(t *testing.T) {
	items := []RouteStop{}
	for _, id := range []string{"one", "two", "three", "four"} {
		items = append(items, RouteStop{StopObject: StopObject{ID: id}})
	}
	applyTrackService("D", items, nil, []stopPattern{{StopIDs: []string{"one", "two"}}, {StopIDs: []string{"three", "four"}}}, true)
	if *items[1].TrackServiceBelow || *items[2].TrackServiceAbove || !*items[0].TrackServiceBelow {
		t.Fatal("separate operating sections were connected")
	}
}

func TestEmptyRealtimeTrackServiceDiffersFromUnavailableFeed(t *testing.T) {
	items := []RouteStop{{StopObject: StopObject{ID: "one"}}, {StopObject: StopObject{ID: "two"}}}
	applyTrackService("E", items, nil, nil, false)
	if items[0].TrackServiceBelow != nil {
		t.Fatal("unavailable feed was treated as confirmed no service")
	}
	applyTrackService("E", items, nil, nil, true)
	if items[0].TrackServiceBelow == nil || *items[0].TrackServiceBelow {
		t.Fatal("empty live feed activated track")
	}
}

func TestABranchesFollowFutureRealtimeInsteadOfScheduledOrPastTrains(t *testing.T) {
	data := &staticData{StopTimesByTrip: map[string][]stopTime{}}
	candidates := []staticTrip{}
	for id, stops := range map[string][]string{
		"lefferts": {"A02", "A61", "A63", "A65"},
		"far":      {"A02", "A61", "H02", "H04", "H06", "H11"},
		"park":     {"A02", "A61", "H02", "H04", "H12", "H15"},
	} {
		candidates = append(candidates, staticTrip{ID: id})
		for _, stop := range stops {
			data.StopTimesByTrip[id] = append(data.StopTimesByTrip[id], stopTime{StopID: stop, ArrivalTime: "10:00:00", DepartureTime: "10:00:00"})
		}
	}
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	entity := func(route, stop string, offset int64, canceled bool) *gtfs.FeedEntity {
		trip := &gtfs.TripDescriptor{RouteId: proto.String(route)}
		if canceled {
			trip.ScheduleRelationship = gtfs.TripDescriptor_CANCELED.Enum()
		}
		return &gtfs.FeedEntity{TripUpdate: &gtfs.TripUpdate{Trip: trip, StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{StopId: proto.String(stop), Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: proto.Int64(now.Unix() + offset)}}}}}
	}
	build := func(entities []*gtfs.FeedEntity, known bool) []string {
		_, branches := buildARouteBranches(candidates, candidates, data, []string{"A02", "A61"}, nil, entities, now, known, func(id string) RouteStop { return RouteStop{StopObject: StopObject{ID: id}} })
		ids := []string{}
		for _, branch := range branches {
			ids = append(ids, branch.ID)
		}
		return ids
	}
	if got := build([]*gtfs.FeedEntity{entity("A", "H11S", 100, false), entity("A", "H15S", -100, false)}, true); !reflect.DeepEqual(got, []string{"far-rockaway"}) {
		t.Fatalf("branches=%v", got)
	}
	if got := build(nil, true); len(got) != 0 {
		t.Fatalf("empty live feed restored scheduled branches: %v", got)
	}
	if got := build(nil, false); len(got) != 3 {
		t.Fatalf("unavailable feed lost schedule fallback: %v", got)
	}
	if got := build([]*gtfs.FeedEntity{entity("A", "H15S", 100, true), entity("SR", "H15S", 100, false)}, true); len(got) != 0 {
		t.Fatalf("canceled or shuttle service restored A branch: %v", got)
	}
	if got := build([]*gtfs.FeedEntity{entity("A", "H12N", 100, false)}, true); !reflect.DeepEqual(got, []string{"far-rockaway", "rockaway-park"}) {
		t.Fatalf("live park branch missing: %v", got)
	}
}

func TestExpressRepresentativeCannotShortcutBranchTrack(t *testing.T) {
	main := []RouteStop{}
	for _, id := range []string{"501", "505", "213", "208", "214"} {
		main = append(main, RouteStop{StopObject: StopObject{ID: id}})
	}
	branch := RouteBranch{ID: "nereid"}
	for _, id := range []string{"204", "205", "206", "207", "208", "209", "210", "211", "212"} {
		branch.Stops = append(branch.Stops, RouteStop{StopObject: StopObject{ID: id}, HasService: proto.Bool(false)})
	}
	applyTrackService("5", main, []RouteBranch{branch}, []stopPattern{{StopIDs: []string{"214", "213", "208"}}}, true)
	for _, stop := range branch.Stops[5:] {
		if *stop.HasService || !*stop.TrackServiceAbove || !*stop.TrackServiceBelow {
			t.Fatalf("skipped stop %s lost its traversed track", stop.ID)
		}
	}
}
