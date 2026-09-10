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

<div class="swatch" class:swatch--translucent={translucent}>
    <div class="swatch__chip" style:background={value}></div>
    <div class="swatch__meta">
        <span class="swatch__label">{label ?? name}</span>
        <code class="swatch__code">{name}</code>
    </div>
</div>

<style>
    .swatch {
        display: flex;
        flex-direction: column;
        gap: 6px;
        min-width: 100px;
    }

    .swatch__chip {
        aspect-ratio: 3 / 2;
        border-radius: 8px;
        border: 1px solid var(--border-subtle);
        box-shadow: var(--shadow-xs);
    }

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

    .swatch__meta {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .swatch__label {
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        color: var(--text-primary);
    }

    .swatch__code {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-tertiary, var(--text-secondary));
        overflow-wrap: anywhere;
    }
</style>
