<script lang="ts">
  import { page } from "$app/state";
  import StatusPage from "#lib/pages/StatusPage.svelte";
</script>
<StatusPage section={page.params.section||"subway"}/>
