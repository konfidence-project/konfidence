<script lang="ts">
    import type { Snippet } from "svelte";
    import PageHeader from "../PageHeader.svelte";

    interface Props {
        title: string;
        description?: string;
        eyebrow?: Snippet;
        actions?: Snippet;
        id?: string;
    }

    let { title, description, eyebrow, actions, id }: Props = $props();
</script>

<div data-testid="page-header-root" style="padding:24px; background:var(--surface-canvas);">
    <PageHeader {title} {description} {eyebrow} {actions} {id} />
</div>
