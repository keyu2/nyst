import test from "node:test";
import assert from "node:assert/strict";
import { getBullet, hasTransferRouteChange, normalizeRouteId, routeFamily } from "../src/lib/routes.ts";

test("normalizes MTA shuttle route identifiers", () => {
  assert.equal(normalizeRouteId("S"), "GS");
  assert.equal(normalizeRouteId("SF"), "FS");
  assert.equal(normalizeRouteId("SR"), "H");
  assert.equal(normalizeRouteId("SS"), "SIX");
  assert.equal(normalizeRouteId("A"), "A");
});

test("groups express variants with their route family", () => {
  assert.equal(routeFamily("7X"), "7");
  assert.equal(routeFamily("6X"), "6");
  assert.equal(routeFamily("FX"), "F");
  assert.equal(routeFamily("SIX"), "SI");
});

test("resolves public bullet assets", () => {
  assert.equal(getBullet("7X"), "/bullets/7d.svg");
  assert.equal(getBullet("GS"), "/bullets/S.svg");
  assert.equal(getBullet("A"), "/bullets/A.svg");
  assert.equal(getBullet("SIX"), "/bullets/SIRd.svg");
});

test("detects the start and end of shared transfer-service runs", () => {
  assert.equal(hasTransferRouteChange(["Q"], ["Q"], ["Q"]), false);
  assert.equal(hasTransferRouteChange(["Q"], [], ["Q"]), true);
  assert.equal(hasTransferRouteChange(["Q", "SF"], ["Q"], ["Q"]), true);
});

