<script lang="ts">
  import RouteBullet from "./RouteBullet.svelte";
  import { normalizeRouteId } from "../routes";
  let { text = "" }: { text?: string } = $props();
  const routeIds = new Set(["1", "2", "3", "4", "5", "6", "6X", "7", "7X", "A", "C", "E", "B", "D", "F", "FX", "M", "G", "J", "Z", "L", "N", "Q", "R", "W", "S", "GS", "FS", "H", "SI"]);
  const icons = [
    { pattern: /accessib|wheelchair/i, file: "accessible", label: "Accessible station", size: 20, offset: "-4px" },
    { pattern: /elevator/i, file: "elevator", label: "Elevator", size: 20, offset: "-4px" },
    { pattern: /shuttle\s*bus/i, file: "shuttle-bus", label: "Shuttle bus", size: 21, offset: "-5px" },
    { pattern: /\bbus\b/i, file: "bus", label: "Bus", size: 21, offset: "-5px" },
  ];
</script>
{#each String(text || "").split(/(\[[^\]]+\])/g) as part}
  {@const value = part.match(/^\[([^\]]+)\]$/)?.[1].trim()}
  {@const route = value ? normalizeRouteId(value.toUpperCase()) : ""}
  {@const icon = value ? icons.find((item) => item.pattern.test(value)) : undefined}
  {#if route && routeIds.has(route)}<RouteBullet {route} size={21} title={`${value?.toUpperCase()} train`} class="rich-alert-inline" />{:else if icon}<img class="rich-alert-inline" src={`/icons/${icon.file}.svg`} alt={icon.label} title={icon.label} width={icon.size} height={icon.size} style:--rich-alert-offset={icon.offset}/>{:else}{part}{/if}
{/each}
