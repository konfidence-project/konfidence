<script lang="ts">
    import Tag from "../Tag.svelte";
    import type { TagColor, TagSize } from "../types.js";

    interface Props { label?: string; color?: TagColor; size?: TagSize; mono?: boolean; icon?: string; interactive?: boolean; href?: string; disabled?: boolean; onclick?: () => void; }
    let { label = "Promotion", color = "teal", size = "md", mono = false, icon, interactive = false, href, disabled = false, onclick }: Props = $props();
</script>
{#if href !== undefined}
    <Tag {href} {disabled} {onclick} {color} {size} {mono} {icon}>{label}</Tag>
{:else if interactive}
    <Tag interactive {disabled} {onclick} {color} {size} {mono} {icon}>{label}</Tag>
{:else}
    <Tag {color} {size} {mono} {icon}>{label}</Tag>
{/if}
