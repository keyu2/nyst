<script lang="ts">
  const paths = {
  "Search": "<path d=\"m21 21-4.34-4.34\"></path><circle cx=\"11\" cy=\"11\" r=\"8\"></circle>",
  "Star": "<path d=\"M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.123 2.123 0 0 0 1.595 1.16l5.166.756a.53.53 0 0 1 .294.904l-3.736 3.638a2.123 2.123 0 0 0-.611 1.878l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.122 2.122 0 0 0-1.973 0L6.396 21.01a.53.53 0 0 1-.77-.56l.881-5.139a2.122 2.122 0 0 0-.611-1.879L2.16 9.795a.53.53 0 0 1 .294-.906l5.165-.755a2.122 2.122 0 0 0 1.597-1.16z\"></path>",
  "ChevronRight": "<path d=\"m9 18 6-6-6-6\"></path>",
  "ChevronDown": "<path d=\"m6 9 6 6 6-6\"></path>",
  "House": "<path d=\"M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8\"></path><path d=\"M3 10a2 2 0 0 1 .709-1.528l7-6a2 2 0 0 1 2.582 0l7 6A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z\"></path>",
  "Map": "<path d=\"M14.106 5.553a2 2 0 0 0 1.788 0l3.659-1.83A1 1 0 0 1 21 4.619v12.764a1 1 0 0 1-.553.894l-4.553 2.277a2 2 0 0 1-1.788 0l-4.212-2.106a2 2 0 0 0-1.788 0l-3.659 1.83A1 1 0 0 1 3 19.381V6.618a1 1 0 0 1 .553-.894l4.553-2.277a2 2 0 0 1 1.788 0z\"></path><path d=\"M15 5.764v15\"></path><path d=\"M9 3.236v15\"></path>",
  "MapPin": "<path d=\"M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0\"></path><circle cx=\"12\" cy=\"10\" r=\"3\"></circle>",
  "Settings": "<path d=\"M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915\"></path><circle cx=\"12\" cy=\"12\" r=\"3\"></circle>",
  "TriangleAlert": "<path d=\"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3\"></path><path d=\"M12 9v4\"></path><path d=\"M12 17h.01\"></path>",
  "Maximize": "<path d=\"M8 3H5a2 2 0 0 0-2 2v3\"></path><path d=\"M21 8V5a2 2 0 0 0-2-2h-3\"></path><path d=\"M3 16v3a2 2 0 0 0 2 2h3\"></path><path d=\"M16 21h3a2 2 0 0 0 2-2v-3\"></path>",
  "Minimize": "<path d=\"M8 3v3a2 2 0 0 1-2 2H3\"></path><path d=\"M21 8h-3a2 2 0 0 1-2-2V3\"></path><path d=\"M3 16h3a2 2 0 0 1 2 2v3\"></path><path d=\"M16 21v-3a2 2 0 0 1 2-2h3\"></path>",
  "X": "<path d=\"M18 6 6 18\"></path><path d=\"m6 6 12 12\"></path>",
  "ArrowUpRight": "<path d=\"M7 7h10v10\"></path><path d=\"M7 17 17 7\"></path>"
} as const;
  let { name, size = 24, fill = "none", class: className = "" }: { name: keyof typeof paths; size?: number; fill?: string; class?: string } = $props();
</script>

<svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" {fill} stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class={className} aria-hidden="true">{@html paths[name]}</svg>
