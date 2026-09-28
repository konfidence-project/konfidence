<script lang="ts">
    import type { Snippet } from "svelte";
    import Avatar from "../Avatar/Avatar.svelte";

    interface Props {
        /** User initials rendered inside the header avatar. */
        initials?: string;
        /** Optional slot to replace the default avatar entirely. */
        avatar?: Snippet;
        /** Primary line (usually the display name). */
        name: Snippet;
        /** Secondary line (usually the email address). */
        mail?: Snippet;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { initials, avatar, name, mail, class: className }: Props = $props();

</script>

<div class={["menu__header", className]}>
    {#if avatar}
        {@render avatar()}
    {:else if initials}
        <Avatar {initials} />
    {/if}
    <div class="menu__header-info min-w-0">
        <div
            class="menu__header-name overflow-hidden text-ui-sm font-semibold text-ellipsis whitespace-nowrap text-content-primary"
        >{@render name()}</div>
        {#if mail}
            <div
                class="menu__header-mail overflow-hidden text-meta text-ellipsis whitespace-nowrap text-content-tertiary"
            >{@render mail()}</div>
        {/if}
    </div>
</div>
