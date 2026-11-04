import test from "node:test";
import assert from "node:assert/strict";
import {
  directionGroupLabel,
  disambiguateDirectionGroupLabels,
} from "../src/lib/directionLabels.ts";

test("uses uptown and downtown language for Manhattan stations", () => {
  assert.equal(directionGroupLabel("Manhattan", "Bronx", "northbound"), "Uptown / The Bronx");
  assert.equal(directionGroupLabel("Manhattan", "Brooklyn", "southbound"), "Downtown / Brooklyn");
  assert.equal(directionGroupLabel("Manhattan", "Manhattan", "southbound"), "Downtown");
});

test("uses borough destinations outside Manhattan", () => {
  assert.equal(directionGroupLabel("Brooklyn", "Brooklyn", "southbound"), "Brooklyn");
  assert.equal(directionGroupLabel("Brooklyn", "Manhattan", "northbound"), "Manhattan");
  assert.equal(directionGroupLabel("Bronx", "Manhattan", "southbound"), "Manhattan");
  assert.equal(directionGroupLabel("Queens", "Queens", "northbound"), "Queens");
});

test("falls back to cardinal subway language when borough metadata is unavailable", () => {
  assert.equal(directionGroupLabel(null, null, "northbound"), "Uptown");
  assert.equal(directionGroupLabel(null, null, "southbound"), "Downtown");
});

test("uses actual termini when borough labels would be identical", () => {
  const groups = disambiguateDirectionGroupLabels([
    { id: "northbound", label: "Brooklyn", options: [{ label: "Bedford-Nostrand Avs" }] },
    { id: "southbound", label: "Brooklyn", options: [{ label: "Church Av" }] },
  ]);

  assert.deepEqual(groups.map((group) => group.label), [
    "Bedford-Nostrand Avs",
    "Church Av",
  ]);
  assert.deepEqual(groups.map((group) => group.options[0].label), [
    "Bedford-Nostrand Avs",
    "Church Av",
  ]);
});

test("lists each actual terminus for a direction with multiple branches", () => {
  const groups = disambiguateDirectionGroupLabels([
    {
      id: "northbound",
      label: "Brooklyn",
      options: [{ label: "Bedford-Nostrand Avs" }, { label: "Court Sq" }],
    },
    { id: "southbound", label: "Brooklyn", options: [{ label: "Church Av" }] },
  ]);

  assert.equal(groups[0].label, "Bedford-Nostrand Avs / Court Sq");
  assert.equal(groups[1].label, "Church Av");
});

