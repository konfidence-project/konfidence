<script lang="ts">
    interface Props {
        /** Components/patterns that are planned for this section. */
        planned?: string[];
        /** Optional single-sentence explanation. */
        note?: string;
    }

    let { planned = [], note }: Props = $props();
</script>

<div class="nyi">
    <div class="nyi__icon" aria-hidden="true">◌</div>
    <div class="nyi__body">
        <p class="nyi__title">Not yet implemented in <code>@konfidence/design-system</code></p>
        {#if note}
            <p class="nyi__note">{note}</p>
        {/if}
        {#if planned.length > 0}
            <p class="nyi__label">Planned primitives:</p>
            <ul class="nyi__list">
                {#each planned as item (item)}
                    <li>{item}</li>
                {/each}
            </ul>
        {/if}
        <p class="nyi__cta">
            See the roadmap in
            <code>packages/konfidence-design-system/README.md</code>.
        </p>
    </div>
</div>

<style>
    .nyi {
        display: flex;
        align-items: flex-start;
        gap: 14px;
        padding: 18px 20px;
        border: 1px dashed var(--border-default, var(--border-subtle));
        border-radius: 12px;
        background:
            repeating-linear-gradient(
                45deg,
                transparent 0 8px,
                var(--surface-sunken, rgba(0, 0, 0, 0.02)) 8px 12px
            );
        color: var(--text-secondary);
    }

    .nyi__icon {
        font-size: 24px;
        line-height: 1;
        color: var(--text-tertiary, var(--text-secondary));
        flex-shrink: 0;
    }

    .nyi__body {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .nyi__title {
        margin: 0;
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold, 600);
        color: var(--text-primary);
    }

    .nyi__note {
        margin: 0;
        font-size: var(--text-sm);
        max-width: 640px;
    }

    .nyi__label {
        margin: 6px 0 0;
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        text-transform: uppercase;
        letter-spacing: 0.04em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .nyi__list {
        margin: 0;
        padding: 0 0 0 18px;
        font-size: var(--text-sm);
        line-height: 1.55;
    }

    .nyi__cta {
        margin: 8px 0 0;
        font-size: var(--text-meta);
        color: var(--text-tertiary, var(--text-secondary));
    }

    code {
        font-family: var(--font-mono);
        font-size: 0.9em;
        padding: 1px 5px;
        border-radius: 4px;
        background: var(--surface-subtle);
        color: var(--text-primary);
    }
</style>
