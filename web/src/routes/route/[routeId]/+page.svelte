<script lang="ts">
  import { page } from "$app/state";
  import RoutePage from "#lib/pages/RoutePage.svelte";
</script>
<RoutePage routeId={page.params.routeId||""}/>
