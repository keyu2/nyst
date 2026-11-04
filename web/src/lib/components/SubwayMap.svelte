<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import type { MapFeature, MapVehicle, Station } from "../api";
  import { createSubwayMap, selectSubwayMapStation, updateSubwayMapData, updateSubwayMapTheme, type VehicleSelection } from "../subwayMapClient";
  import { stationActiveRouteIds } from "../stationDiscovery";
  import { readPreferences, watchPreferences, effectiveTheme } from "../preferences";

  let { lines, stations, vehicles = [], selectedStationId = "", center = [-73.94,40.72], zoom = 10.3, compact = false, bounds, class: className = "subway-map__canvas", onSelect, onVehicleSelect, stationLabelOverlap = false }: {
    lines: MapFeature[]; stations: Station[]; vehicles?: MapVehicle[]; selectedStationId?: string;
    center?: number[]; zoom?: number; compact?: boolean; bounds?: number[][]; class?: string;
    onSelect?: (id: string) => void; onVehicleSelect?: (vehicle: VehicleSelection) => void; stationLabelOverlap?: boolean;
  } = $props();
  let mapStations = $derived(stations.map((station) => {
    if (!station.services.length) return station;
    return { ...station, routeIds: stationActiveRouteIds(station) };
  }));
  let container: HTMLDivElement;
  let error = $state("");
  let map = $state<Awaited<ReturnType<typeof createSubwayMap>> | null>(null);
  let preferences=$state(readPreferences());
  let theme=$derived(effectiveTheme(preferences.theme));
  $effect(()=>watchPreferences((value)=>preferences=value));
  $effect(()=>{if(map)updateSubwayMapTheme(map,theme)});

  $effect(() => {
    if (map) updateSubwayMapData(map, { lines, stations: mapStations, vehicles }).catch(() => error = "Map unavailable");
  });
  $effect(() => { selectSubwayMapStation(map, selectedStationId); });

  onMount(() => {
    let cancelled = false;
    createSubwayMap({
      container, bounds, center, zoom, controls: !compact, lineCasing: false, theme,
      lines, stations: mapStations, vehicles, selectedStationId, showStationLabels: true, stationLabelOverlap,
      onStationSelect: (id: string) => onSelect?.(id),
      onVehicleSelect: (vehicle: VehicleSelection) => {
        if (onVehicleSelect) onVehicleSelect(vehicle);
        else goto(`/trip/${encodeURIComponent(vehicle.routeId)}/${encodeURIComponent(vehicle.id)}`);
      },
    }).then((instance) => {
      if (cancelled) instance.remove();
      else map = instance;
    }).catch(() => { if (!cancelled) error = "Map unavailable"; });
    return () => { cancelled = true; map?.remove(); map = null; };
  });
</script>

<div class={className} bind:this={container}></div>
{#if error}<span class="subway-map__status">{error}</span>{/if}
