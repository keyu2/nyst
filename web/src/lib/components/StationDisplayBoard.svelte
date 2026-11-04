<script lang="ts">
  import Icon from "./Icon.svelte";
  import { goto } from "$app/navigation";
  import RichAlertText from "./RichAlertText.svelte";
  import {serviceAlertLabel,serviceAlertTone} from "../alertClassification";
  import {createBoardRouteGroups} from "../boardRouteGroups";
  import {formatArrivalCountdown} from "../time";
  import RouteBullet from "./RouteBullet.svelte";
  import type { Arrival, RouteAlert, StationService } from "../api";
  import { routeFamily } from "../stationDiscovery";
  import { directionGroupLabel } from "../directionLabels";

  let { stopId, stopName, stopBorough, northbound, southbound, services, alertsByRoute, now, showSeconds, invertArrivingTrains }: { stopId:string; stopName:string; stopBorough?:string; northbound:Arrival[]; southbound:Arrival[]; services:StationService[]; alertsByRoute:Record<string,RouteAlert[]>; now:number; showSeconds:boolean; invertArrivingTrains:boolean } = $props();
  let fullscreen = $state(false);
  let savedSelection=$state("");
  let board=$state<HTMLElement>();
  let content=$state<HTMLDivElement>();
  const timeFormatter=new Intl.DateTimeFormat("en-US",{timeZone:"America/New_York",hour:"2-digit",minute:"2-digit",second:"2-digit"});
  const dateFormatter=new Intl.DateTimeFormat("en-US",{timeZone:"America/New_York",weekday:"short",month:"short",day:"numeric"});
  let currentDate=$derived(new Date(now*1000));
  let timeParts=$derived(timeFormatter.formatToParts(currentDate));
  let alertPage = $state(0);
  let routeIds=$derived([...new Set([...northbound,...southbound].map((train)=>train.trip.route.id))].sort((a,b)=>a.localeCompare(b,undefined,{numeric:true})));
  let groups=$derived(createBoardRouteGroups(routeIds,services));
  let selectedIds=$derived.by(()=>{
    const ids=groups.map((group)=>group.id);
    const valid=savedSelection.split("|").filter((id)=>ids.includes(id));
    return valid.length?valid:ids;
  });
  let selectedRoutes = $derived(new Set(groups.filter((group) => selectedIds.includes(group.id)).flatMap((group) => group.members.map(routeFamily))));
  let filteredNorthbound = $derived(northbound.filter((train) => selectedRoutes.has(routeFamily(train.trip.route.id))));
  let filteredSouthbound = $derived(southbound.filter((train) => selectedRoutes.has(routeFamily(train.trip.route.id))));
  let alerts = $derived.by(() => {
    const value = new Map<string,RouteAlert>();
    for(const group of groups){
      if(!selectedIds.includes(group.id))continue;
      for(const route of new Set([...group.members,...group.members.map(routeFamily)])){
        for(const alert of alertsByRoute[route]||[]){
          const id=alert.id||`${alert.alertType}:${alert.header}:${alert.description}`;
          if(!value.has(id))value.set(id,alert);
        }
      }
    }
    return [...value.values()];
  });
  let alertPages = $derived(Array.from({ length:Math.ceil(alerts.length/2) }, (_, index) => alerts.slice(index*2,index*2+2)));
  let alertPageCount = $derived(alertPages.length);
  let activeAlerts = $derived(alertPages[alertPage] || []);

  function arrivalTime(train:Arrival) { return Number(train.arrival?.time || train.departure?.time || 0); }
  function current(trains:Arrival[]) { return trains.filter((train) => arrivalTime(train) >= now-30).sort((a,b) => arrivalTime(a)-arrivalTime(b)).slice(0,4); }
  function toggleGroup(id:string) {
    let selected:string[];
    if(selectedIds.length===groups.length)selected=[id];
    else if(selectedIds.includes(id))selected=selectedIds.length===1?selectedIds:selectedIds.filter((value)=>value!==id);
    else selected=[...selectedIds,id];
    const key=`subway-display-board-routes:${stopId}`;
    if(selected.length===groups.length)sessionStorage.removeItem(key);
    else sessionStorage.setItem(key,selected.join("|"));
    savedSelection=sessionStorage.getItem(key)||"";
  }
  async function toggleFullscreen() {
    try{if(document.fullscreenElement)await document.exitFullscreen();else await board?.requestFullscreen?.();}catch{return false}
    return true;
  }
  function boardLabel(trains:Arrival[],direction:string){
    const parts=new Set<string>();
    for(const train of current(trains))for(const part of directionGroupLabel(stopBorough,train.destination?.borough,direction).split("/").map((value)=>value.trim()).filter(Boolean))parts.add(part);
    return [...parts].join(" / ")||(direction==="northbound"?"Northbound":"Southbound");
  }
  let carouselKey=$derived(alerts.map((alert,index)=>alert.id||`${alert.header}-${index}`).join("|"));
  let previousCarouselKey="";
  $effect(()=>{if(carouselKey!==previousCarouselKey){previousCarouselKey=carouselKey;alertPage=0}});
  $effect(()=>{
    const sync=()=>savedSelection=sessionStorage.getItem(`subway-display-board-routes:${stopId}`)||"";
    sync();
    window.addEventListener("storage",sync);
    return()=>window.removeEventListener("storage",sync);
  });
  $effect(()=>{
    selectedIds.join("|");alerts.length;
    const element=board,fitContent=content;
    if(!element||!fitContent)return;
    const reset=()=>{fitContent.style.width="";fitContent.style.transform=""};
    if(!fullscreen){reset();return;}
    let frame=0;
    const fit=()=>{cancelAnimationFrame(frame);frame=requestAnimationFrame(()=>{
      reset();
      let scale=Math.min(1,element.clientHeight/Math.max(fitContent.scrollHeight,1));
      fitContent.style.width=`${100/scale}%`;
      scale=Math.min(1,element.clientHeight/Math.max(fitContent.scrollHeight,1));
      fitContent.style.width=`${100/scale}%`;
      fitContent.style.transform=`scale(${scale})`;
    })};
    const observer=new ResizeObserver(fit);observer.observe(fitContent);
    window.addEventListener("resize",fit);fit();
    return()=>{cancelAnimationFrame(frame);observer.disconnect();window.removeEventListener("resize",fit);reset()};
  });
  $effect(() => {
    carouselKey;
    const count=alertPageCount;
    if (count < 2) return;
    const page=alertPage;
    const timer=setTimeout(() => alertPage=(page+1)%count,15000);
    return () => clearTimeout(timer);
  });
  $effect(() => {
    const sync=()=>fullscreen=document.fullscreenElement===board;
    document.addEventListener("fullscreenchange",sync);
    return()=>document.removeEventListener("fullscreenchange",sync);
  });
</script>

{#snippet direction(directionId:string, trains:Arrival[])}
  {@const label=boardLabel(trains,directionId)}
  {@const visible=current(trains)}
  {#if visible.length}
    <section class="display-board-direction" aria-label={`${label} arrivals`}>
      <h2>{label}</h2>
      <div class="display-board-trains">
        {#each visible as train}
          {@const seconds=Math.max(0,arrivalTime(train)-now)}
          <a class:display-board-train--arriving={seconds<60 && invertArrivingTrains} class="display-board-train" href={`/trip/${train.trip.route.id}/${encodeURIComponent(train.trip.id)}?from=${encodeURIComponent(stopId)}`}>
            <RouteBullet route={train.trip.route.id} size={56}/>
            <span class="display-board-destination"><strong>{directionGroupLabel(stopBorough,train.destination?.borough,directionId)}</strong><span>{train.destination.id.startsWith(stopId) ? "Terminating" : train.destination.name || train.headsign}</span>{#if train.delayMinutes}<small>Est. delayed {train.delayMinutes} min</small>{/if}</span>
            <strong class="display-board-time" aria-label={formatArrivalCountdown(seconds,showSeconds)}><span class="display-board-countdown" aria-hidden="true"><span class="display-board-minute-stack"><span class="display-board-minutes">{Math.floor(seconds/60)}</span><small>MIN</small></span>{#if showSeconds}<sup>{String(seconds%60).padStart(2,"0")}</sup>{/if}</span></strong>
          </a>
        {/each}
      </div>
    </section>
  {/if}
{/snippet}

<div class="station-trains station-trains--board">
  <section class="station-display-board" bind:this={board}>
    <button class="display-board-fullscreen-exit" type="button" onclick={toggleFullscreen} aria-label="Exit full screen"></button>
    <div class="display-board-fit" bind:this={content}>
      <div class="display-board-masthead"><span class="display-board-brand">NYST</span><time datetime={currentDate.toISOString()} title={dateFormatter.format(currentDate)}>{#each timeParts as part}<span class:display-board-clock-seconds={part.type==="second"} class:display-board-clock-period={part.type==="dayPeriod"}>{part.value}</span>{/each}</time></div>
      <header class="display-board-header">
        <div class="display-board-identity">
          {#if groups.length>1}<div class="display-board-route-filters" aria-label="Lines shown">{#each groups as group}<button type="button" class:active={selectedIds.includes(group.id)} aria-label={`${group.members.join(" and ")} line filter`} aria-pressed={selectedIds.includes(group.id)} onclick={()=>toggleGroup(group.id)}>{#each group.members as route}<RouteBullet {route} size={30}/>{/each}</button>{/each}</div>{:else}<div class="display-board-routes">{#each routeIds as route}<RouteBullet {route} size={34}/>{/each}</div>{/if}
          <h1>{stopName}</h1>
        </div>
        <div class="display-board-controls"><button type="button" onclick={()=>{if(history.length>1)history.back();else goto("/subway")}}>Back</button><a href="/settings">Settings</a><button type="button" onclick={toggleFullscreen} aria-pressed={fullscreen}><Icon name={fullscreen?"Minimize":"Maximize"}/>{fullscreen?"Exit full screen":"Full screen"}</button></div>
      </header>
      <div class="display-board-directions">{@render direction("northbound",filteredNorthbound)}{@render direction("southbound",filteredSouthbound)}</div>
      {#if !current(filteredNorthbound).length && !current(filteredSouthbound).length}<p class="display-board-empty">No upcoming trains reported.</p>{/if}
      <section class="display-board-alerts" aria-labelledby="display-board-alerts-title">
        <h2 id="display-board-alerts-title">Happening now</h2>
        {#if !alerts.length}<p class="display-board-clear">No active alerts for the selected lines.</p>{:else}
          {#key carouselKey}<div class="display-board-alert-progress" style:--display-alert-count={alertPages.length} style:--display-alert-duration="15000ms" aria-hidden="true">{#each alertPages as _,index}<span class:complete={alertPages.length===1||index<alertPage} class:active={alertPages.length>1&&index===alertPage}><i></i></span>{/each}</div>{/key}
          <div class="display-board-alert-slide" aria-live="polite" aria-atomic="true">{#each activeAlerts as alert (alert.id||alert.header)}<details class={`display-board-alert display-board-alert--${serviceAlertTone(alert)}`} open><summary><span>{serviceAlertLabel(alert)}</span><strong><RichAlertText text={alert.header||"Service alert"}/></strong></summary>{#if alert.timeLabel||alert.description!==alert.header}<div class="display-board-alert__details">{#if alert.timeLabel}<b>{alert.timeLabel}</b>{/if}{#if alert.description!==alert.header}<p><RichAlertText text={alert.description}/></p>{/if}</div>{/if}</details>{/each}</div>
        {/if}
      </section>
    </div>
  </section>
</div>
