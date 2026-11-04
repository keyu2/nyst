import type { Station } from "./api";

const routeAliases: Record<string, string> = { S: "GS", SR: "H", SF: "FS", SIR: "SI", SS: "SIX" };
const routeFamilies: Record<string, string> = { "6X": "6", "7X": "7", FX: "F", SIX: "SI" };

export function routeFamily(route: string) {
  const normalized = routeAliases[route] || route;
  return routeFamilies[normalized] || normalized;
}

function routeSort(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true });
}

export function stationRouteIds(station: Station) {
  return [...new Set(station.services.flatMap((service) => [
    ...(service.routes||[]),
    ...(service.realtimeRoutes||[]),
  ].map(routeFamily)))].sort(routeSort);
}

export function stationActiveRouteIds(station: Station) {
  const availabilityKnown = station.services.some((service) => (
    Array.isArray(service.scheduledRoutes) || Array.isArray(service.realtimeRoutes)
  ));
  if (!availabilityKnown) return stationRouteIds(station);
  const realtimeRoutes = [...new Set(station.services.flatMap((service) => (service.realtimeRoutes||[]).map(routeFamily)))];
  const activeRoutes = realtimeRoutes.length
    ? realtimeRoutes
    : [...new Set(station.services.flatMap((service) => (service.scheduledRoutes||[]).map(routeFamily)))];
  return activeRoutes.sort(routeSort);
}

export function searchDiscovery(stations: Station[], routeIds: string[], query: string, limit = 12) {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return { stations: [] as Station[], routes: [] as string[] };
  const availableRoutes = [...new Set(routeIds.map(routeFamily))];
  const routeByLowercaseId = new Map(availableRoutes.map((id) => [id.toLowerCase(), id]));
  const tokens = normalized.split(/\s+/);
  const requestedRoutes = tokens.map((token) => routeByLowercaseId.get(token)).filter((route): route is string => Boolean(route));
  const stationTerms = tokens.filter((token) => !routeByLowercaseId.has(token));
  const routes = availableRoutes.filter((id) => id.toLowerCase() === normalized).sort(routeSort);
  const matchingStations = stations.filter((station) => {
    if (stationTerms.length === 0) return requestedRoutes.length === 0;
    const name = station.name.toLowerCase();
    if (!stationTerms.every((term) => name.includes(term))) return false;
    if (!requestedRoutes.length) return true;
    const stationRoutes = new Set(stationRouteIds(station));
    return requestedRoutes.every((id) => stationRoutes.has(id));
  });
  return { stations: matchingStations.slice(0, limit), routes };
}

export function nearestStations(stations:Station[],location:{latitude:number;longitude:number}|null,limit=3){
  if(!location)return [];
  const radians=(value:number)=>value*Math.PI/180;
  return stations.map((station)=>{const latitude=radians(station.latitude-location.latitude);const longitude=radians(station.longitude-location.longitude);const a=Math.sin(latitude/2)**2+Math.cos(radians(location.latitude))*Math.cos(radians(station.latitude))*Math.sin(longitude/2)**2;return{...station,distance:3958.8*2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a))}}).sort((a,b)=>a.distance-b.distance||a.name.localeCompare(b.name)).slice(0,limit);
}
