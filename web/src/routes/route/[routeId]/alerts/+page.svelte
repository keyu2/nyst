<script lang="ts">
  import { page } from "$app/state";
  import PageNavigation from "#lib/components/PageNavigation.svelte";
  import { getRouteAlerts, type RouteAlert } from "#lib/api.ts";
  import RouteBullet from "#lib/components/RouteBullet.svelte";
  import ServiceAlerts from "#lib/components/ServiceAlerts.svelte";
  let routeId=$derived(page.params.routeId||"");
  let alerts = $state<RouteAlert[]>([]);
  let error = $state("");
  let loaded = $state(false);
  $effect(() => {
    const controller = new AbortController();
    getRouteAlerts(routeId,controller.signal).then((value)=>{alerts=value;loaded=true;}).catch(()=>{if(!controller.signal.aborted)error="Live service alerts are temporarily unavailable.";});
    return ()=>controller.abort();
  });
</script>
<main class="page-shell alerts-page"><PageNavigation fallbackHref={`/route/${encodeURIComponent(routeId)}`} home={false}/><div class="alerts-page-header"><RouteBullet route={routeId} size={52}/><h1 class="alerts-page-title">Service Alerts</h1></div>{#if error}<p role="alert">{error}</p>{:else if loaded && !alerts.length}<p class="alerts-page-empty">No current service alerts.</p>{/if}<ServiceAlerts {alerts} mode="grouped"/></main>
