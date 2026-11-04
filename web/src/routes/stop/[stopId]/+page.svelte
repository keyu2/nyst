<script lang="ts">
  import Icon from "#lib/components/Icon.svelte";
  import { page } from "$app/state";
  import { goto } from "$app/navigation";
  import PageNavigation from "#lib/components/PageNavigation.svelte";
  import ServiceAlerts from "#lib/components/ServiceAlerts.svelte";
  import { getBootstrap, getStationMap, getStop, type Arrival, type RouteAlert, type StationMapResponse, type StopResponse } from "#lib/api.ts";
  import { readPreferences, watchPreferences } from "#lib/preferences.ts";
  import { watchLiveData } from "#lib/live.ts";
  import { routeFamily } from "#lib/stationDiscovery.ts";
  import { formatArrivalCountdown } from "#lib/time.ts";
  import RouteBullet from "#lib/components/RouteBullet.svelte";
  import StationDisplayBoard from "#lib/components/StationDisplayBoard.svelte";
  import SubwayMap from "#lib/components/SubwayMap.svelte";
  import { isFavoriteStation,toggleFavoriteStation,watchFavorites } from "#lib/favorites.ts";
  import {realtimeDirectionOptions,groupDirectionOptions,directionKey} from "#lib/stationDirections.ts";
  import { createStationRouteFamilies, createStationRouteGroups } from "#lib/boardRouteGroups.ts";

  let stopId=$derived(page.params.stopId||"");
  let stop = $state.raw<StopResponse | null>(null);
  let error = $state("");
  let now = $state(Math.floor(Date.now()/1000));
  const requestedRoute=page.url.searchParams.get("route")||"";
  let selectedRoute = $state(requestedRoute);
  let selectedDirection = $state("");
  let selectedTerminus = $state("");
  let branchMenu=$state<HTMLDetailsElement>();
  let visibleLimit = $state(6);
  let preferences = $state(readPreferences());
  let alertsByRoute = $state<Record<string,RouteAlert[]>>({});
  let mapData = $state.raw<StationMapResponse|null>(null);
  let favorite = $state(false);
  let transferGroups=$derived.by(()=>{
    const groups=new Map<string,{name:string;services:StopResponse["transfers"]}>();
    for(const transfer of stop?.transfers||[]){
      const group=groups.get(transfer.stop.name)||{name:transfer.stop.name,services:[]};
      if(!group.services.some((item)=>item.routes.join(",")===transfer.routes.join(",")))group.services.push(transfer);
      groups.set(transfer.stop.name,group);
    }
    return [...groups.values()];
  });
  let sameTransfers=$derived(transferGroups.filter((group)=>group.name.trim().toLocaleLowerCase()===stop?.name.trim().toLocaleLowerCase()).flatMap((group)=>group.services));
  let otherTransfers=$derived(transferGroups.filter((group)=>group.name.trim().toLocaleLowerCase()!==stop?.name.trim().toLocaleLowerCase()));
  let routeSelection = $derived(createStationRouteFamilies(stop?.stopTimes.map((train)=>train.trip.route.id)||[],stop?.services||[],selectedRoute));
  let routeIds = $derived(routeSelection.familyIds);
  let activeRoute = $derived(routeIds.includes(routeFamily(selectedRoute)) ? routeFamily(selectedRoute) : routeIds[0] || "");
  let routeGroups = $derived(createStationRouteGroups(routeIds));
  let mapRouteIds = $derived(routeGroups.find((members) => members.includes(activeRoute)) || []);
  let mapLines = $derived(mapData?.lines.filter((line) => mapRouteIds.includes(routeFamily(line.properties.routeId))) || []);
  let mapVehicles = $derived(mapData?.vehicles.filter((vehicle) => mapRouteIds.includes(routeFamily(vehicle.routeId))) || []);
  let routeTrains = $derived(stop?.stopTimes.filter((train) => routeFamily(train.trip.route.id) === activeRoute) || []);
  let northbound = $derived(routeTrains.filter((train) => train.stop.id.endsWith("N")));
  let southbound = $derived(routeTrains.filter((train) => train.stop.id.endsWith("S")));
  let directionOptions=$derived(realtimeDirectionOptions({activeRoute,northbound,southbound}));
  let directionGroups=$derived(groupDirectionOptions(directionOptions,stop?.borough));
  let activeGroup=$derived(directionGroups.find((group)=>group.id===selectedDirection)||directionGroups[0]);
  let activeTerminus=$derived(activeGroup?.options.find((option)=>directionKey(option)===selectedTerminus));
  let availableDirections=$derived(directionGroups.map((group)=>group.id));
  let activeDirection=$derived(activeGroup?.id||"");
  let activeTrains=$derived((activeTerminus?.trains||activeGroup?.options.flatMap((option)=>option.trains)||[]).filter((train)=>arrivalTime(train)>=now-30).sort((a,b)=>arrivalTime(a)-arrivalTime(b)));
  let activeAlerts = $derived(alertsByRoute[activeRoute]||[]);

  function arrivalTime(train: Arrival) { return Number(train.arrival?.time || train.departure?.time || Infinity); }
  function toggleFavorite(){if(stop)favorite=toggleFavoriteStation({id:stopId,ids:[...new Set([stopId,...stop.services.map((service)=>service.id)])],name:stop.name,route:activeRoute,routes:routeIds})}
  function load() {
    const controller = new AbortController();
    getStop(stopId, controller.signal).then((value) => { stop=value; error=""; now=value.generatedAt; }).catch((reason) => { if(reason.name!=="AbortError") error="Realtime station data is temporarily unavailable."; });
    if(preferences.mapEnabled)getStationMap(stopId,controller.signal).then((data)=>mapData=data).catch(()=>{});
    return controller;
  }
  $effect(() => {
    favorite=isFavoriteStation([stopId]);
    let controller = load();
    function refreshAlerts(){getBootstrap(controller.signal).then((data)=>alertsByRoute=data.routeAlerts||{}).catch(()=>{})}
    refreshAlerts();
    const stopLive = watchLiveData(() => { controller.abort(); controller=load(); refreshAlerts(); }, preferences.mapEnabled&&preferences.stationView!=="board"?15_000:20_000);
    const clock = window.setInterval(() => now=Math.floor(Date.now()/1000), 1_000);
    return () => { controller.abort(); stopLive(); clearInterval(clock); };
  });
  $effect(() => watchPreferences((value) => preferences=value));
  $effect(() => watchFavorites(() => favorite=isFavoriteStation([stopId])));
</script>

<main class="page-shell stop-page">
  {#if error}<p role="alert">{error}</p>{/if}
  {#if stop}
    {#if preferences.stationView === "board"}
      <StationDisplayBoard {stopId} stopName={stop.name} stopBorough={stop.borough} northbound={stop.stopTimes.filter((train)=>train.stop.id.endsWith("N"))} southbound={stop.stopTimes.filter((train)=>train.stop.id.endsWith("S"))} services={stop.services} {alertsByRoute} {now} showSeconds={preferences.showSeconds} invertArrivingTrains={preferences.boardArrivalInversion}/>
    {:else}
      <PageNavigation fallbackHref={requestedRoute?`/route/${encodeURIComponent(requestedRoute)}`:"/subway"}/>
      {#if preferences.mapEnabled && mapData}<section class="station-map-panel" aria-label={`Map of ${stop.name} station`}><SubwayMap lines={mapLines} stations={[mapData.station]} vehicles={mapVehicles} selectedStationId={mapData.station.id} center={[mapData.station.longitude,mapData.station.latitude]} zoom={12.8} compact class="station-map-panel__canvas" onVehicleSelect={(vehicle)=>goto(`/trip/${encodeURIComponent(vehicle.routeId)}/${encodeURIComponent(vehicle.id)}?from=${encodeURIComponent(stopId)}`)}/></section>{/if}
      <div class="stop-page-heading"><h1 class="stop-page-title">{stop.name}</h1>{#if stop.accessible}<img class="stop-accessible-icon" src="/icons/accessible.svg" alt="Accessible station" width="30" height="30" />{/if}<button class="favorite-station-button" class:selected={favorite} type="button" aria-label={`${favorite?"Remove":"Add"} ${stop.name} ${favorite?"from":"to"} favorites`} aria-pressed={favorite} onclick={toggleFavorite}><Icon name="Star" fill={favorite?"currentColor":"none"}/></button></div>
      {#if otherTransfers.length}<div class="transfer-section"><div class="transfer-label">Transfer is available to:</div><div class="transfer-groups">{#each otherTransfers as station}{@const target=station.services[0]}{@const routes=[...new Set(station.services.flatMap((service)=>service.routes))]}<a class="transfer-station-group" href={`/stop/${target.stop.id}?route=${encodeURIComponent(target.routes[0]||"")}`} aria-label={`${station.name}: ${routes.join(", ")} trains`}><span>{station.name}</span><span class="transfer-station-routes">{#each routes as route}<RouteBullet {route} size={24}/>{/each}</span></a>{/each}</div></div>{/if}
      <div class="station-trains station-trains--standard">
      <div class="station-route-tabs" role="tablist" aria-label="Subway lines at this station">
        {#each routeGroups as families}
          <div class="station-route-group" class:active={families.includes(activeRoute)} role="presentation">
        {#each families as route}
          <button class:active={route===activeRoute} type="button" onclick={() => { selectedRoute=route; selectedDirection=""; selectedTerminus=""; visibleLimit=6; }} role="tab" aria-selected={route===activeRoute} aria-label={`${(routeSelection.routeFamilies.get(route)||[route]).join(" and ")} Train`}>
            {#each routeSelection.routeFamilies.get(route)||[route] as member}<RouteBullet route={member} size={28}/>{/each}
          </button>
        {/each}
          </div>
        {/each}
        {#each sameTransfers as service}<a class="same-station-transfer-tab" href={`/stop/${service.stop.id}?route=${encodeURIComponent(service.routes[0]||"")}`} aria-label={`Transfer to ${service.routes.join(" and ")} train`}>{#each service.routes as route}<RouteBullet {route} size={28}/>{/each}<svg class="same-station-transfer-arrow" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M7 7h10v10M7 17 17 7"/></svg></a>{/each}
      </div>
      {#if activeAlerts.length}<div class="station-service-alerts"><ServiceAlerts alerts={activeAlerts} mode="station"/></div>{/if}
      <div class="station-direction-divider" aria-hidden="true"></div>
      {#if availableDirections.length === 0}
        <div class="station-no-service"><RouteBullet route={activeRoute} size={26} /><span>trains are not scheduled to stop at this station in either direction. Use alternative service.</span></div>
      {:else}
        <div class="direction-groups"><div class="direction-slider" role="tablist" aria-label="Train direction" style:--direction-count={availableDirections.length} style:--direction-index={Math.max(0,availableDirections.indexOf(activeDirection))}>
          {#each availableDirections as direction}
            <button role="tab" aria-selected={direction===activeDirection} type="button" class:active={direction===activeDirection} onclick={() => { selectedDirection=direction; selectedTerminus=""; visibleLimit=6; }}>{directionGroups.find((group)=>group.id===direction)?.label}</button>
          {/each}
        </div>
        {#if activeGroup&&activeGroup.options.length>1}<details class="direction-branch-picker" bind:this={branchMenu} aria-label={`${activeGroup.label} terminal filter`}><summary><span>{activeTerminus?.label||"All branches"}</span><svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg></summary><div class="direction-branch-options"><button class:active={!activeTerminus} type="button" aria-pressed={!activeTerminus} onclick={()=>{selectedTerminus="";visibleLimit=6;branchMenu?.removeAttribute("open")}}>All branches</button>{#each activeGroup.options as option}<button class:active={directionKey(option)===selectedTerminus} type="button" aria-pressed={directionKey(option)===selectedTerminus} onclick={()=>{selectedTerminus=directionKey(option);visibleLimit=6;branchMenu?.removeAttribute("open")}}>{option.label}</button>{/each}</div></details>{/if}
        </div><section class="train-direction">
          <div class="arrival-list">
            {#each activeTrains.slice(0,visibleLimit) as train}
              <a class="arrival-row" href={`/trip/${train.trip.route.id}/${encodeURIComponent(train.trip.id)}?from=${encodeURIComponent(stopId)}`}>
                <RouteBullet route={train.trip.route.id} size={38} class="arrival-bullet" />
                <div class="arrival-destination"><div>{train.destination.id.startsWith(stopId) ? "Terminating" : train.destination.name || "Unknown destination"}</div>{#if train.delayMinutes}<div class="arrival-delay">Est. Delayed: {train.delayMinutes} min</div>{/if}</div>
                <div class:arrival-time--pending={train.notDeparted} class="arrival-time"><span>{formatArrivalCountdown(Number.isFinite(arrivalTime(train))?Math.max(0,arrivalTime(train)-now):null,preferences.showSeconds)}</span>{#if train.notDeparted}<svg aria-label="Train has not departed" role="img" viewBox="0 0 24 24" width="20" height="20" class="arrival-clock-icon"><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2"/><path d="M12 7v5l3 2" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>{/if}</div>
              </a>
            {/each}
            {#if activeTrains.length>6}<button class="arrival-show-more" onclick={() => visibleLimit = visibleLimit>=Math.min(18,activeTrains.length) ? 6 : Math.min(18,visibleLimit+6)}>{visibleLimit>=Math.min(18,activeTrains.length)?"Show fewer trains":"Show more trains"}</button>{/if}
          </div>
        </section>
      {/if}
    </div>{/if}
  {/if}
</main>
