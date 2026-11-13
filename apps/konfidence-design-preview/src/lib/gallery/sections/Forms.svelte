<script lang="ts">
    import { SearchInput, Select } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let { name = "SearchInput" }: { name?: string } = $props();

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
    id={name}
    title={name}
>
    <div class="stack flex flex-col gap-5">
        {#if name === "SearchInput"}
        <Sample title="SearchInput" description="Debounced-ready search field with SAP-Icons `search` glyph and a clear button.">
            <SearchInput bind:value={search} placeholder="Search artifacts…" />
            <span class="log rounded-md bg-surface-subtle px-2 py-[3px] font-mono text-meta text-content-secondary">value = <b>{search || "(empty)"}</b></span>
        </Sample>

        <Sample title="SearchInput (disabled)">
            <SearchInput bind:value={searchDisabled} placeholder="Disabled" disabled />
        </Sample>

        {/if}
        {#if name === "Select"}
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

            <span class="log rounded-md bg-surface-subtle px-2 py-[3px] font-mono text-meta text-content-secondary">stage = <b>{stage}</b> · region = <b>{region || "(none)"}</b></span>
        </Sample>

        {/if}
    </div>
</Section>
