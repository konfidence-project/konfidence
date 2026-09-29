<script lang="ts">
    interface Props {
        /** CSS custom-property name, e.g. `--amber-500`. */
        name: string;
        /** CSS value referencing that variable, e.g. `var(--amber-500)`. */
        value: string;
        /** Optional short caption below the swatch (defaults to `name`). */
        label?: string;
        /**
         * If true, render the swatch with a checker background behind so
         * translucent tokens (glow / scrim) read correctly.
         */
        translucent?: boolean;
    }

    let { name, value, label, translucent = false }: Props = $props();
</script>

<div class={["swatch flex min-w-[100px] flex-col gap-1.5", translucent && "swatch--translucent"]}>
    <div class="swatch__chip aspect-[3/2] rounded-lg border border-outline-subtle shadow-elevation-xs" style:background={value}></div>
    <div class="swatch__meta flex flex-col gap-0.5">
        <span class="swatch__label text-meta font-ui-semibold text-content-primary">{label ?? name}</span>
        <code class="swatch__code font-ui-mono text-[11px] text-content-tertiary [overflow-wrap:anywhere]">{name}</code>
    </div>
</div>

<style>
    .swatch--translucent .swatch__chip {
        background-color: transparent;
        background-image:
            linear-gradient(45deg, var(--surface-sunken) 25%, transparent 25%),
            linear-gradient(-45deg, var(--surface-sunken) 25%, transparent 25%),
            linear-gradient(45deg, transparent 75%, var(--surface-sunken) 75%),
            linear-gradient(-45deg, transparent 75%, var(--surface-sunken) 75%);
        background-size: 12px 12px;
        background-position:
            0 0,
            0 6px,
            6px -6px,
            -6px 0;
    }

</style>
