import test from "node:test";
import assert from "node:assert/strict";
import { formatArrivalCountdown, gtfsMinute, gtfsSecond, tripIdStartSecond, tripStartTimestamp, unixSeconds } from "../src/lib/time.ts";

test("uses the trip service date across midnight", () => {
  assert.equal(tripStartTimestamp("20260905", 600), Date.parse("2026-09-05T00:10:00-04:00") / 1000);
  assert.equal(tripStartTimestamp("20260904", 24 * 3600 + 600), tripStartTimestamp("20260905", 600));
  assert.equal(tripStartTimestamp(undefined, 600), null);
  assert.equal(tripStartTimestamp("20260905", null), null);
});

test("parses standard and after-midnight GTFS times", () => {
  assert.equal(gtfsMinute("09:31:00"), 571);
  assert.equal(gtfsSecond("25:05:30"), 90330);
  assert.equal(gtfsSecond(""), null);
});

test("normalizes protobuf and numeric Unix timestamps", () => {
  assert.equal(unixSeconds(123), 123);
  assert.equal(unixSeconds({ toNumber: () => 456 }), 456);
  assert.equal(unixSeconds(null), undefined);
});

test("formats arrival countdowns with seconds and hours", () => {
  assert.equal(formatArrivalCountdown(20), "0m");
  assert.equal(formatArrivalCountdown(20, true), "0m 20s");
  assert.equal(formatArrivalCountdown(3723), "1h 2m");
  assert.equal(formatArrivalCountdown(3723, true), "1h 2m 3s");
});

test("reads scheduled start times encoded in MTA trip IDs", () => {
  assert.equal(tripIdStartSecond("106900_R..N"), 17 * 3600 + 49 * 60);
  assert.equal(tripIdStartSecond("106950_1..S15R"), 17 * 3600 + 49 * 60 + 30);
  assert.equal(tripIdStartSecond("not-encoded"), null);
});

