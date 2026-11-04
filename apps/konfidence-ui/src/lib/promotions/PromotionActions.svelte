<script lang="ts">
    import { Button } from "@konfidence/design-system/components";

    interface Props { explain?: boolean; }
    let { explain = true }: Props = $props();
</script>

<div class="flex max-w-sm flex-col gap-2">
    <div class="flex flex-wrap gap-2">
        <Button variant="primary" disabled>Approve</Button>
        <Button variant="danger" disabled>Reject</Button>
    </div>
    {#if explain}<p class="m-0 text-meta text-content-secondary">Approval and rejection are unavailable in the dashboard in this release.</p>{/if}
</div>
