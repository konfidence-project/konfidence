<script lang="ts">
    import StagePhase from "../StagePhase/StagePhase.svelte";
    import type { StagePhaseItem } from "../StagePhase/types.js";
    import type { StageCardStatusRole } from "./types.js";

    interface Props {
        title: string;
        landscapeName?: string;
        href: string;
        ariaLabel: string;
        statusRole: StageCardStatusRole;
        phases: StagePhaseItem[];
        phaseAriaLabel: string;
        targetVector?: string;
        activeVersionText: string;
        live?: boolean;
        selected?: boolean;
    }

    let {
        title,
        landscapeName,
        href,
        ariaLabel,
        statusRole,
        phases,
        phaseAriaLabel,
        targetVector,
        activeVersionText,
        live = false,
        selected = false,
    }: Props = $props();
    const statusId = $props.id();

    const ACCENT_TOKEN: Record<StageCardStatusRole, string> = {
        deploying: "var(--status-deploying-solid)",
        error: "var(--status-error-solid)",
        healthy: "var(--status-healthy-solid)",
        neutral: "var(--border-strong)",
        warning: "var(--status-warning-solid)",
    };
    const stageAccent = $derived(ACCENT_TOKEN[statusRole]);
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- The consumer is responsible for supplying a resolved URL. -->
<a {href}
    class={[
        "stage-card box-border relative flex h-64 w-full flex-col gap-3 overflow-hidden rounded-[var(--card-radius)] border border-outline-subtle bg-surface-card p-4 pt-[calc(var(--space-4)+3px)] text-content-primary no-underline shadow-elevation-sm",
        "transition-[transform,box-shadow,border-color] duration-200 hover:-translate-y-0.5 hover:shadow-elevation-md not-[.stage-card--selected]:hover:border-outline-focus motion-reduce:transition-none motion-reduce:hover:translate-y-0",
        "focus-visible:outline-2 focus-visible:outline-offset-3 focus-visible:outline-outline-focus",
        selected && "stage-card--selected",
    ]}
    data-status={statusRole}
    style:--stage-accent={stageAccent}
    aria-label={ariaLabel}
    aria-describedby={statusId}
>
    <span
        class="pointer-events-none absolute inset-x-0 top-0 h-[3px]"
        style:background="var(--stage-accent)"
        aria-hidden="true"
    ></span>

    <header class="flex items-baseline gap-3">
        <div class="flex min-w-0 flex-col gap-0.5">
            {#if landscapeName}
                <span
                    class="truncate text-[length:var(--text-meta)] font-semibold uppercase tracking-wide text-[var(--text-tertiary)]"
                    title={landscapeName}
                >
                    {landscapeName}
                </span>
            {/if}
            <h4
                class="m-0 truncate text-[length:var(--text-body)] font-bold"
                {title}
            >
                {title}
            </h4>
        </div>
        {#if live}
            <span
                class="ml-auto inline-flex shrink-0 items-center gap-1.5 text-meta font-semibold text-status-healthy-fg"
            >
                <span
                    class="size-2 shrink-0 rounded-full bg-status-healthy-solid"
                    aria-hidden="true"
                ></span>
                live
            </span>
        {/if}
    </header>

    <div class="flex min-h-14 flex-col gap-1">
        <span class="text-meta text-content-tertiary"
            >Target vector</span
        >
        {#if targetVector}
            <span
                class="line-clamp-2 font-mono text-compact leading-normal [overflow-wrap:anywhere]"
                title={targetVector}>{targetVector}</span
            >
        {:else}
            <span
                class="text-compact text-content-secondary"
                >No target version yet</span
            >
        {/if}
    </div>

    <div class="flex flex-col gap-1">
        <span class="text-meta text-content-tertiary"
            >Active version</span
        >
        <span
            class="truncate text-compact text-content-secondary"
            title={activeVersionText}
        >
            {activeVersionText}
        </span>
    </div>

    <footer
        class="mt-auto border-t border-outline-subtle pt-3"
        id={statusId}
    >
        <StagePhase {phases} ariaLabel={phaseAriaLabel} />
    </footer>
</a>

<style>
    .stage-card--selected {
        border-color: var(--stage-accent);
        box-shadow:
            0 0 0 3px
                color-mix(in srgb, var(--stage-accent) 32%, transparent),
            var(--card-rim-glow);
    }

    .stage-card--selected:hover {
        transform: none;
        box-shadow:
            0 0 0 3px
                color-mix(in srgb, var(--stage-accent) 32%, transparent),
            var(--shadow-md);
    }
</style>
