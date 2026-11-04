import assert from "node:assert/strict";
import test from "node:test";
import { buildRouteTopology } from "../src/lib/routeTopology.ts";

const stop = (id) => ({ id, name: id });
const branch = (id, stopIds, extra = {}) => ({
  id,
  stops: stopIds.map(stop),
  ...extra,
});

test("maps the A route branches around Broad Channel", () => {
  const topology = buildRouteTopology(
    [stop("A01"), stop("A02")],
    [
      branch("lefferts", ["A03", "A04"]),
      branch("far-rockaway", ["A05", "H04", "H06", "H07"]),
      branch("rockaway-park", ["H04", "H12"]),
    ],
    "A"
  );

  assert.equal(topology.kind, "a");
  assert.deepEqual(topology.farRockawayTrunk.map(({ id }) => id), ["A05", "H04"]);
  assert.deepEqual(topology.farRockawayTail.map(({ id }) => id), ["H06", "H07"]);
  assert.equal(topology.hasRockawayParkSplit, true);
});

test("splits the 5 route at East 180 St", () => {
  const topology = buildRouteTopology(
    [stop("201"), stop("213"), stop("214")],
    [branch("nereid", ["213", "215"])],
    "5"
  );

  assert.equal(topology.kind, "five");
  assert.deepEqual(topology.trunkBefore.map(({ id }) => id), ["201"]);
  assert.deepEqual(topology.trunkAfter.map(({ id }) => id), ["213", "214"]);
});

test("places temporary branches on the configured side of their junction", () => {
  const topology = buildRouteTopology(
    [stop("1"), stop("2"), stop("3")],
    [
      branch("before", ["B1"], { temporary: true, junctionId: "2", placement: "before" }),
      branch("after", ["A1"], { temporary: true, junctionId: "2", placement: "after" }),
    ],
    "Q"
  );

  assert.equal(topology.kind, "temporary");
  assert.deepEqual(topology.trunkBefore.map(({ id }) => id), ["1"]);
  assert.deepEqual(topology.trunkAfter.map(({ id }) => id), ["2", "3"]);
  assert.deepEqual(topology.branchesBefore.map(({ id }) => id), ["before"]);
  assert.deepEqual(topology.branchesAfter.map(({ id }) => id), ["after"]);
});

