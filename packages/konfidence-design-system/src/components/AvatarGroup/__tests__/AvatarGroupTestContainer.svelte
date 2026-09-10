<script lang="ts">
    import Avatar from "../../Avatar/Avatar.svelte";
    import AvatarGroup from "../AvatarGroup.svelte";

    interface Props {
        showMore?: boolean;
    }

    let { showMore = true }: Props = $props();
</script>

<div style="padding: 16px; background: var(--surface-default);">
    <AvatarGroup>
        <Avatar initials="RM" />
        <Avatar initials="AK" />
        <Avatar initials="JD" />
        {#snippet more()}
            {#if showMore}+5{/if}
        {/snippet}
    </AvatarGroup>
</div>
