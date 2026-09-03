<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /** Slot for a trailing `+N` indicator (`.avatar-group__more`). */
        more?: Snippet;
        /** Default slot for `<Avatar>` children. */
        children: Snippet;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { more, children, class: className }: Props = $props();

    const composedClass = $derived(
        className ? `avatar-group ${className}` : "avatar-group",
    );
</script>

<div class={composedClass}>
    {@render children()}
    {#if more}
        <span class="avatar-group__more">{@render more()}</span>
    {/if}
</div>
