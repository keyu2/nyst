package mta

import (
	"testing"
	"time"
)

func TestActiveServiceIDsUsesNewYorkCalendarAndExceptions(t *testing.T) {
	data := &staticData{
		Calendar: []calendarService{
			{ID: "weekday", StartDate: "20261001", EndDate: "20261031", Weekdays: [7]bool{true, true, true, true, true, false, false}},
			{ID: "weekend", StartDate: "20261001", EndDate: "20261031", Weekdays: [7]bool{false, false, false, false, false, true, true}},
		},
		CalendarDates: []calendarException{
			{ServiceID: "weekday", Date: "20261012", Type: "2"},
			{ServiceID: "planned-work", Date: "20261012", Type: "1"},
		},
	}
	instant := time.Date(2026, 10, 13, 2, 30, 0, 0, time.UTC) // Still Oct 12 in New York.
	active := activeServiceIDs(data, instant)
	if _, found := active["weekday"]; found {
		t.Fatal("removed service remained active")
	}
	if _, found := active["planned-work"]; !found {
		t.Fatal("added service was not active")
	}
	if _, found := active["weekend"]; found {
		t.Fatal("weekend service was active on Monday")
	}
}

func TestRegularRoutesIgnoreRarePlannedWorkTrips(t *testing.T) {
	trips := make([]staticTrip, 0, 21)
	times := map[string][]stopTime{}
	for index := 0; index < 20; index++ {
		id := "regular-" + string(rune('a'+index))
		trips = append(trips, staticTrip{ID: id, RouteID: "F"})
		times[id] = []stopTime{{StopID: "F01N"}}
	}
	trips = append(trips, staticTrip{ID: "reroute", RouteID: "F"})
	times["reroute"] = []stopTime{{StopID: "A01S"}}
	routes := regularRoutesByStop(trips, times)
	if _, found := routes["F01"]["F"]; !found {
		t.Fatal("regular F stop was omitted")
	}
	if _, found := routes["A01"]["F"]; found {
		t.Fatal("rare planned-work stop became regular F service")
	}
}

func TestRouteFamiliesMatchExistingBulletLogic(t *testing.T) {
	cases := map[string]string{"7X": "7", "6X": "6", "FX": "F", "SIR": "SI", "SF": "FS"}
	for route, want := range cases {
		if got := routeFamily(route); got != want {
			t.Fatalf("routeFamily(%q) = %q, want %q", route, got, want)
		}
	}
}
