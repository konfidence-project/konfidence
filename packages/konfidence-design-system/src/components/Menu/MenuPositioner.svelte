<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    /**
     * `Menu.Positioner` with the Konfidence `.menu-positioner` class
     * pre-applied. Skeleton's positioner is placed inline by Zag with an
     * explicit `z-index`; the class rule uses `!important` to override
     * so the menu stacks above the mobile drawer.
     */
    type SkPositionerProps = ComponentProps<typeof SkMenu.Positioner>;

    let { class: className, ...rest }: SkPositionerProps = $props();

    const composedClass = $derived(
        className ? `menu-positioner ${className}` : "menu-positioner",
    );
</script>

<SkMenu.Positioner class={composedClass} {...rest} />
