<script lang="ts">
  import Icon from "../components/Icon.svelte";
  import RoutePage from "./RoutePage.svelte";
  import FavoritePicker from "../components/FavoritePicker.svelte";
  import {ROUTE_GROUPS,ROUTE_STATUS} from "../homeConfig";
  import AppNavigation from "../components/AppNavigation.svelte";
  import LineList from "../components/LineList.svelte";
  import SearchResults from "../components/SearchResults.svelte";
  import { getBootstrap, type Station } from "../api";
  import { searchDiscovery } from "../stationDiscovery";
  import { nearestStations } from "../stationDiscovery";
  import { readPreferences, watchPreferences } from "../preferences";
  import { watchLiveData } from "../live";
  import StationItem from "../components/StationItem.svelte";
  import RouteBullet from "../components/RouteBullet.svelte";
  import { favoriteLines, favoriteStations, removeFavoriteStation, toggleFavoriteLine, watchFavorites } from "../favorites";

  let {mode="subway",focusedRoute=""}:{mode?:string;focusedRoute?:string}=$props();
  const fallbackRoutes = ["1", "2", "3", "4", "5", "6", "7", "A", "C", "E", "B", "D", "F", "M", "G", "J", "Z", "L", "N", "Q", "R", "W", "GS", "FS", "H", "SI"];
  let pickerOpen=$state(false);
  let query = $state("");
  let routeIds = $state(fallbackRoutes);
  let stations = $state.raw<Station[]>([]);
  let message = $state("");
  let routeStatuses = $state<Record<string,string>>({});
  let results = $derived(searchDiscovery(stations, routeIds, query));
  let preferences = $state(readPreferences());
  let lineFavorites = $state(favoriteLines());
  let stationFavorites = $state(favoriteStations());
  let locationValue = $state<{latitude:number;longitude:number}|null>(null);
  let locationMessage = $state("");
  let nearby = $derived(nearestStations(stations,locationValue,8));
  let activeFocusedRoute = $derived(routeIds.includes(focusedRoute)?focusedRoute:routeIds.includes("1")?"1":routeIds[0]||"");
  function locate(){if(!navigator.geolocation){locationMessage="Location is not available in this browser.";return}locationMessage="Finding nearby stations…";navigator.geolocation.getCurrentPosition(({coords})=>{locationValue={latitude:coords.latitude,longitude:coords.longitude};locationMessage=""},()=>locationMessage="Allow location access in your browser settings, then try again.",{enableHighAccuracy:true,timeout:10000,maximumAge:30000})}
  function toggleLine(route:string){toggleFavoriteLine(route);lineFavorites=favoriteLines()}
  function removeStation(id:string){removeFavoriteStation(id);stationFavorites=favoriteStations()}

  $effect(() => {
    const controller = new AbortController();
    function refresh(){return getBootstrap(controller.signal).then((data) => {
      routeIds = data.routes;
      stations = data.stations;
      routeStatuses = data.routeStatuses || {};
      message = "";
    }).catch((error) => {
      if (error.name !== "AbortError") message = "Live station data is temporarily unavailable.";
    });}
    void refresh();
    const stopLive=watchLiveData(refresh);
    return () => {controller.abort();stopLive()};
  });
  $effect(() => watchPreferences((value) => preferences=value));
  $effect(() => watchFavorites(() => {lineFavorites=favoriteLines();stationFavorites=favoriteStations()}));
</script>

<main class="new-app-shell">
  <div class="new-status-page">
    <AppNavigation active={mode} mapEnabled={preferences.mapEnabled} />
    {#if mode === "nearby"}
      <section class="new-nearby"><div class="new-nearby-intro"><h2>Stations around you</h2><p>Use your current location to find the closest subway entrances.</p></div><button class="new-location-button" type="button" onclick={locate}>{locationValue?"Refresh nearby stations":"Find stations near me"}</button>{#if locationMessage}<p role="status">{locationMessage}</p>{/if}{#if nearby.length}<div class="new-nearby-results">{#each nearby as station}<StationItem {station} distance={station.distance}/>{/each}</div>{/if}</section>
    {:else}
    {#if mode!=="subway"||preferences.lineView!=="focus"}<div class="new-search">
      <Icon name="Search"/>
      <input type="search" bind:value={query} placeholder="Search" aria-label="Search" />
    </div>
    {#if query.trim()}<SearchResults routes={results.routes} stations={results.stations} />{/if}{/if}
    {#if message}<p class="new-data-message" role="status">{message}</p>{/if}
    {#if mode === "subway" && preferences.lineView === "focus"}
      <section class="focused-line-browser focused-line-browser--route" aria-label={`${activeFocusedRoute} line stations`}><nav class="route-strip" aria-label="Choose a subway line">{#each fallbackRoutes.filter((id)=>routeIds.includes(id)) as route}<a class:active={route===activeFocusedRoute} href={`/subway?line=${encodeURIComponent(route)}`} aria-current={route===activeFocusedRoute?"page":undefined} aria-label={`${route} train${routeStatuses[route]?`, ${routeStatuses[route]} service status`:""}`}><RouteBullet {route} size={40}/>{#if routeStatuses[route]&&ROUTE_STATUS[routeStatuses[route] as keyof typeof ROUTE_STATUS]}<span class="route-strip__status" style:background={ROUTE_STATUS[routeStatuses[route] as keyof typeof ROUTE_STATUS].color}></span>{/if}</a>{/each}</nav><RoutePage routeId={activeFocusedRoute} embedded/></section>
    {:else if mode === "favorites"}
      {#if lineFavorites.length||stationFavorites.length}<div class="favorites-page">
      {#if lineFavorites.length}<section class="favorite-lines"><div class="favorites-section-heading"><h2>Lines</h2><button class="new-manage-favorites" type="button" onclick={()=>pickerOpen=true}>Manage</button></div><LineList routeIds={lineFavorites.filter((id)=>routeIds.includes(id))} groups={ROUTE_GROUPS.map((group)=>group.filter((id)=>lineFavorites.includes(id))).filter((group)=>group.length)} {routeStatuses} mobileLayout={preferences.mobileLineLayout}/></section>{:else if stationFavorites.length}<button class="new-manage-favorites" type="button" onclick={()=>pickerOpen=true}>Add favorite lines</button>{/if}
      {#if stationFavorites.length}<section class="favorite-stations"><h2>Stations</h2><div class="favorite-stations__list">{#each stationFavorites as favorite}{@const station=stations.find((item)=>item.services.some((service)=>(favorite.ids.length?favorite.ids:[favorite.id]).includes(service.id)))||{id:favorite.id,name:favorite.name,latitude:0,longitude:0,services:[{id:favorite.id,routes:favorite.routes.length?favorite.routes:[favorite.route].filter(Boolean),scheduledRoutes:[],realtimeRoutes:[]}]}}{#if station}<div class="favorite-station-row"><StationItem {station} preferredStopId={favorite.id} preferredRoute={favorite.route}/><button type="button" aria-label={`Remove ${favorite.name} from favorites`} onclick={()=>removeStation(favorite.id)}><Icon name="Star" fill="currentColor"/></button></div>{/if}{/each}</div></section>{/if}
      </div>{/if}
      {#if !lineFavorites.length&&!stationFavorites.length}<section class="new-empty"><Icon name="Star"/><h2>No favorites yet</h2><p>Choose favorite lines, or tap the star on any station page.</p><button type="button" onclick={()=>pickerOpen=true}>Choose favorite lines</button></section>{/if}
    {:else}<LineList {routeIds} {routeStatuses} mobileLayout={preferences.mobileLineLayout} />{/if}
    {/if}
  </div>
{#if pickerOpen}<FavoritePicker available={new Set(routeIds)} favorites={lineFavorites} onToggle={toggleLine} onClose={()=>pickerOpen=false}/>{/if}
</main>

