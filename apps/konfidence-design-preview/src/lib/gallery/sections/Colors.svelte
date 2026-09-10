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
    <div class="stack">
        <RampGrid title="Amber (action)" items={AMBER_RAMP} />
        <RampGrid title="Teal (information)" items={TEAL_RAMP} />
        <RampGrid title="Warm neutral" items={NEUTRAL_RAMP} />
        <RampGrid title="Warm dark ('space')" items={DARK_RAMP} />

        <Sample
            title="Chart accents"
            description="Categorical palette for charts — non-valued, accessible pairing."
        >
            <div class="chart-grid">
                {#each CHART_ACCENTS as accent (accent.name)}
                    <ColorSwatch name={accent.name} value={accent.cssVar} />
                {/each}
            </div>
        </Sample>

        <Sample
            title="Gradients"
            description="Signature Konfidence gradients used for hero, tiles, glows."
        >
            <div class="gradient-grid">
                {#each GRADIENTS as gradient (gradient.name)}
                    <div class="gradient-tile">
                        <div class="gradient-tile__chip" style:background={gradient.cssVar}></div>
                        <div class="gradient-tile__meta">
                            <span class="gradient-tile__label">{gradient.label}</span>
                            <code>{gradient.name}</code>
                        </div>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample title="Status roles" description="Semantic color pairs used by StatusBadge.">
            <div class="status-grid">
                {#each STATUS_ROLES as role (role.key)}
                    <div class="status-pill" style:background={role.bg} style:color={role.fg}>
                        <span class="dot" style:background={role.solid}></span>
                        <span class="label">{role.key}</span>
                    </div>
                {/each}
            </div>
        </Sample>
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 24px;
    }

    .chart-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
        width: 100%;
    }

    .gradient-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        gap: 14px;
        width: 100%;
    }

    .gradient-tile {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .gradient-tile__chip {
        height: 80px;
        border-radius: 10px;
        border: 1px solid var(--border-subtle);
    }

    .gradient-tile__meta {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .gradient-tile__label {
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold, 600);
    }

    .gradient-tile__meta code {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .status-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
    }

    .status-pill {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 6px 12px;
        border-radius: 999px;
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold, 600);
    }

    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }

    .label {
        text-transform: capitalize;
    }
</style>
