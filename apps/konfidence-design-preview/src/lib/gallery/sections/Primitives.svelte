<script lang="ts">
    import { BrandLogo, Brandbar, ColorModeSelect, Link, OrbitLoader, StageCardPlaceholder } from "@konfidence/design-system/components";
    import type { ColorMode } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let { name = "Link" }: { name?: string } = $props();
    let clicks = $state(0);
    let appearance = $state<ColorMode>("system");
</script>

<Section id={name} title={name}>
{#if name === "Brandbar"}
    <Sample title="Brand strip"><div class="w-full"><Brandbar /></div></Sample>
{:else if name === "BrandLogo"}
    <Sample title="Brand logo"><BrandLogo class="w-48" /></Sample>
{:else if name === "ColorModeSelect"}
    <Sample title="Appearance selection"><ColorModeSelect value={appearance} onValueChange={(value) => appearance = value} /><span>Selected: {appearance}</span></Sample>
{:else if name === "OrbitLoader"}
    <Sample title="Loading indicator"><OrbitLoader /><OrbitLoader label="Loading deployments" /></Sample>
{:else if name === "StageCardPlaceholder"}
    <Sample title="Empty stage"><div class="w-full max-w-72"><StageCardPlaceholder title="Production" landscapeName="checkout" message="No stages configured" /></div></Sample>
{:else if name === "Link"}
    <Sample title="Navigation and action links">
        <Link href="?component=Button">View Button</Link>
        <Link onclick={() => clicks += 1}>Run action</Link>
        <Link disabled>Disabled action</Link>
        <span class="text-compact" aria-live="polite">Actions: {clicks}</span>
    </Sample>
{/if}
</Section>
