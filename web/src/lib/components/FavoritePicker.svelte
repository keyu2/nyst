<script lang="ts">
  import RouteBullet from "./RouteBullet.svelte";
  import {ROUTE_GROUPS} from "../homeConfig";
  let {available,favorites,onToggle,onClose}:{available:Set<string>;favorites:string[];onToggle:(route:string)=>void;onClose:()=>void}=$props();
</script>
<div class="new-picker-backdrop" role="presentation" onmousedown={(event)=>{if(event.target===event.currentTarget)onClose()}}>
  <div class="new-picker" role="dialog" aria-modal="true" aria-labelledby="favorite-picker-title" tabindex="-1"><header><div><h2 id="favorite-picker-title">Favorite lines</h2><p>Select the lines you want to see first.</p></div><button type="button" aria-label="Close favorite line picker" onclick={onClose}>Close</button></header><div class="new-picker-grid">{#each ROUTE_GROUPS as group}{@const routes=group.filter((id)=>available.has(id))}{#if routes.length}<div class="new-picker-group">{#each routes as route}{@const selected=favorites.includes(route)}<button type="button" class:selected aria-label={`${selected?"Remove":"Add"} ${route} train from favorites`} aria-pressed={selected} onclick={()=>onToggle(route)}><RouteBullet {route} size={42}/><span class={selected?"remove":"add"} aria-hidden="true"></span></button>{/each}</div>{/if}{/each}</div></div>
</div>
