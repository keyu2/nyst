<script lang="ts">
  import PageNavigation from "../components/PageNavigation.svelte";
  import ServiceAlerts from "../components/ServiceAlerts.svelte";
  import RouteStopList from "../components/RouteStopList.svelte";
  import RouteBullet from "../components/RouteBullet.svelte";
  import {getRoute,type RouteBranch,type RouteResponse} from "../api";
  import {buildRouteTopology} from "../routeTopology";
  import {routeColors,routeFamily} from "../routes";
  import {watchLiveData} from "../live";
  let {routeId,embedded=false}:{routeId:string;embedded?:boolean}=$props();
  let route=$state.raw<RouteResponse|null>(null);
  let query=$state("");
  let error=$state("");
  let stops=$derived(route?.serviceMaps.find((item)=>item.configId==="alltimes")?.stops||route?.serviceMaps[0]?.stops||[]);
  let topology=$derived(buildRouteTopology(stops,route?.branches||[],routeId));
  $effect(()=>{
    const controller=new AbortController();
    const refresh=()=>getRoute(routeId,controller.signal).then((value)=>{route=value;error=""}).catch(()=>{if(!controller.signal.aborted)error="Route data is temporarily unavailable."});
    void refresh();
    const stopLive=watchLiveData(refresh);
    return()=>{controller.abort();stopLive()};
  });
</script>

{#snippet branch(item:RouteBranch|undefined,connectsAtStart=true,mergesBack=false)}
  {#if item?.stops.length}<section class="branch-offshoot" class:branch-offshoot--terminal-start={!connectsAtStart} style:--branch-color={`#${routeColors[routeFamily(routeId)]||"808183"}`}><h2 class="visually-hidden">{item.name} branch</h2>{#if mergesBack}<span class="branch-offshoot__merge" aria-hidden="true"></span>{/if}<RouteStopList noService={route?.serviceAvailable===false} stops={item.stops} {routeId} {query} showEndpoint={false}/></section>{/if}
{/snippet}

<svelte:element this={embedded?"div":"main"} class={embedded?"":"page-shell route-page"}>
  {#if !embedded}<PageNavigation fallbackHref="/subway"/>{/if}
  <div class={embedded?"focused-line-browser__identity":"route-page-header"}><RouteBullet route={routeId} size={64} class="route-page-bullet"/>{#if route?.serviceAvailable===false}<span class="route-no-service">No service right now</span>{/if}</div>
  {#if error}<p role="alert">{error}</p>{/if}
  {#if route}
    <ServiceAlerts alerts={route.alerts}/>
    <div>
      <label class="route-station-search"><span class="route-station-search__label">Search</span><input type="search" bind:value={query} placeholder="Enter a station name"/></label>
      {#if topology.kind==="standard"||topology.kind==="a"}<RouteStopList noService={route?.serviceAvailable===false} {stops} {routeId} {query} continuesBelow={topology.kind==="a"||Boolean(topology.branches?.length)}/>{/if}
      {#if topology.kind==="five"}<section aria-label="5 train branches"><RouteStopList noService={route?.serviceAvailable===false} stops={topology.trunkBefore} {routeId} {query} continuesBelow/>{@render branch(topology.branch,false,true)}<RouteStopList noService={route?.serviceAvailable===false} stops={topology.trunkAfter} {routeId} {query} continuesAbove showEndpoint={false}/></section>{/if}
      {#if topology.kind==="temporary"}<section aria-label={`${routeId} train temporary service pattern`}><RouteStopList noService={route?.serviceAvailable===false} stops={topology.trunkBefore} {routeId} {query} continuesBelow/>{#each topology.branchesBefore as item}{@render branch(item,false,true)}{/each}{#each topology.branchesAfter as item}{@render branch(item)}{/each}<RouteStopList noService={route?.serviceAvailable===false} stops={topology.trunkAfter} {routeId} {query} continuesAbove showEndpoint={false}/></section>{/if}
      {#if topology.kind==="a"}<section aria-label="A train branches">{@render branch(topology.lefferts)}<RouteStopList noService={route?.serviceAvailable===false} stops={topology.hasRockawayParkSplit?topology.farRockawayTrunk:[...topology.farRockawayTrunk,...topology.farRockawayTail]} {routeId} {query} continuesAbove continuesBelow={topology.hasRockawayParkSplit} showEndpoint={false}/>{#if topology.hasRockawayParkSplit}{@render branch(topology.rockawayPark)}<RouteStopList noService={route?.serviceAvailable===false} stops={topology.farRockawayTail} {routeId} {query} continuesAbove showEndpoint={false}/>{/if}</section>{/if}
      {#if topology.kind==="standard"&&topology.branches.length}<section aria-label={`${routeId} train branches`}>{#each topology.branches as item}{@render branch(item)}{/each}</section>{/if}
    </div>
  {/if}
</svelte:element>
