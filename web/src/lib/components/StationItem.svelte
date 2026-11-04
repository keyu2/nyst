<script lang="ts">
  import type { Station } from "../api";
  import Icon from "./Icon.svelte";
  import RouteBullet from "./RouteBullet.svelte";
  import { stationActiveRouteIds,stationRouteIds } from "../stationDiscovery";
  let {station,distance,preferredStopId,preferredRoute}:{station:Station;distance?:number;preferredStopId?:string;preferredRoute?:string}=$props();
  let routes=$derived(stationRouteIds(station));
  let active=$derived(new Set(stationActiveRouteIds(station)));
  let service=$derived(station.services.find((item)=>item.id===preferredStopId)||station.services[0]);
  let route=$derived(preferredRoute||(service?stationRouteIds({...station,services:[service]})[0]:"")||service?.routes[0]||"");
</script>
{#if service}<a class="new-station-result" href={`/stop/${service.id}?route=${encodeURIComponent(route)}`}><span><strong>{station.name}</strong>{#if Number.isFinite(distance)}<small>{distance?.toFixed(2)} mi away</small>{/if}</span><span>{#each routes as item}<RouteBullet route={item} size={25} class={active.has(item)?"":"station-route-bullet--inactive"} alt={active.has(item)?`${item} train`:`${item} train, no service right now`} title={active.has(item)?undefined:"No service right now"}/>{/each}<Icon name="ChevronRight"/></span></a>{/if}
