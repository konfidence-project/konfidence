<script lang="ts">
    import { TreeGroup, TreeRow } from "@konfidence/design-system";
    import Section from "../Section.svelte";
    import Sample from "../Sample.svelte";

    let selectedPromotion = $state<string>();
    let selectedProduction = $state<string>();
    let selectedFile = $state<string | undefined>("package.json");

    const files = [
        { change: "modified", name: "src/index.ts" },
        { change: "modified", name: "package.json" },
        { change: "added", name: "README.md" },
        { change: "added", name: "dist/bundle.js" },
    ];
</script>

<Section
    id="tree-group"
    title="Tree Group"
    subtitle="Expand a group and select a row. Each group keeps its own selection, including when collapsed."
>
    <div class="grid items-start gap-4 xl:grid-cols-2">
        <Sample title="Promotion groups" description="Click a header to expand or collapse. Use Tab, Enter or Space to operate the controls.">
            <div class="flex w-full min-w-0 flex-col gap-3">
                <TreeGroup label="dev-us30 → test-us30" open bind:selectedValue={selectedPromotion}>
                    {#snippet metadata()}3 waiting{/snippet}
                    <TreeRow value="v2.14.0" label="v2.14.0">
                        {#snippet metadata()}by S. Meier · 5 min ago{/snippet}
                    </TreeRow>
                    <TreeRow value="v2.13.5" label="v2.13.5">
                        {#snippet metadata()}by CI · 1h ago{/snippet}
                    </TreeRow>
                </TreeGroup>
                <TreeGroup label="test-jp20 → prod-ap30" bind:selectedValue={selectedProduction}>
                    {#snippet metadata()}1 waiting{/snippet}
                    <TreeRow value="v2.14.0" label="v2.14.0">
                        {#snippet metadata()}by CI · 20 min ago{/snippet}
                    </TreeRow>
                    <TreeRow value="v2.13.3" label="v2.13.3">
                        {#snippet metadata()}by M. Klein · 1d ago{/snippet}
                    </TreeRow>
                </TreeGroup>
            </div>
        </Sample>
        <Sample title="Changed files" description="A preselected row with the amber selection marker. Select another file to move it.">
            <div class="w-full min-w-0">
                <TreeGroup label="Changed Files" open bind:selectedValue={selectedFile}>
                    {#each files as file (file.name)}
                        <TreeRow value={file.name} label={file.name}>
                            {#snippet metadata()}{file.change}{/snippet}
                        </TreeRow>
                    {/each}
                </TreeGroup>
            </div>
        </Sample>
    </div>
</Section>
