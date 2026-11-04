<script lang="ts">
  import Icon from "./Icon.svelte";
  import { goto } from "$app/navigation";
  let { active = "subway", mapEnabled = false }: { active?: string; mapEnabled?: boolean } = $props();
  const baseItems = [
    { id: "subway", label: "Home", href: "/subway", icon: "House" as const },
    { id: "favorites", label: "Favorite", href: "/subway/favorites", icon: "Star" as const },
    { id: "nearby", label: "Nearby", href: "/subway/nearby", icon: "MapPin" as const },
    { id: "status", label: "Status", href: "/status", icon: "TriangleAlert" as const },
  ];
  const items = $derived(mapEnabled
    ? [...baseItems.slice(0, 3), { id: "map", label: "Map", href: "/map", icon: "Map" as const }, baseItems[3]]
    : baseItems);
</script>

<header class="app-navigation">
  <a class="app-navigation__brand" href="/subway">NYST</a>
  <nav class="new-tabs new-tabs--primary" data-item-count={items.length} aria-label="Main sections">
    {#each items as item}
      <button type="button" class:active={active === item.id} aria-current={active === item.id ? "page" : undefined} aria-pressed={active===item.id} onclick={() => goto(item.href)}>
        <Icon name={item.icon}/>
        <span>{item.label}</span>
      </button>
    {/each}
  </nav>
  <a class="app-navigation__settings" class:active={active === "settings"} href="/settings" aria-current={active === "settings" ? "page" : undefined} aria-label="Settings">
    <Icon name="Settings"/>
    <span>Settings</span>
  </a>
</header>

