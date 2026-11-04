<script lang="ts">
  import type { Station } from "../api";
  import StationItem from "./StationItem.svelte";
  import RouteBullet from "./RouteBullet.svelte";
  let { routes, stations }: { routes: string[]; stations: Station[] } = $props();
</script>

<section class="new-search-results">
  {#each routes as route}
    <a href={`/route/${route}`}><strong>{route} train</strong><RouteBullet {route} size={28} /></a>
  {/each}
  {#each stations as station}
    <StationItem {station}/>
  {/each}
  {#if routes.length === 0 && stations.length === 0}<p>No stations found.</p>{/if}
</section>

