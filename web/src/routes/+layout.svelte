<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/state";
  import { applyTheme, readPreferences, watchPreferences } from "#lib/preferences.ts";
  import "../app.css";
  let { children } = $props();
  onMount(() => {
    applyTheme(readPreferences().theme);
    return watchPreferences((value) => applyTheme(value.theme));
  });
</script>

{#key page.url.href}{@render children()}{/key}
