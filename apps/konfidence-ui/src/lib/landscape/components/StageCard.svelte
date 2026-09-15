<script lang="ts">
    import { StagePhase } from "@konfidence/design-system/components";
    import type { StagePhaseItem } from "@konfidence/design-system/components";
    import type { Stage } from "$lib/landscape/landscapeApi";
    import type { StageVersion } from "$lib/landscape/stageStatus";
    import { stageStatuses } from "$lib/landscape/stageStatus";

    interface Props {
        stage: Stage;
        landscapeName: string;
        category?: string;
        href: string;
        /**
         * Applies the design-system selected ring — a 3 px halo in the
         * card's current status colour. Owned by the parent (e.g. the
         * `LandscapeFlow` selection state).
         */
        selected?: boolean;
    }
    let {
        stage,
        landscapeName,
        category,
        href,
        selected = false,
    }: Props = $props();
    const statusId = $props.id();
    const target = $derived(stage.targetStageVersion);
    const active = $derived(stage.activeStageVersion);
    const targetIsActive = $derived(
        target !== undefined && target.id === active?.id,
    );

    /**
     * Card status role — mapped from the API stage-version status onto
     * the design-system's canonical stripe/ring vocabulary. Drives the
     * top stripe colour, the selected ring colour (both via
     * `--stage-accent`), and the `data-status` hook used for tests.
     */
    type StatusRole = "healthy" | "deploying" | "warning" | "error" | "neutral";

    const ACCENT_TOKEN: Record<StatusRole, string> = {
        deploying: "var(--status-deploying-solid)",
        error: "var(--status-error-solid)",
        healthy: "var(--status-healthy-solid)",
        neutral: "var(--border-strong)",
        warning: "var(--status-warning-solid)",
    };

    const statusRole: StatusRole = $derived.by((): StatusRole => {
        if (!target) {
            return "neutral";
        }
        switch (target.status) {
            case "Ready": {
                return targetIsActive ? "healthy" : "neutral";
            }
            case "DeployingVector":
            case "MigratingVector":
            case "ActivatingVector": {
                return "deploying";
            }
            case "Failed": {
                return "error";
            }
            case "PendingDeployment": {
                return "neutral";
            }
            default: {
                return "neutral";
            }
        }
    });
    const stageAccent = $derived(ACCENT_TOKEN[statusRole]);

    /**
     * Build the 3-segment phase strip from the target status. Mirrors
     * the design-system `.stage-progress` semantics:
     *
     *   • done    — completed step
     *   • active  — currently running (60 % blue fill)
     *   • failed  — this step failed (100 % red fill)
     *   • pending — not started yet
     */
    type PhaseState = StagePhaseItem["state"];
    const buildPhases = (
        deploy: PhaseState,
        migrate: PhaseState,
        activate: PhaseState,
    ): StagePhaseItem[] => [
        { label: "Deploy", state: deploy },
        { label: "Migrate", state: migrate },
        { label: "Activate", state: activate },
    ];
    const phases: StagePhaseItem[] = $derived.by(() => {
        if (!target) {
            return buildPhases("pending", "pending", "pending");
        }
        switch (target.status) {
            case "PendingDeployment": {
                return buildPhases("pending", "pending", "pending");
            }
            case "DeployingVector": {
                return buildPhases("active", "pending", "pending");
            }
            case "MigratingVector": {
                return buildPhases("done", "active", "pending");
            }
            case "ActivatingVector": {
                return buildPhases("done", "done", "active");
            }
            case "Ready": {
                return buildPhases("done", "done", "done");
            }
            case "Failed": {
                // Without a specific failed-phase signal from the API we
                // infer position: if there is already an active version
                // we made it past deploy+migrate, so Activate failed;
                // otherwise Deploy is what tripped.
                return active
                    ? buildPhases("done", "done", "failed")
                    : buildPhases("failed", "pending", "pending");
            }
            default: {
                return buildPhases("pending", "pending", "pending");
            }
        }
    });

    const statusLabel = (version: StageVersion): string =>
        stageStatuses[version.status].label;
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- The route helper supplies the resolved details URL. -->
<a {href}
    class="stage-card box-border relative flex h-64 w-full flex-col gap-3 overflow-hidden rounded-[var(--card-radius)] border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 pt-[calc(var(--space-4)+3px)] text-[var(--text-primary)] no-underline shadow-[var(--shadow-sm)] transition-[transform,box-shadow,border-color] duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)] focus-visible:outline-2 focus-visible:outline-offset-3 focus-visible:outline-[var(--border-focus)]"
    class:stage-card--selected={selected}
    data-status={statusRole}
    style:--stage-accent={stageAccent}
    aria-label={`View details for stage ${stage.name} in ${landscapeName}${category ? `, ${category}` : ""}`}
    aria-describedby={statusId}
>
    <!-- 3 px status stripe pinned to the top edge; colour driven by --stage-accent. -->
    <span
        class="pointer-events-none absolute inset-x-0 top-0 h-[3px]"
        style:background="var(--stage-accent)"
        aria-hidden="true"
    ></span>

    <header class="flex items-baseline gap-3">
        <h4
            class="m-0 truncate text-[length:var(--text-body)] font-bold"
            title={stage.name}
        >
            {stage.name}
        </h4>
        {#if targetIsActive}
            <span
                class="ml-auto inline-flex shrink-0 items-center gap-1.5 text-[length:var(--text-meta)] font-semibold text-[var(--status-healthy-fg)]"
            >
                <span
                    class="size-2 shrink-0 rounded-full bg-[var(--status-healthy-solid)]"
                    aria-hidden="true"
                ></span>
                live
            </span>
        {/if}
    </header>

    <div class="flex min-h-14 flex-col gap-1">
        <span class="text-[length:var(--text-meta)] text-[var(--text-tertiary)]"
            >Target vector</span
        >
        {#if target}
            <span
                class="line-clamp-2 font-[family-name:var(--font-mono)] text-[length:var(--text-sm)] leading-normal [overflow-wrap:anywhere]"
                title={target.vector}>{target.vector}</span
            >
        {:else}
            <span
                class="text-[length:var(--text-sm)] text-[var(--text-secondary)]"
                >No target version yet</span
            >
        {/if}
    </div>

    <div class="flex flex-col gap-1">
        <span class="text-[length:var(--text-meta)] text-[var(--text-tertiary)]"
            >Active version</span
        >
        <span
            class="truncate text-[length:var(--text-sm)] text-[var(--text-secondary)]"
            title={active?.id}
        >
            {targetIsActive
                ? "Matches target"
                : (active?.id ?? "Nothing active yet")}
        </span>
    </div>

    <footer
        class="mt-auto border-t border-[var(--border-subtle)] pt-3"
        id={statusId}
    >
        <StagePhase
            {phases}
            ariaLabel={target ? statusLabel(target) : "No target"}
        />
    </footer>
</a>

<style>
    /* Selected state — a 3 px halo in the card's current status colour,
       matching the design-system `.stage-card--selected` rule. Uses
       color-mix to soften the ring to ~32 % opacity while keeping the
       border fully saturated. */
    .stage-card--selected {
        border-color: var(--stage-accent);
        box-shadow:
            0 0 0 3px
                color-mix(in srgb, var(--stage-accent) 32%, transparent),
            var(--card-rim-glow);
    }

    /* Hovering over the still-selected card keeps the ring; drop the
       default translate so the halo stays crisp. */
    .stage-card--selected:hover {
        transform: none;
        box-shadow:
            0 0 0 3px
                color-mix(in srgb, var(--stage-accent) 32%, transparent),
            var(--shadow-md);
    }

    /* Default (non-selected) hover uses the design-system border-focus
       colour so keyboard focus and mouse hover feel connected. */
    .stage-card:not(.stage-card--selected):hover {
        border-color: var(--border-focus);
    }

    /* Users who opt out of animations get an instant state change. */
    @media (prefers-reduced-motion: reduce) {
        .stage-card {
            transition: none;
        }
        .stage-card:hover {
            transform: none;
        }
    }
</style>
