<script lang="ts">
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
    import { RADIUS_STEPS, SPACING_STEPS, TYPE_SCALE } from "$lib/tokens/read-tokens";
</script>

<Section
    id="foundation-typography"
    title="Foundation · Typography, spacing & radii"
    subtitle="System font stack, tabular numerals, negative tracking on display sizes."
    status="implemented"
>
    <div class="stack">
        <Sample title="Type scale" description="Rendered live from --text-* tokens.">
            <div class="type-scale">
                {#each TYPE_SCALE as item (item.name)}
                    <div class="type-scale__row">
                        <span class="type-scale__sample" style:font-size={item.cssVar}
                            >{item.sample}</span
                        >
                        <code>{item.name}</code>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample title="Tabular numerals" description="font-feature-settings: 'tnum' on all metrics.">
            <table class="numeric-table">
                <tbody>
                    <tr>
                        <td>Deployments</td>
                        <td>1,234</td>
                    </tr>
                    <tr>
                        <td>Errors (24h)</td>
                        <td>12</td>
                    </tr>
                    <tr>
                        <td>Success rate</td>
                        <td>98.7%</td>
                    </tr>
                </tbody>
            </table>
        </Sample>

        <Sample title="Spacing scale" description="Powers of 2 blended with fibonacci-ish jumps.">
            <div class="spacing">
                {#each SPACING_STEPS as step (step)}
                    <div class="spacing__row">
                        <div class="spacing__bar" style:width="var(--space-{step})"></div>
                        <code>--space-{step}</code>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample
            title="Radii"
            description="Signature radii 10 / 14 / 20 give Konfidence its slightly softer geometry."
        >
            <div class="radii">
                {#each RADIUS_STEPS as step (step)}
                    <div
                        class="radii__tile"
                        style:border-radius="var(--radius-{step})"
                    >
                        <code>--radius-{step}</code>
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
        gap: 20px;
    }

    .type-scale {
        display: flex;
        flex-direction: column;
        gap: 10px;
        width: 100%;
    }

    .type-scale__row {
        display: flex;
        align-items: baseline;
        gap: 16px;
    }

    .type-scale__sample {
        font-weight: var(--weight-display, 600);
        color: var(--text-primary);
        letter-spacing: -0.5px;
    }

    .type-scale__row code {
        font-family: var(--font-mono);
        color: var(--text-tertiary, var(--text-secondary));
        font-size: 12px;
    }

    .numeric-table {
        font-variant-numeric: tabular-nums;
        border-collapse: collapse;
    }

    .numeric-table td {
        padding: 6px 16px;
        border-bottom: 1px solid var(--border-subtle);
        color: var(--text-primary);
        font-size: var(--text-sm);
    }

    .numeric-table td:last-child {
        text-align: right;
        font-weight: var(--weight-semibold, 600);
    }

    .spacing {
        display: flex;
        flex-direction: column;
        gap: 6px;
        width: 100%;
    }

    .spacing__row {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .spacing__bar {
        height: 10px;
        background: var(--accent-primary, var(--amber-500));
        border-radius: 3px;
    }

    .spacing__row code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .radii {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
    }

    .radii__tile {
        width: 72px;
        height: 72px;
        border: 1px solid var(--border-default, var(--border-subtle));
        background: var(--surface-subtle);
        display: flex;
        align-items: flex-end;
        justify-content: center;
        padding: 6px;
    }

    .radii__tile code {
        font-family: var(--font-mono);
        font-size: 10px;
        color: var(--text-tertiary, var(--text-secondary));
    }
</style>
