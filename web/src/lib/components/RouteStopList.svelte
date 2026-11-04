<script lang="ts">
  import type { RouteStop } from "../api";
  import RouteBullet from "./RouteBullet.svelte";
  import { hasTransferRouteChange,routeColors,routeFamily } from "../routes";
  let {stops,routeId,query="",noService=false,continuesAbove=false,continuesBelow=false,showEndpoint=true}:{stops:RouteStop[];routeId:string;query?:string;noService?:boolean;continuesAbove?:boolean;continuesBelow?:boolean;showEndpoint?:boolean}=$props();
  let searching=$derived(Boolean(query.trim()));
  let visible=$derived(searching?stops.filter((stop)=>stop.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())):stops);
  let color=$derived(`#${routeColors[routeFamily(routeId)]||"808183"}`);
  const inactiveColor="color-mix(in srgb, var(--muted) 48%, var(--background))";
</script>
{#snippet arrow(up=false)}<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">{#if up}<path d="m5 12 7-7 7 7M12 19V5"/>{:else}<path d="M12 5v14m7-7-7 7-7-7"/>{/if}</svg>{/snippet}
{#if !visible.length}<p class="route-stations-empty">No matching stations.</p>{:else}<div>
{#if showEndpoint&&!searching&&visible[0].borough}<div class="route-endpoint-marker"><span class="borough-direction-label">{visible[0].borough}{@render arrow()}</span></div>{/if}
{#each visible as stop,index}
  {@const inactive=noService||stop.hasService===false}
  {@const previousInactive=noService||visible[index-1]?.hasService===false}
  {@const nextInactive=noService||visible[index+1]?.hasService===false}
  {@const trackAbove=!noService&&(stop.trackServiceAbove??(!inactive&&!previousInactive))}
  {@const trackBelow=!noService&&(stop.trackServiceBelow??(!inactive&&!nextInactive))}
  {@const boundary=!searching&&visible[index-1]?.borough&&stop.borough&&visible[index-1].borough!==stop.borough}
  {@const bullets=stop.transferRoutes||[]}
  {@const showBullets=searching||index===0||index===visible.length-1||hasTransferRouteChange(bullets,visible[index-1]?.transferRoutes,visible[index+1]?.transferRoutes)}
  {#if boundary}<div class="borough-boundary" style:--strip-color={trackAbove?color:inactiveColor} aria-label={`Borough boundary between ${visible[index-1].borough} and ${stop.borough}`}><span class="borough-boundary-line" aria-hidden="true"></span><span class="borough-direction-label">{visible[index-1].borough}{@render arrow(true)}</span><span class="borough-direction-label">{stop.borough}{@render arrow()}</span></div>{/if}
  <a href={`/stop/${stop.id}?route=${encodeURIComponent(routeId)}`} class="strip-stop-row" aria-label={`${stop.name}${inactive?", no current train service":""}`}>
    <span class="strip-marker" style:--strip-color={inactive?inactiveColor:color} style:--strip-above-color={trackAbove?color:inactiveColor} style:--strip-below-color={trackBelow?color:inactiveColor}>
      {#if index>0||(continuesAbove&&!searching)}<span class="strip-line strip-line--above"></span>{/if}{#if index<visible.length-1||(continuesBelow&&!searching)}<span class="strip-line strip-line--below"></span>{/if}<span class="strip-dot" aria-hidden="true"></span>
    </span>
    <span class="strip-stop-content" class:strip-stop-content--inactive={inactive}><span class="strip-stop-name">{stop.name}{#if stop.accessible}<img class="route-stop-accessible-icon" src="/icons/accessible.svg" alt="Accessible station" width="24" height="24"/>{/if}</span>{#if showBullets&&bullets.length}<span class="strip-transfer-bullets">{#each bullets as route}<RouteBullet {route} size={22}/>{/each}</span>{/if}</span><span class="strip-stop-chevron" aria-hidden="true">›</span>
  </a>
{/each}</div>{/if}
