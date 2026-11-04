<script lang="ts">
  import { page } from "$app/state";
  import Home from "#lib/pages/Home.svelte";
</script>
<Home focusedRoute={page.url.searchParams.get("line")||""}/>
