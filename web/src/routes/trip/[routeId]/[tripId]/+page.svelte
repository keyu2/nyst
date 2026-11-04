<script lang="ts">
  import Icon from "#lib/components/Icon.svelte";
  import { page } from "$app/state";
  import { goto } from "$app/navigation";
  import PageNavigation from "#lib/components/PageNavigation.svelte";
  import { getTrip,getRouteAlerts,type RouteAlert,type MapFeature,type Station,type TripResponse } from "#lib/api.ts";
  import RouteBullet from "#lib/components/RouteBullet.svelte";
  import { readPreferences, watchPreferences } from "#lib/preferences.ts";
  import { watchLiveData } from "#lib/live.ts";
  import { buildTripProgress,parentStopId,selectedStopStatus } from "#lib/tripProgress.ts";
  import {hasTransferRouteChange} from "#lib/routes.ts";
  import {serviceAlertStatus} from "#lib/alertClassification.ts";
  const timeFormatter=new Intl.DateTimeFormat("en-US",{hour:"2-digit",minute:"2-digit",hour12:false,timeZone:"America/New_York"});
  const alertAppearance={suspended:{color:"#991b1b",border:"#dc2626",background:"#fff0f0"},delay:{color:"#92400e",border:"#d97706",background:"#fff6e8"}};
  let alerts=$state<RouteAlert[]>([]);
  let tripAlerts=$derived(alerts.flatMap((alert)=>{const status=serviceAlertStatus(alert);const appearance=status==="planned"?null:alertAppearance[status];return appearance?[{alert,appearance,label:alert.alertType?.trim()||"Service Alert"}]:[]}));
  import SubwayMap from "#lib/components/SubwayMap.svelte";
  let routeId=$derived(page.params.routeId||"");let tripId=$derived(page.params.tripId||"");let trip=$state.raw<TripResponse|null>(null);let error=$state("");let now=$state(Math.floor(Date.now()/1000));let preferences=$state(readPreferences());let selectedStop=page.url.searchParams.get("from")||"";
  let progress=$derived(buildTripProgress(trip?.stopTimes||[],selectedStop));
  let mapStations=$derived((trip?.stopTimes||[]).filter((item)=>item.stop.longitude!=null&&item.stop.latitude!=null).map((item)=>({id:parentStopId(item.stop.id),name:item.stop.name,latitude:Number(item.stop.latitude),longitude:Number(item.stop.longitude),services:[]} as Station)));
  let mapLines=$derived(trip&&trip.mapCoordinates.length>1?[{type:"Feature",properties:{routeId,color:`#${trip.route.color||"808183"}`,offset:0},geometry:{type:"LineString",coordinates:trip.mapCoordinates}} as MapFeature]:[]);
  let mapBounds=$derived.by(()=>{if(!trip?.mapCoordinates.length)return undefined;const longitude=trip.mapCoordinates.map((item)=>item[0]);const latitude=trip.mapCoordinates.map((item)=>item[1]);return [[Math.min(...longitude),Math.min(...latitude)],[Math.max(...longitude),Math.max(...latitude)]];});
  $effect(() => {
    const controller = new AbortController();
    let loading = false;
    async function refresh() {
      if (loading) return;
      loading = true;
      try {
        const value = await getTrip(routeId, tripId, controller.signal);
        if (!controller.signal.aborted) { trip = value; now = Math.floor(Date.now()/1000); error = ""; }
      } catch (reason) {
        if (!controller.signal.aborted) error = "Trip data is temporarily unavailable.";
      } finally { loading = false; }
    }
    void refresh();
    getRouteAlerts(routeId,controller.signal).then((value)=>alerts=value).catch(()=>{});
    const stopLive = preferences.mapEnabled&&preferences.stationView!=="board"?watchLiveData(refresh):undefined;
    const clock = setInterval(() => now = Math.floor(Date.now()/1000), 1000);
    return () => { controller.abort(); stopLive?.(); clearInterval(clock); };
  });
  $effect(() => watchPreferences((value) => preferences=value));
</script>

<main class="page-shell trip-page" style:--trip-route-color={`#${trip?.route.color||"666"}`}>
  <PageNavigation fallbackHref={`/route/${routeId}`}/>
  {#if error}<p role="alert">{error}</p>{/if}
  {#if trip}
    {#if preferences.mapEnabled && preferences.stationView !== "board" && mapLines.length}<section class="trip-map-panel" aria-label={`Map of this ${routeId} train trip`}><SubwayMap lines={mapLines} stations={mapStations} selectedStationId={parentStopId(progress.nextStop?.id||"")} center={trip.mapCoordinates[Math.floor(trip.mapCoordinates.length/2)]} bounds={mapBounds} vehicles={trip.vehicle?[trip.vehicle]:[]} onSelect={(id)=>goto(`/stop/${encodeURIComponent(id)}?route=${encodeURIComponent(routeId)}`)} stationLabelOverlap compact class="trip-map-panel__canvas"/></section>{/if}
    <div class="trip-header"><a class="trip-route-link" aria-label={`View ${routeId} train route`} href={`/route/${encodeURIComponent(routeId)}`}><RouteBullet route={routeId} size={90} class="trip-route-bullet" /></a><div class="trip-heading-text"><div class="trip-terminal">{trip.origin?.name||`${routeId} Train`}</div><div class="trip-heading-separator">to</div><div class="trip-terminal">{trip.stopTimes.at(-1)?.stop.name||"Unknown destination"}</div></div>{#if tripAlerts.length}<div aria-label="Current service alerts" class="trip-alerts">{#each tripAlerts as {alert,appearance,label}}<a href={`/route/${encodeURIComponent(routeId)}/alerts`} title={alert.header||label} class="trip-alert-link" style:--trip-alert-border={appearance.border} style:--trip-alert-background={appearance.background} style:--trip-alert-color={appearance.color}>{label}</a>{/each}</div>{/if}</div>
    {#if progress.nextStop}<div class="trip-next-stop"><span>Next stop:</span> <strong>{progress.nextStop.name}</strong></div>{/if}
    {#if progress.previousStopCount>0}<input class="trip-previous-toggle" id="trip-previous-toggle" type="checkbox"/><label class="trip-previous-control" for="trip-previous-toggle"><span class="show-copy">Show previous stops ({progress.previousStopCount})</span><span class="hide-copy">Hide previous stops</span><svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg></label>{/if}
    {#if progress.intermediateStopCount>0}<input class="trip-between-toggle" id="trip-between-toggle" type="checkbox"/>{/if}
    <div class="trip-timeline">
      {#each trip.stopTimes as item,index}
        {@const timestamp=Number(item.arrival?.time||item.departure?.time||0)}
        {@const isNext=index===progress.nextStopIndex}
        {@const isSelected=index===progress.selectedStopIndex}
        {@const transfers=item.transferRoutes||[]}
        {@const showTransfers=index===0||index===trip.stopTimes.length-1||hasTransferRouteChange(transfers,trip.stopTimes[index-1]?.transferRoutes,trip.stopTimes[index+1]?.transferRoutes)}
        <div class:trip-between-stop={progress.intermediateStopCount>0&&index>progress.nextStopIndex&&index<progress.selectedStopIndex} class:trip-past-stop={progress.nextStopIndex!==-1&&index<progress.nextStopIndex}>
          <div class:trip-stop-row--past={progress.nextStopIndex!==-1&&index<progress.nextStopIndex} class:trip-stop-row--next={isNext} class="trip-stop-row" style:--trip-row-height={transfers.length&&showTransfers?"78px":"58px"}>
            <a class="trip-stop-link" href={`/stop/${parentStopId(item.stop.id)}?route=${routeId}`} aria-label={`View ${item.stop.name} station`}></a>
            <div class="trip-stop-time">{timestamp?timeFormatter.format(new Date(timestamp*1000)):""}</div>
            <div class="trip-stop-track">{#if index!==0}<div class="trip-track-line trip-track-line--above"></div>{/if}{#if index!==trip.stopTimes.length-1}<div class="trip-track-line trip-track-line--below"></div>{/if}{#if isNext}<span class="trip-train-icon"><svg width="1.5em" height="1.5em" viewBox="0 0 24 24" stroke-width="2" fill="none" xmlns="http://www.w3.org/2000/svg" color="currentColor" aria-hidden="true"><g stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"><path d="M14 16.01L14.01 15.9989"/><path d="M10 16.01L10.01 15.9989"/><path d="M22 12V15C22 17.7614 19.7614 20 17 20H7C4.23858 20 2 17.7614 2 15V12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12Z"/><path d="M18 12V15C18 17.7614 15.7614 20 13 20H11C8.23858 20 6 17.7614 6 15V12C6 9.23858 8.23858 7 11 7H13C15.7614 7 18 9.23858 18 12Z"/><path d="M10.5 20L8.5 22.5M13.5 20L15.5 22.5M16.5 20L18.5 22.5M7.5 20L5.5 22.5"/><path d="M11.7857 10H12.2143C13.2005 10 14 10.7995 14 11.7857C14 11.9041 13.9041 12 13.7857 12H10.2143C10.0959 12 10 11.9041 10 11.7857C10 10.7995 10.7995 10 11.7857 10Z"/></g></svg></span>{:else}<div class="trip-stop-dot"></div>{/if}</div>
            <div class:trip-stop-details--emphasized={isNext||isSelected} class="trip-stop-details"><div class:trip-selected-stop-name={isSelected}><span>{item.stop.name}</span>{#if isSelected}<span class="trip-stops-away">{selectedStopStatus(progress.stopsAway,timestamp,now,preferences.tripProgressUnit)}</span>{/if}</div>{#if showTransfers&&transfers.length}<div class="trip-transfer-routes" aria-label={`Transfer available to ${transfers.join(", ")}`}>{#each item.transferRoutes as transfer}<a class="trip-transfer-link" aria-label={`View ${transfer} train route`} href={`/route/${encodeURIComponent(transfer)}`}><RouteBullet route={transfer} size={20}/></a>{/each}</div>{/if}</div>
          </div>
        </div>
        {#if isNext&&progress.intermediateStopCount>0}<label class="trip-between-control" for="trip-between-toggle"><span class="trip-between-line"></span><span class="show-copy">Show intermediate stops</span><span class="hide-copy">Hide intermediate stops</span><Icon name="ChevronDown"/></label>{/if}
      {/each}
    </div>
    <footer class="trip-footer"><div>Trip ID: {trip.id}</div></footer>
  {/if}
</main>
