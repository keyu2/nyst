<script lang="ts">
  import AppNavigation from "#lib/components/AppNavigation.svelte";
  import PageNavigation from "#lib/components/PageNavigation.svelte";
  import { readPreferences, watchPreferences, writePreferences, type Preferences } from "#lib/preferences.ts";

  let preferences = $state(readPreferences());
  const options: { key: keyof Preferences; label: string; values: { value: string | boolean; label: string }[] }[] = [
    { key: "theme", label: "Theme", values: [{ value: "system", label: "System" }, { value: "dark", label: "Dark" }, { value: "light", label: "Light" }] },
    { key: "showSeconds", label: "Show seconds", values: [{ value: false, label: "Off" }, { value: true, label: "On" }] },
    { key: "tripProgressUnit", label: "Arrival estimate", values: [{ value: "minutes", label: "Minutes" }, { value: "stops", label: "Stops away" }] },
    { key: "stationView", label: "Station page", values: [{ value: "standard", label: "Arrival list" }, { value: "board", label: "Station board" }] },
    { key: "lineView", label: "Home page", values: [{ value: "overview", label: "Traditional" }, { value: "focus", label: "Slider" }] },
    { key: "mobileLineLayout", label: "Line layout on phones", values: [{ value: "grouped", label: "Grouped" }, { value: "rows", label: "Full-width rows" }] },
    { key: "mapEnabled", label: "Map", values: [{ value: true, label: "On" }, { value: false, label: "Off" }] },
  ];
  let visibleOptions = $derived(preferences.stationView === "board" ? [...options.slice(0, 4), { key: "boardArrivalInversion" as keyof Preferences, label: "Highlight arriving trains", values: [{ value: true, label: "On" }, { value: false, label: "Off" }] }, ...options.slice(4)] : options);
  function update(key: keyof Preferences, value: string | boolean) {
    preferences = { ...preferences, [key]: value };
    writePreferences(preferences);
  }
  $effect(() => watchPreferences((value) => preferences=value));
</script>

<main class="new-app-shell settings-page">
  <div class="new-status-page">
    <AppNavigation active="settings" mapEnabled={preferences.mapEnabled} />
    <header class="settings-header">
      <PageNavigation home={false}/>
      <h1>Settings</h1>
    </header>
    <div class="settings-list">
      {#each visibleOptions as option}
        <section class="settings-row" class:settings-row--mobile-only={option.key==="mobileLineLayout"}>
          <div class="settings-row__copy"><h2>{option.label}</h2></div>
          <div class="settings-segmented" role="radiogroup" aria-label={option.label}>
            {#each option.values as item}
              <button type="button" role="radio" aria-checked={preferences[option.key] === item.value} class:active={preferences[option.key] === item.value} onclick={() => update(option.key, item.value)}>{item.label}</button>
            {/each}
          </div>
        </section>
        {#if option.key === "showSeconds"}
          <div class="settings-preview"><span>Preview</span><strong>{preferences.showSeconds ? "4m 32s" : "4 min"}</strong></div>
        {/if}
      {/each}
    </div>
  </div>
</main>
