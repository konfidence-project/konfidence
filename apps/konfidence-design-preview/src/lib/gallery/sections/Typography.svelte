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
    <div class="stack flex flex-col gap-5">
        <Sample title="Type scale" description="Rendered live from --text-* tokens.">
            <div class="type-scale flex w-full flex-col gap-2.5">
                {#each TYPE_SCALE as item (item.name)}
                    <div class="type-scale__row flex items-baseline gap-4">
                        <span class="type-scale__sample font-ui-display tracking-[-0.5px] text-content-primary" style:font-size={item.cssVar}
                            >{item.sample}</span
                        >
                        <code class="font-ui-mono text-meta text-content-tertiary">{item.name}</code>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample title="Tabular numerals" description="font-feature-settings: 'tnum' on all metrics.">
            <table class="numeric-table border-collapse [font-variant-numeric:tabular-nums] [&_td]:border-b [&_td]:border-outline-subtle [&_td]:px-4 [&_td]:py-1.5 [&_td]:text-ui-sm [&_td]:text-content-primary [&_td:last-child]:text-right [&_td:last-child]:font-ui-semibold">
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
            <div class="spacing flex w-full flex-col gap-1.5">
                {#each SPACING_STEPS as step (step)}
                    <div class="spacing__row flex items-center gap-3">
                        <div class="spacing__bar h-2.5 rounded-[3px] bg-[var(--accent-primary,var(--amber-500))]" style:width="var(--space-{step})"></div>
                        <code class="font-ui-mono text-meta text-content-tertiary">--space-{step}</code>
                    </div>
                {/each}
            </div>
        </Sample>

        <Sample
            title="Radii"
            description="Signature radii 10 / 14 / 20 give Konfidence its slightly softer geometry. `--radius-pill` is a shape token, not a scale step — it clamps to a stadium on elongated elements."
        >
            <div class="radii flex flex-wrap gap-3">
                {#each RADIUS_STEPS as step (step)}
                    <div
                        class={[
                            "radii__tile flex h-[72px] items-end justify-center border border-outline-default bg-surface-subtle p-1.5",
                            step === "pill" ? "radii__tile--pill w-40" : "w-[72px]",
                        ]}
                        style:border-radius="var(--radius-{step})"
                    >
                        <code class="font-ui-mono text-[10px] text-content-tertiary">--radius-{step}</code>
                    </div>
                {/each}
            </div>
        </Sample>
    </div>
</Section>
