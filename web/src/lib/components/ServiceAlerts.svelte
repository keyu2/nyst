<script lang="ts">
  import Icon from "./Icon.svelte";
  import type { RouteAlert } from "../api";
  import { serviceAlertLabel, serviceAlertTone } from "../alertClassification";
  import RichAlertText from "./RichAlertText.svelte";
  let { alerts, mode = "route" }: { alerts: RouteAlert[]; mode?: "route" | "station" | "grouped" } = $props();
  let groups = $derived.by(() => {
    const result = new Map<string, {title:string;tone:string;alerts:RouteAlert[]}>();
    for (const alert of alerts) {
      const title = serviceAlertLabel(alert, "Service Alert");
      const key = title.trim().toLowerCase();
      if (result.has(key)) result.get(key)!.alerts.push(alert);
      else result.set(key, { title, tone: serviceAlertTone(alert), alerts: [alert] });
    }
    return [...result.values()];
  });
  const priority = {planned:1,warning:2,critical:3};
  let primary = $derived([...alerts].sort((a,b) => priority[serviceAlertTone(b)] - priority[serviceAlertTone(a)])[0]);
</script>

{#snippet content(alert:RouteAlert, kind="route-service-alert")}
  {#if alert.timeLabel}<div class={`${kind}__time`}>{alert.timeLabel}</div>{/if}
  {#if alert.header}<strong class={kind==="alert-card"?"alert-card__heading":undefined}><RichAlertText text={alert.header}/></strong>{/if}
  {#if alert.description && alert.description !== alert.header}<div class={`${kind}__description`}><RichAlertText text={alert.description}/></div>{/if}
{/snippet}

{#if alerts.length}
  {#if mode === "station" && primary}
    <details class={`station-service-alert station-service-alert--${serviceAlertTone(primary)}`}>
      <summary class="station-service-alert__summary"><Icon name="TriangleAlert"/><span>{serviceAlertLabel(primary)}</span>{#if alerts.length>1}<span class="station-service-alert__count">+{alerts.length-1}</span>{/if}</summary>
      <div class="station-service-alert__details">{#each alerts as alert}<article class="station-service-alert__item">{#if alert.alertType}<h3>{serviceAlertLabel(alert)}</h3>{/if}{@render content(alert,"station-service-alert")}</article>{/each}</div>
    </details>
  {:else if mode === "grouped"}
    {#each [{category:"happening",title:"Happening Now"},{category:"planned",title:"Planned Service Changes"}] as group}
      {@const items = alerts.filter((alert)=>alert.category===group.category)}
      {#if items.length}<section class="alert-group"><h2>{group.title}</h2><div class="alert-group__list">{#each items as alert}<details class={`alert-card alert-card--${serviceAlertTone(alert)}`}><summary class="alert-card__summary"><span class="alert-card__type">{serviceAlertLabel(alert,"Service Alert")}</span></summary><div class="alert-card__details">{@render content(alert,"alert-card")}</div></details>{/each}</div></section>{/if}
    {/each}
  {:else}
    <section class="route-service-alerts" aria-labelledby="route-service-alerts-title"><h2 id="route-service-alerts-title">Service Alerts</h2><div class="route-service-alerts__list">{#each groups as group}<details class={`route-service-alert route-service-alert--${group.tone}`}><summary><span class="route-service-alert__label"><span>{group.title}</span>{#if group.alerts.length>1}<span class="route-service-alert__count">+{group.alerts.length-1}</span>{/if}</span></summary><div class="route-service-alert__details">{#each group.alerts as alert}<article class="route-service-alert__item">{@render content(alert)}</article>{/each}</div></details>{/each}</div></section>
  {/if}
{/if}
