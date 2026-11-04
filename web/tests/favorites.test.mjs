import assert from "node:assert/strict";
import test from "node:test";
import { favoriteLines, favoriteStations, isFavoriteStation, toggleFavoriteLine, toggleFavoriteStation, watchFavorites } from "../src/lib/favorites.ts";

function storage() {
  const values = new Map();
  let cookie = "";
  globalThis.localStorage = { getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
  globalThis.document = { get cookie() { return cookie; }, set cookie(value) { cookie = value.split(";")[0]; } };
  globalThis.window = new EventTarget();
}

test("favorite line listeners see the newly saved cookie and selection immediately", () => {
  storage();
  const selections = [];
  const stop = watchFavorites(() => selections.push(favoriteLines()));
  toggleFavoriteLine("E");
  toggleFavoriteLine("N");
  toggleFavoriteLine("E");
  stop();
  assert.deepEqual(selections, [["E"], ["E", "N"], ["N"]]);
  assert.deepEqual(favoriteLines(), ["N"]);
});

test("favorite stations recognize the same complex across its platform entries", () => {
  storage();
  const station = { id: "R09", ids: ["R09", "718"], name: "Queensboro Plaza", route: "N", routes: ["N", "7"] };
  toggleFavoriteStation(station);
  assert.equal(isFavoriteStation(["718"]), true);
  toggleFavoriteStation({ ...station, id: "718" });
  assert.deepEqual(favoriteStations(), []);
});

test("an older station favorite without platform IDs can still be removed", () => {
  storage();
  localStorage.setItem("subway-favorite-stations", JSON.stringify([{ id: "A32", ids: [], name: "W 4 St", route: "E" }]));
  assert.equal(isFavoriteStation(["A32"]), true);
  toggleFavoriteStation({ id: "A32", ids: ["A32"], name: "W 4 St", route: "E", routes: ["E"] });
  assert.deepEqual(favoriteStations(), []);
});
