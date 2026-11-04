export type LiveStatus = {
  alertCount: number;
  updatedAt: string;
  lastSuccessfulAt: string;
  consecutiveFailures: number;
  stale: boolean;
};

export type StationService = {
  id: string;
  routes: string[];
  scheduledRoutes: string[];
  realtimeRoutes: string[];
};

export type Station = {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
  services: StationService[];
};

export type Bootstrap = {
  routes: string[];
  stations: Station[];
  generatedAt: string;
  routeAlerts: Record<string, RouteAlert[]>;
  routeStatuses: Record<string, string>;
};
export type RouteAlert = { id:string; header:string; description:string; alertType:string; timeLabel:string; category:string };

export type StopObject = {
  id: string;
  name: string;
  latitude: number | null;
  longitude: number | null;
  accessible: boolean;
  borough?: string;
};
export type Arrival = {
  stop: StopObject;
  arrival?: { time: number };
  departure?: { time: number };
  future: boolean;
  notDeparted: boolean;
  delayMinutes?: number;
  trip: { id: string; route: { id: string; color: string } };
  destination: StopObject;
  headsign: string;
  transferRoutes?: string[];
  estimated?: boolean;
};
export type StopResponse = {
  id: string;
  name: string;
  borough?: string;
  accessible: boolean;
  stopTimes: Arrival[];
  services: StationService[];
  transfers: { stop: StopObject; routes: string[] }[];
  generatedAt: number;
};
export type RouteStop = StopObject & { transferRoutes: string[]; hasService: boolean | null; trackServiceAbove?: boolean; trackServiceBelow?: boolean };
export type RouteBranch = { id:string; name:string; placement?:string; junctionId?:string; temporary?:boolean; stops:RouteStop[] };
export type RouteResponse = { id:string; color:string; serviceMaps:{configId:string;stops:RouteStop[]}[]; branches:RouteBranch[]; alerts:RouteAlert[];serviceAvailable:boolean|null };
export type TripResponse = { id:string; updatedAt:number; origin:StopObject; route:{id:string;color:string}; mapCoordinates:number[][]; stopTimes:Arrival[]; vehicle:MapVehicle|null };
export type MapCoordinate = [number, number];
export type MapFeature = { type:"Feature"; properties:{routeId:string;color:string;offset:number;laneOffsets?:number[]}; geometry:{type:"LineString";coordinates:MapCoordinate[]} };
export type MapResponse = { lines:MapFeature[]; stations:Station[]; vehicles:MapVehicle[] };
export type MapVehicle = { id:string;routeId:string;longitude:number;latitude:number;directionBearing:number;bearing?:number;offset?:number };
export type StationMapResponse = { lines:MapFeature[];station:Station;vehicles:MapVehicle[] };

export async function getLiveStatus(signal?: AbortSignal): Promise<LiveStatus> {
  const response = await fetch("/api/status", { signal });
  if (!response.ok) {
    throw new Error(`Live status request failed: ${response.status}`);
  }
  return response.json();
}

export async function getBootstrap(signal?: AbortSignal): Promise<Bootstrap> {
  const response = await fetch("/api/bootstrap", { signal });
  if (!response.ok) {
    throw new Error(`Bootstrap request failed: ${response.status}`);
  }
  return response.json();
}

export async function getStop(id: string, signal?: AbortSignal): Promise<StopResponse> {
  const response = await fetch(`/api/stops/${encodeURIComponent(id)}`, { signal });
  if (!response.ok) throw new Error(`Stop request failed: ${response.status}`);
  return response.json();
}
export async function getRoute(id: string, signal?: AbortSignal): Promise<RouteResponse> {
  const response = await fetch(`/api/routes/${encodeURIComponent(id)}`, { signal });
  if (!response.ok) throw new Error(`Route request failed: ${response.status}`);
  return response.json();
}
export async function getRouteAlerts(id: string, signal?: AbortSignal): Promise<RouteAlert[]> {
  const response = await fetch(`/api/routes/${encodeURIComponent(id)}/alerts`, { signal });
  if (!response.ok) throw new Error(`Route alerts request failed: ${response.status}`);
  return response.json();
}

export async function getTrip(routeId:string, tripId:string, signal?:AbortSignal):Promise<TripResponse> {
  const response=await fetch(`/api/trips/${encodeURIComponent(routeId)}/${encodeURIComponent(tripId)}`,{signal});
  if(!response.ok) throw new Error(`Trip request failed: ${response.status}`);
  return response.json();
}
export async function getMap(signal?:AbortSignal):Promise<MapResponse> {
  const response=await fetch("/api/map",{signal});
  if(!response.ok) throw new Error(`Map request failed: ${response.status}`);
  return response.json();
}
export async function getStationMap(id:string,signal?:AbortSignal):Promise<StationMapResponse> {
  const response=await fetch(`/api/maps/stations/${encodeURIComponent(id)}`,{signal});
  if(!response.ok) throw new Error(`Station map request failed: ${response.status}`);
  return response.json();
}
