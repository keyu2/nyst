<script lang="ts">
  import { getBullet } from "../routes";
  let { route, size = 32, class: className = "",alt,title }: { route: string; size?: number; class?: string;alt?:string;title?:string } = $props();
</script>

<img src={getBullet(route)} alt={alt??`${route} train`} {title} width={size} height={size} class={className} loading="eager" draggable="false" />
