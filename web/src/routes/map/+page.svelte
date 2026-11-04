<script lang="ts">
  import AppNavigation from "#lib/components/AppNavigation.svelte";
  import SubwayMap from "#lib/components/SubwayMap.svelte";
  import RouteBullet from "#lib/components/RouteBullet.svelte";
  import { getMap, type MapResponse, type Station } from "#lib/api.ts";
  import { readPreferences, watchPreferences } from "#lib/preferences.ts";
  import { watchLiveData } from "#lib/live.ts";
  import { stationActiveRouteIds, stationRouteIds } from "#lib/stationDiscovery.ts";
  let preferences=$state(readPreferences());
  let data=$state.raw<MapResponse|null>(null);
  let selectedId=$state("");
  let selected=$derived(data?.stations.find((station)=>station.id===selectedId)||null);
  let error=$state("");
  $effect(() => {
    if (!preferences.mapEnabled) return;
    const controller = new AbortController();
    let loading = false;
    async function refresh() {
      if (loading) return;
      loading = true;
      try { const value = await getMap(controller.signal); if (!controller.signal.aborted) {data = value;error="";} }
      catch { if(!controller.signal.aborted)error="The map could not be loaded."; }
      finally { loading = false; }
    }
    void refresh();
    const stopLive = watchLiveData(refresh);
    return () => { controller.abort(); stopLive(); };
  });
  function stationHref(station:Station,route:string){const service=station.services.find((item)=>item.routes.includes(route))||station.services[0];return service?`/stop/${service.id}?route=${route}`:"/subway";}
  $effect(() => watchPreferences((value) => preferences=value));
</script>

<main class="new-app-shell map-app-shell"><div class="map-page" class:map-page--disabled={!preferences.mapEnabled}><AppNavigation active="map" mapEnabled={preferences.mapEnabled}/>{#if !preferences.mapEnabled}<section><h1>Map is off</h1><p>Turn it on in Settings when you want the interactive subway map.</p><a href="/settings">Open Settings</a></section>{:else if data}<section class="subway-map" aria-label="New York City subway map"><SubwayMap lines={data.lines} stations={data.stations} vehicles={data.vehicles||[]} selectedStationId={selectedId} onSelect={(id)=>selectedId=id}/>{#if selected}<aside class="subway-map-sheet" aria-label={`${selected.name} station`}><button class="subway-map-sheet__close" type="button" aria-label="Close station details" onclick={()=>selectedId=""}><svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m18 6-12 12M6 6l12 12"/></svg></button><p>{stationActiveRouteIds(selected).length?"Live service":"Scheduled service"}</p><h1>{selected.name}</h1><div class="subway-map-sheet__routes" aria-label="Services at this station">{#each stationActiveRouteIds(selected).length?stationActiveRouteIds(selected):stationRouteIds(selected) as route}<a href={stationHref(selected,route)}><RouteBullet {route} size={42}/></a>{/each}</div><a class="subway-map-sheet__arrivals" href={stationHref(selected,stationActiveRouteIds(selected)[0]||stationRouteIds(selected)[0]||"")}>View live arrivals</a></aside>{/if}</section>{:else}<section class="subway-map" aria-label="New York City subway map"><p class="subway-map__status">{error||"Loading map…"}</p></section>{/if}</div></main>
