<script lang="ts">
  import { goto } from "$app/navigation";
  let { fallbackHref="/subway", home = true }: {fallbackHref?:string;home?:boolean} = $props();
  function back() { if (history.length > 1) history.back(); else goto(fallbackHref); }
</script>
<button type="button" class="nav-button nav-button--back" onclick={back}>Back</button>
{#if home}<a href="/subway" class="nav-button nav-button--home">Home</a>{/if}
