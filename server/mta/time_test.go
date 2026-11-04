package mta

import (
	"testing"
	"time"
)

func TestServiceDayStartMatchesOriginalAcrossDaylightSavingChanges(t *testing.T) {
	for _, test := range []struct{ date, expected string }{
		{"20260308", "2026-03-08T04:10:00Z"},
		{"20261101", "2026-11-01T05:10:00Z"},
		{"20260905", "2026-09-05T04:10:00Z"},
	} {
		want, err := time.Parse(time.RFC3339, test.expected)
		if err != nil {
			t.Fatal(err)
		}
		if got := tripStartTimestamp(test.date, 600); got != want.Unix() {
			t.Fatalf("%s service time=%s; want %s", test.date, time.Unix(got, 0).UTC(), want)
		}
	}
	if tripStartTimestamp("20260904", 24*3600+600) != tripStartTimestamp("20260905", 600) {
		t.Fatal("after-midnight trip moved to a different service date")
	}
}
