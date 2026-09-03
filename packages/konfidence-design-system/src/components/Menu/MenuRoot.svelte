<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    /**
     * Root of the Konfidence menu — a thin passthrough over Skeleton's
     * `Menu` state provider so consumers can write `<Menu>…</Menu>`.
     */
    let { children, ...rest }: ComponentProps<typeof SkMenu> = $props();
</script>

<SkMenu {...rest}>{@render children?.()}</SkMenu>
