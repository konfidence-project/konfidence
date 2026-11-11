<script lang="ts">
    import ChangeList from "../ChangeList.svelte";
    import type { ChangeListItem } from "../types.js";

    const DEFAULT_WIDTH = 520;
    let { width = DEFAULT_WIDTH }: { width?: number } = $props();
    const items: ChangeListItem[] = [
        { group: "Artifacts", id: "sdk", name: "analytics-sdk", newValue: "1.2.0", state: "added" },
        { group: "Artifacts", id: "payment", name: "payment-svc", newValue: "2.14.0", oldValue: "2.13.1", state: "changed" },
        { group: "Artifacts", id: "legacy", name: "legacy-adapter", oldValue: "0.9.4", state: "removed" },
        { group: "Artifacts", id: "long", name: "international-payment-reconciliation-service", newValue: "2.14.0-release-candidate.20261111", oldValue: "2.13.1-release-candidate.20261109", state: "changed" },
        { group: "Feature toggles", id: "enabled", label: "Enabled", name: "new-search", newValue: true, oldValue: false, state: "changed" },
        { group: "Feature toggles", id: "disabled", label: "Disabled", name: "legacy-checkout", newValue: false, oldValue: true, state: "changed" },
        { group: "Without values", id: "name", name: "name-only", state: "added" },
    ];
</script>

<div class="change-list-fixture bg-surface-card p-4 text-content-primary" style:width={`${width}px`}>
    <ChangeList {items} ariaLabel="Example changes" />
    <ChangeList items={[]} emptyLabel="No artifact changes in this promotion." />
</div>
