<script lang="ts">
  import { page } from "$app/state";
  import Home from "#lib/pages/Home.svelte";
</script>
<Home mode={page.params.section||"subway"} focusedRoute={page.url.searchParams.get("line")||""}/>
