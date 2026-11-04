<script lang="ts">
  import {ROUTE_GROUPS,ROUTE_FAMILIES,MOBILE_LINE_SECTIONS as sections,ROUTE_STATUS} from "../homeConfig";
  import RouteBullet from "./RouteBullet.svelte";
  let { routeIds, routeStatuses = {}, mobileLayout = "rows", groups = ROUTE_GROUPS }: { routeIds: string[]; routeStatuses?: Record<string, string>; mobileLayout?: "grouped" | "rows";groups?:string[][] } = $props();
  const colors=ROUTE_FAMILIES.map((family)=>family.color);
  const available = $derived(new Set(routeIds));
  const mobileGroups = $derived(ROUTE_FAMILIES.map((family)=>groups.flat().filter((route)=>family.routes.includes(route))));
  const statusColors:Record<string,string>=Object.fromEntries(Object.entries(ROUTE_STATUS).map(([key,status])=>[key,status.color]));
</script>

{#snippet routeButtons(group: string[], size = 40)}
  {#each group as route}
    {#if available.has(route)}
      <a class="new-route-button" href={`/route/${route}`} aria-label={`${route} train${routeStatuses[route]&&ROUTE_STATUS[routeStatuses[route] as keyof typeof ROUTE_STATUS]?`, ${ROUTE_STATUS[routeStatuses[route] as keyof typeof ROUTE_STATUS].label}`:""}`}>
        <RouteBullet {route} {size} />
        {#if statusColors[routeStatuses[route]]}<span title={ROUTE_STATUS[routeStatuses[route] as keyof typeof ROUTE_STATUS]?.label} style:background={statusColors[routeStatuses[route]]}></span>{/if}
      </a>
    {/if}
  {/each}
{/snippet}

<section class="new-line-list new-line-list--desktop" class:new-line-list--mobile-selected={mobileLayout === "rows"} aria-label="Subway lines">
  {#each groups as group, index}
    <div class="new-line-row" style:--route-family-color={ROUTE_FAMILIES.find((family)=>group.some((route)=>family.routes.includes(route)))?.color||"#808183"}><div class="new-line-bullets">{@render routeButtons(group)}</div></div>
  {/each}
</section>

<section class="new-line-list new-line-list--mobile-bands" class:new-line-list--mobile-selected={mobileLayout === "grouped"} aria-label="Subway lines">
  {#each sections as section}
    <div class={`new-line-mobile-section new-line-mobile-section--${section.families.length} new-line-mobile-section--${section.layout}`}>
      {#each section.families as familyIndex}
        <div class="new-line-mobile-family" style:--route-family-color={colors[familyIndex]}><span class="new-line-mobile-bullets">{@render routeButtons(mobileGroups[familyIndex], 46)}</span></div>
      {/each}
    </div>
  {/each}
</section>
