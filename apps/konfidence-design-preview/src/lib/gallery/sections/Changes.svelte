<script lang="ts">
    import { ChangeList, TogglePill } from "@konfidence/design-system/components";
    import type { ChangeListItem } from "@konfidence/design-system/components";
    import Section from "../Section.svelte";
    import Sample from "../Sample.svelte";

    const artifacts: ChangeListItem[] = [
        { id: "sdk", name: "analytics-sdk", newValue: "1.2.0", state: "added" },
        { id: "payment", name: "payment-svc", newValue: "2.14.0", oldValue: "2.13.1", state: "changed" },
        { id: "gateway", name: "api-gateway", newValue: "5.1.0", oldValue: "5.0.9", state: "changed" },
        { id: "legacy", name: "legacy-adapter", oldValue: "0.9.4", state: "removed" },
    ];
    const toggles: ChangeListItem[] = [
        { id: "search", label: "Enabled", name: "new-search", newValue: true, oldValue: false, state: "changed" },
        { id: "checkout", label: "Disabled", name: "legacy-checkout", newValue: false, oldValue: true, state: "changed" },
    ];
    const namesOnly: ChangeListItem[] = [
        { group: "Added", id: "added", name: "analytics-sdk", state: "added" },
        { group: "Changed", id: "changed", name: "payment-svc", state: "changed" },
        { group: "Removed", id: "removed", name: "legacy-adapter", state: "removed" },
    ];
    const longContent: ChangeListItem[] = [
        { id: "long", name: "international-payment-reconciliation-service", newValue: "2.14.0-release-candidate.20261111", oldValue: "2.13.1-release-candidate.20261109", state: "changed" },
        { id: "long-added", name: "observability-collector", newValue: "1.2.0+build.abcdefghijklmnopqrstuvwxyz0123456789", state: "added" },
    ];
</script>

<Section id="change-list" title="Change list" subtitle="Compact diffs for artifacts and feature toggles. Color, a visible label and previous/new values explain each change.">
    <div class="grid min-w-0 gap-5">
        <Sample title="Artifact changes (4)" description="Added, Changed and Removed with optional version values.">
            <ChangeList items={artifacts} ariaLabel="Artifact changes" class="w-full max-w-[560px]" />
        </Sample>
        <Sample title="Feature toggle changes (2)" description="Enabled and Disabled share the Changed tone. On/Off pills display read-only values.">
            <ChangeList items={toggles} ariaLabel="Feature toggle changes" class="w-full max-w-[560px]" />
        </Sample>
        <Sample title="Without values · grouped" description="All three states also work with a name only. Optional labels separate consecutive groups.">
            <ChangeList items={namesOnly} ariaLabel="Grouped changes without values" class="w-full max-w-[560px]" />
        </Sample>
        <Sample title="Long names and values · narrow panel" description="Names truncate with the full name available on hover. Values wrap below the name in narrow containers.">
            <ChangeList items={longContent} ariaLabel="Long changes" class="w-full max-w-[320px]" />
        </Sample>
        <Sample title="Empty sections (0)" description="A short inline hint replaces the list when there are no changes.">
            <div class="w-full"><ChangeList items={[]} emptyLabel="No artifact changes in this promotion." /><ChangeList items={[]} emptyLabel="No feature toggle changes in this promotion." /></div>
        </Sample>
        <Sample title="Toggle pill" description="Read-only boolean status, also available outside a change list."><TogglePill on={true} /><TogglePill on={false} /></Sample>
    </div>
</Section>
