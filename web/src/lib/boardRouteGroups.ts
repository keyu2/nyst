import type { StationService } from "./api";
import { routeFamily } from "./routes.ts";

function uniqueRouteIds(routeIds: string[]) {
  return [...new Set(routeIds.filter(Boolean))].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));
}

export function createBoardRouteGroups(routeIds: string[], stationServices: StationService[] = []) {
  const groups: { id: string; members: string[] }[] = [];
  const groupedFamilies = new Set<string>();
  const availableFamilies = new Set(routeIds.map(routeFamily));
  for (const service of stationServices) {
    const members = uniqueRouteIds((service.routes || []).filter((routeId) => availableFamilies.has(routeFamily(routeId))));
    const families = uniqueRouteIds(members.map(routeFamily));
    if (!families.length || families.every((family) => groupedFamilies.has(family))) continue;
    groups.push({ id: families.join("-"), members });
    for (const family of families) groupedFamilies.add(family);
  }
  for (const family of uniqueRouteIds(routeIds.map(routeFamily))) {
    if (groupedFamilies.has(family)) continue;
    groups.push({ id: family, members: uniqueRouteIds(routeIds.filter((routeId) => routeFamily(routeId) === family)) });
    groupedFamilies.add(family);
  }
  return groups;
}

export function createStationRouteFamilies(realtimeRouteIds: string[], stationServices: StationService[] = [], requestedRouteId = "") {
  const fallbackRouteIds = stationServices.flatMap((service) => service.routes || []).concat(requestedRouteId);
  const routeIds = uniqueRouteIds(realtimeRouteIds.length ? realtimeRouteIds : fallbackRouteIds);
  const routeFamilies = new Map<string, string[]>();
  for (const routeId of routeIds) {
    const familyId = routeFamily(routeId);
    routeFamilies.set(familyId, [...(routeFamilies.get(familyId) || []), routeId]);
  }
  return { routeIds, routeFamilies, familyIds: [...routeFamilies.keys()] };
}

export function createStationRouteGroups(familyIds: string[]) {
  const trunks = [
    ["1", "2", "3"], ["4", "5", "6"], ["7"],
    ["A", "C", "E"], ["B", "D", "F", "M"],
    ["J", "Z"], ["N", "Q", "R", "W"],
  ];
  const groups: string[][] = [];
  for (const family of familyIds) {
    const trunk = trunks.find((members) => members.includes(family)) || [family];
    if (groups.some((members) => members.includes(family))) continue;
    groups.push(trunk.filter((member) => familyIds.includes(member)));
  }
  return groups;
}
