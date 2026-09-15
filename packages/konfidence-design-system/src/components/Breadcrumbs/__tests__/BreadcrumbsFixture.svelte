<script lang="ts">
    import Breadcrumbs from "../Breadcrumbs.svelte";
    import type { BreadcrumbItem } from "../types.js";

    interface Props {
        items: BreadcrumbItem[];
    }

    let { items }: Props = $props();
</script>

<div data-testid="breadcrumbs-root" style="padding:24px; background:var(--surface-canvas);">
    <Breadcrumbs {items} />
</div>
