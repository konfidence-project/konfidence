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

<style>
    .avatar-group {
        display: inline-flex;
    }
    .avatar-group :global(.avatar) {
        width: 30px;
        height: 30px;
        font-size: 11px;
        border: 2px solid var(--surface-default);
        margin-left: -8px;
    }
    .avatar-group :global(.avatar:first-child) {
        margin-left: 0;
    }
    .avatar-group__more {
        width: 30px;
        height: 30px;
        border-radius: 50%;
        background: var(--surface-sunken);
        color: var(--text-secondary);
        border: 2px solid var(--surface-default);
        margin-left: -8px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-size: 11px;
        font-weight: var(--weight-semibold);
    }
</style>
