<script lang="ts">
    interface Props {
        /** Components/patterns that are planned for this section. */
        planned?: string[];
        /** Optional single-sentence explanation. */
        note?: string;
    }

    let { planned = [], note }: Props = $props();
</script>

<div class="nyi flex items-start gap-3.5 rounded-xl border border-dashed border-outline-default px-5 py-[18px] text-content-secondary">
    <div class="nyi__icon shrink-0 text-2xl leading-none text-content-tertiary" aria-hidden="true">◌</div>
    <div class="nyi__body flex flex-col gap-1.5 [&_code]:rounded [&_code]:bg-surface-subtle [&_code]:px-[5px] [&_code]:py-px [&_code]:font-mono [&_code]:text-[0.9em] [&_code]:text-content-primary">
        <p class="nyi__title m-0 text-compact font-semibold text-content-primary">Not yet implemented in <code>@konfidence/design-system</code></p>
        {#if note}
            <p class="nyi__note m-0 max-w-[640px] text-compact">{note}</p>
        {/if}
        {#if planned.length > 0}
            <p class="nyi__label mt-1.5 text-meta font-semibold tracking-[0.04em] text-content-tertiary uppercase">Planned primitives:</p>
            <ul class="nyi__list m-0 pl-[18px] text-compact leading-[1.55]">
                {#each planned as item (item)}
                    <li>{item}</li>
                {/each}
            </ul>
        {/if}
        <p class="nyi__cta mt-2 text-meta text-content-tertiary">
            See the roadmap in
            <code>packages/konfidence-design-system/README.md</code>.
        </p>
    </div>
</div>

<style>
    .nyi {
        background:
            repeating-linear-gradient(
                45deg,
                transparent 0 8px,
                var(--surface-sunken, rgba(0, 0, 0, 0.02)) 8px 12px
            );
    }
</style>
