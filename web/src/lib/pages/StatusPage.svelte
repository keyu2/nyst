<script lang="ts">
  import FavoritePicker from "../components/FavoritePicker.svelte";
  import AppNavigation from "../components/AppNavigation.svelte";
  import RouteBullet from "../components/RouteBullet.svelte";
  import RichAlertText from "../components/RichAlertText.svelte";
  import {getBootstrap,type RouteAlert} from "../api";
  import {readPreferences,watchPreferences} from "../preferences";
  import {watchLiveData} from "../live";
  import {ROUTE_GROUPS,ROUTE_FAMILIES,MOBILE_LINE_SECTIONS} from "../homeConfig";
  import {favoriteLines,toggleFavoriteLine,watchFavorites} from "../favorites";
  let {section="subway"}:{section?:string}=$props();
  type StatusAlert=RouteAlert & {routes:string[]};
  type Entry={routes:string[];alerts:StatusAlert[];color:string;familyIndex:number;label:string};
  let routeAlerts=$state<Record<string,RouteAlert[]>>({});
  let available=$state(new Set<string>());
  let selected=$state<Entry|null>(null);
  let preferences=$state(readPreferences());
  let message=$state("");
  let pickerOpen=$state(false);
  let favorites=$state(favoriteLines());
  let entries=$derived.by(()=>{
    return ROUTE_GROUPS.map((group,index)=>{
      const routes=group.filter((route)=>available.has(route)&&(section!=="favorites"||favorites.includes(route)));
      const unique=new Map<string,StatusAlert>();
      for(const route of routes)for(const alert of routeAlerts[route]||[]){
        const key=alert.id||`${alert.header}:${alert.description}`;
        if(!unique.has(key))unique.set(key,{...alert,id:key,routes:[]});
        unique.get(key)!.routes.push(route);
      }
      const alerts=[...unique.values()];
      return {routes,alerts,color:ROUTE_FAMILIES[index].color,familyIndex:index,label:alerts.length>1?`${alerts.length} alerts`:alerts[0]?.alertType||(alerts.length?"1 alert":"No alerts")};
    }).filter((entry)=>entry.routes.length);
  });
  $effect(()=>{
    const controller=new AbortController();
    const refresh=()=>getBootstrap(controller.signal).then((data)=>{routeAlerts=data.routeAlerts||{};available=new Set(data.routes);message=""}).catch(()=>{if(!controller.signal.aborted)message="Live service alerts are temporarily unavailable."});
    void refresh();
    const stopLive=watchLiveData(refresh);
    return()=>{controller.abort();stopLive()};
  });
  $effect(() => watchPreferences((value) => preferences=value));
  $effect(() => watchFavorites(() => favorites=favoriteLines()));
</script>

<main class="new-app-shell"><div class="new-status-page">
{#if selected}
  <section class="status-alert-detail">
    <button class="new-manage-favorites" onclick={()=>selected=null}>Back</button>
    <h2>Alerts for {#each selected.routes as route}<RouteBullet {route} size={32}/>{/each}</h2>
    {#each selected.alerts as alert}<article><h3>{#each alert.routes as route}<RouteBullet {route} size={28}/>{/each}{alert.alertType||"Service alert"}</h3><p class="status-alert-summary"><RichAlertText text={alert.header||alert.description}/></p>{#if alert.timeLabel||(alert.description&&alert.description!==alert.header)}<details><summary>Show more details</summary>{#if alert.timeLabel}<p class="status-alert-time">{alert.timeLabel}</p>{/if}{#if alert.description!==alert.header}<p class="status-alert-description"><RichAlertText text={alert.description}/></p>{/if}</details>{/if}</article>{/each}
  </section>
{:else}
  <AppNavigation active="status" mapEnabled={preferences.mapEnabled}/>
  {#if section==="favorites"}<button class="new-manage-favorites" type="button" onclick={()=>pickerOpen=true}>Manage favorite lines</button>{/if}
  {#if message}<p class="new-data-message">{message}</p>{/if}
  <section class="status-alert-list status-alert-list--desktop" class:status-alert-list--mobile-selected={preferences.mobileLineLayout==="rows"} aria-label="Service alerts">
    {#each entries as entry}<button class="status-alert-row" disabled={!entry.alerts.length} style:--route-family-color={entry.color} onclick={()=>selected=entry}><span class="status-alert-bullets">{#each entry.routes as route}<RouteBullet {route} size={40}/>{/each}</span><span class="status-alert-label">{entry.label}</span></button>{/each}
    {#if section==="favorites"&&!entries.length}<p>No favorite lines yet.</p>{/if}
  </section>
  <section class="status-alert-list status-alert-list--mobile-bands" class:status-alert-list--mobile-selected={preferences.mobileLineLayout==="grouped"} aria-label="Service alerts">
    {#each MOBILE_LINE_SECTIONS as group}
      {@const members=group.families.map((index)=>entries.find((entry)=>entry.familyIndex===index)).filter((entry):entry is Entry=>Boolean(entry))}
      {#if members.length}<div class={`new-line-mobile-section new-line-mobile-section--${members.length} new-line-mobile-section--${group.layout} status-alert-mobile-section`}>{#each members as entry}<button class="new-line-mobile-family status-alert-mobile-family" disabled={!entry.alerts.length} style:--route-family-color={entry.color} onclick={()=>selected=entry}><span class="new-line-mobile-bullets">{#each entry.routes as route}<RouteBullet {route} size={46}/>{/each}</span><span class="status-alert-mobile-label">{entry.label}</span></button>{/each}</div>{/if}
    {/each}
    {#if section==="favorites"&&!entries.length}<p>No favorite lines yet.</p>{/if}
  </section>
{/if}
</div>{#if pickerOpen}<FavoritePicker {available} {favorites} onToggle={toggleFavoriteLine} onClose={()=>pickerOpen=false}/>{/if}</main>
