<script lang="ts">
    import { SearchInput, Select } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let search = $state("");
    let searchDisabled = $state("");
    let stage = $state("prod");
    let region = $state("");
    const REGIONS = [
        { id: "eu-de-1", label: "eu-de-1" },
        { id: "eu-nl-1", label: "eu-nl-1" },
        { id: "us-east-1", label: "us-east-1" },
        { id: "ap-tokyo-1", label: "ap-tokyo-1" },
    ] as const;
</script>

<Section
    id="forms"
    title="Forms & switches"
    subtitle="Real <SearchInput> and <Select> from @konfidence/design-system. Additional form primitives are still on the roadmap."
    status="partial"
>
    <div class="stack">
        <Sample title="SearchInput" description="Debounced-ready search field with SAP-Icons `search` glyph and a clear button.">
            <SearchInput bind:value={search} placeholder="Search artifacts…" />
            <span class="log">value = <b>{search || "(empty)"}</b></span>
        </Sample>

        <Sample title="SearchInput (disabled)">
            <SearchInput bind:value={searchDisabled} placeholder="Disabled" disabled />
        </Sample>

        <Sample title="Select" description="Native <select> styled with Konfidence tokens; focus-glow follows the amber rule.">
            <Select bind:value={stage} aria-label="Deployment stage">
                <option value="dev">Development</option>
                <option value="staging">Staging</option>
                <option value="prod">Production</option>
            </Select>

            <Select bind:value={region} aria-label="Region">
                <option value="" disabled>Pick a region…</option>
                {#each REGIONS as region (region.id)}
                    <option value={region.id}>{region.label}</option>
                {/each}
            </Select>

            <span class="log">stage = <b>{stage}</b> · region = <b>{region || "(none)"}</b></span>
        </Sample>

        <NotYetImplemented
            note="Skeleton v5 provides the underlying primitives (Switch, SegmentedControl, checkbox, radio). Konfidence has not yet published its opinionated wrappers with amber focus glow and warm-tinted validation states."
            planned={[
                "<TextInput>",
                "<Checkbox>",
                "<Radio> / <RadioGroup>",
                "<Switch variant='amber' | 'onoff' | 'status' | 'segment'>",
                "Amber focus glow across form controls",
            ]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .log {
        font-family: var(--font-mono);
        font-size: var(--text-meta);
        color: var(--text-secondary);
        padding: 3px 8px;
        border-radius: 6px;
        background: var(--surface-subtle);
    }
</style>
