<script lang="ts">
    import "@ui5/webcomponents-icons/dist/AllIcons.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import { Button, EmptyState } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="feedback"
    title="Feedback states"
    subtitle="Real <EmptyState> from @konfidence/design-system (all three tones). Toasts and skeleton loaders are still on the roadmap."
    status="partial"
>
    <div class="stack flex flex-col gap-5">
        <Sample title="EmptyState — empty tone" description="Neutral, `role='status'`, polite live region.">
            <div class="tone-frame w-full rounded-base border border-outline-subtle bg-surface-subtle">
                <EmptyState
                    title="No artifact deployments"
                    description="Once artifacts are promoted through the pipeline they will show up here."
                >
                    {#snippet icon()}
                        <ui5-icon name="product"></ui5-icon>
                    {/snippet}
                    {#snippet action()}
                        <Button variant="primary">Deploy something</Button>
                    {/snippet}
                </EmptyState>
            </div>
        </Sample>

        <Sample title="EmptyState — info tone">
            <div class="tone-frame w-full rounded-base border border-outline-subtle bg-surface-subtle">
                <EmptyState
                    tone="info"
                    title="Nothing matches those filters"
                    description="Try broadening the status filter or clearing the search term."
                >
                    {#snippet icon()}
                        <ui5-icon name="search"></ui5-icon>
                    {/snippet}
                </EmptyState>
            </div>
        </Sample>

        <Sample title="EmptyState — error tone" description="Uses `role='alert'` + assertive live region.">
            <div class="tone-frame w-full rounded-base border border-outline-subtle bg-surface-subtle">
                <EmptyState
                    tone="error"
                    title="Failed to load deployments"
                    description="The API returned an unexpected error. You can retry in a moment."
                >
                    {#snippet icon()}
                        <ui5-icon name="error"></ui5-icon>
                    {/snippet}
                    {#snippet action()}
                        <Button variant="secondary">Retry</Button>
                    {/snippet}
                </EmptyState>
            </div>
        </Sample>

        <NotYetImplemented
            note="Skeleton v5 provides the underlying Toast primitive. Konfidence has not yet wrapped it, nor the skeleton loaders."
            planned={["<Toast>", "<SkeletonBox> / <SkeletonText>"]}
        />
    </div>
</Section>
