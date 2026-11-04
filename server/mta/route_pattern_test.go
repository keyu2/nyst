package mta

import (
	"reflect"
	"testing"
)

func TestSplitServiceAlertPreservesEachSectionAndLeavesGapsInactive(t *testing.T) {
	stops := []StopObject{{ID: "D01", Name: "Norwood-205 St"}, {ID: "D17", Name: "34 St-Herald Sq"}, {ID: "D20", Name: "W 4 St-Wash Sq"}, {ID: "D21", Name: "Broadway-Lafayette St"}, {ID: "D22", Name: "Grand St"}, {ID: "R30", Name: "DeKalb Av"}, {ID: "R31", Name: "Atlantic Av-Barclays Ctr"}, {ID: "D43", Name: "Coney Island-Stillwell Av"}}
	alerts := []RouteAlert{{Header: "D service operates in three sections:", Description: "1. Between Norwood-205 St and 34 St-Herald Sq\n2. Shuttle train between W 4 St-Wash Sq and Grand St, stopping at Broadway-Lafayette St, every 20 minutes\n3. Between Atlantic Av-Barclays Ctr and Coney Island-Stillwell Av"}}
	active := alertSectionStopIDs(alerts, stops)
	want := []string{"D01", "D17", "D20", "D21", "D22", "R31", "D43"}
	if len(active) != len(want) {
		t.Fatalf("active stops=%v", active)
	}
	for _, id := range want {
		if _, ok := active[id]; !ok {
			t.Fatalf("missing active section stop %s", id)
		}
	}
	if _, ok := active["R30"]; ok {
		t.Fatal("service gap marked active")
	}
	if isSplitServiceAlert(RouteAlert{Description: "No D between 34 St and Atlantic Av"}) {
		t.Fatal("ordinary suspension treated as split service")
	}
}

func TestRealtimeBridgeRerouteConnectsBothEnds(t *testing.T) {
	normal := []string{"R31", "R30", "R29", "R28", "R27", "R26", "R25", "R24", "R23", "R22", "R21"}
	got := applyRealtimeDetour(normal, []stopPattern{{StopIDs: []string{"R31", "R30", "Q01", "R22", "R21"}}})
	want := []string{"R31", "R30", "Q01", "R22", "R21"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRealtimeTerminalOutsideNormalRouteDoesNotBreakRoutePage(t *testing.T) {
	base := []string{"north", "junction", "south"}
	for _, stops := range [][]string{
		{"north", "junction", "branch-near", "branch-terminal"},
		{"branch-terminal", "branch-near", "junction", "north"},
		{"branch-terminal", "branch-near", "junction", "south"},
	} {
		t.Run(stringsJoin(stops, ","), func(t *testing.T) {
			got := applyRealtimeDetour(base, []stopPattern{{StopIDs: stops}})
			if !reflect.DeepEqual(got, base) {
				t.Fatalf("terminal branch replaced normal route: got %v", got)
			}
		})
	}
}

func TestRealtimeDetourRemainsConnectedWhenTripContinuesToNewTerminal(t *testing.T) {
	base := []string{"north", "first", "skipped", "second", "south"}
	got := applyRealtimeDetour(base, []stopPattern{{StopIDs: []string{"north", "first", "rerouted", "second", "branch-near", "branch-terminal"}}})
	want := []string{"north", "first", "rerouted", "second", "south"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func TestShortTurnExtendsOnlyToFrequentCompleteRoute(t *testing.T) {
	short := stopPattern{Count: 8, StopIDs: []string{"604", "603", "602", "601"}}
	full := stopPattern{Count: 12, StopIDs: []string{"606", "605", "604", "603", "602", "601"}}
	got := extendShortTurnPattern(&short, []stopPattern{short, full})
	if !reflect.DeepEqual(got.StopIDs, full.StopIDs) {
		t.Fatalf("got %v", got.StopIDs)
	}
}
func TestTemporaryTerminalBranch(t *testing.T) {
	got := offRouteTerminalBranches([]string{"pelham", "hunts", "whitlock", "125", "south"}, []stopPattern{{StopIDs: []string{"149-hostos", "138", "125", "south"}}, {StopIDs: []string{"south", "125", "138", "149-hostos"}}, {StopIDs: []string{"138", "125", "south"}}, {StopIDs: []string{"pelham", "hunts"}}})
	want := []terminalBranch{{JunctionID: "125", TerminalID: "149-hostos", Placement: "before", StopIDs: []string{"149-hostos", "138"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}
