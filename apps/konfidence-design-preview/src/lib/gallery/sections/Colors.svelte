<script lang="ts">
    import ColorSwatch from "../ColorSwatch.svelte";
    import RampGrid from "../RampGrid.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
    import {
        AMBER_RAMP,
        CHART_ACCENTS,
        DARK_RAMP,
        GRADIENTS,
        NEUTRAL_RAMP,
        STATUS_ROLES,
        TEAL_RAMP,
    } from "$lib/tokens/read-tokens";
</script>

<Section
    id="foundation-colors"
    title="Foundation · Colors"
    subtitle="Palette ramps, gradients, glows and status roles rendered live from tokens.css."
    status="implemented"
>
    <div class="flex flex-col gap-6">
        <RampGrid title="Amber (action)" items={AMBER_RAMP} />
        <RampGrid title="Teal (information)" items={TEAL_RAMP} />
        <RampGrid title="Warm neutral" items={NEUTRAL_RAMP} />
        <RampGrid title="Warm dark ('space')" items={DARK_RAMP} />

        <Sample
            title="Chart accents"
            description="Categorical palette for charts — non-valued, accessible pairing."
        >
            <div class="grid w-full grid-cols-[repeat(auto-fill,minmax(120px,1fr))] gap-3">
                {#each CHART_ACCENTS as accent (accent.name)}
                    <ColorSwatch name={accent.name} value={accent.cssVar} />
                {/each}
            </div>
        </Sample>

        <Sample
            title="Gradients"
            description="Signature Konfidence gradients used for hero, tiles, glows."
        >
            <div class="grid w-full grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-3.5">
                {#each GRADIENTS as gradient (gradient.name)}
                    <div class="flex flex-col gap-1.5">
                        <div class="h-20 rounded-[10px] border border-outline-subtle" style:background={gradient.cssVar}></div>
                        <div class="flex flex-col gap-0.5">
                            <span class="text-ui-sm font-ui-semibold">{gradient.label}</span>
                            <code class="font-ui-mono text-[11px] text-content-tertiary">{gradient.name}</code>
                        </div>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample title="Status roles" description="Semantic color pairs used by StatusBadge.">
            <div class="flex flex-wrap gap-2.5">
                {#each STATUS_ROLES as role (role.key)}
                    <div class="inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-ui-sm font-ui-semibold" style:background={role.bg} style:color={role.fg}>
                        <span class="size-2 rounded-full" style:background={role.solid}></span>
                        <span class="capitalize">{role.key}</span>
                    </div>
                {/each}
            </div>
        </Sample>
    </div>
</Section>
