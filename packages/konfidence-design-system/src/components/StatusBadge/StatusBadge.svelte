<script lang="ts">
    import type { ComponentProps } from "svelte";
    import Badge from "../Badge/Badge.svelte";
    // Compatibility entry point. New consumers should use Badge.
    let props: ComponentProps<typeof Badge> = $props();
</script>
<Badge {...props} />
