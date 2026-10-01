<script lang="ts">
    import { StageCard, StagePhase } from "@konfidence/design-system/components";
    import type { StagePhaseItem } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    // Realistic phase compositions mirroring the fixtures shipped with the
    // design-system (see StagePhase/__tests__ + StageCard/__tests__).
    const PHASES_ACTIVE: StagePhaseItem[] = [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "active" },
        { label: "Activate", state: "pending" },
    ];
    const PHASES_DONE: StagePhaseItem[] = [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "done" },
        { label: "Activate", state: "done" },
    ];
    const PHASES_FAILED: StagePhaseItem[] = [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "failed" },
        { label: "Activate", state: "pending" },
    ];
    const PHASES_PENDING: StagePhaseItem[] = [
        { label: "Deploy", state: "pending" },
        { label: "Migrate", state: "pending" },
        { label: "Activate", state: "pending" },
    ];
    const PHASES_DEPLOY_ACTIVE: StagePhaseItem[] = [
        { label: "Deploy", state: "active" },
        { label: "Migrate", state: "pending" },
        { label: "Activate", state: "pending" },
    ];
</script>

<Section
    id="pipeline"
    title="Pipeline patterns"
    subtitle="Real <StageCard> and <StagePhase> from @konfidence/design-system, at every state the dashboard renders. Higher-order composition primitives (promotion, timeline) still on the roadmap."
    status="partial"
>
    <div class="stack">
        <Sample
            title="StagePhase — sizes"
            description="Density variants: compact (in-card, no labels), default (7 px bar + labels), lg (10 px bar)."
        >
            <div class="phase-grid">
                <div>
                    <p class="phase-caption">compact</p>
                    <StagePhase phases={PHASES_ACTIVE} size="compact" ariaLabel="compact deploy" />
                </div>
                <div>
                    <p class="phase-caption">default</p>
                    <StagePhase phases={PHASES_ACTIVE} ariaLabel="default deploy" />
                </div>
                <div>
                    <p class="phase-caption">lg</p>
                    <StagePhase phases={PHASES_ACTIVE} size="lg" ariaLabel="large deploy" />
                </div>
            </div>
        </Sample>

        <Sample
            title="StagePhase — states"
            description="all-pending → deploy-active → migrate-active → all-done → failed."
        >
            <div class="phase-grid">
                <div>
                    <p class="phase-caption">all pending</p>
                    <StagePhase phases={PHASES_PENDING} ariaLabel="all pending" />
                </div>
                <div>
                    <p class="phase-caption">deploy active</p>
                    <StagePhase phases={PHASES_DEPLOY_ACTIVE} ariaLabel="deploy active" />
                </div>
                <div>
                    <p class="phase-caption">migrate active</p>
                    <StagePhase phases={PHASES_ACTIVE} ariaLabel="migrate active" />
                </div>
                <div>
                    <p class="phase-caption">all done</p>
                    <StagePhase phases={PHASES_DONE} ariaLabel="all done" />
                </div>
                <div>
                    <p class="phase-caption">failed</p>
                    <StagePhase phases={PHASES_FAILED} ariaLabel="failed" />
                </div>
            </div>
        </Sample>

        <Sample
            title="StageCard — grid"
            description="Data-block card used by the landscape overview. Colour rail encodes status; the mini phase strip inside repeats the current pipeline state."
        >
            <div class="stage-grid">
                <StageCard
                    title="prod"
                    href="#pipeline"
                    ariaLabel="prod — healthy, live"
                    statusRole="healthy"
                    phases={PHASES_DONE}
                    phaseAriaLabel="prod pipeline"
                    targetVector="registry.konfidence.dev/checkout@v2.4.1"
                    activeVersionText="registry.konfidence.dev/checkout@v2.4.1"
                    live
                />
                <StageCard
                    title="staging"
                    href="#pipeline"
                    ariaLabel="staging — deploying"
                    statusRole="deploying"
                    phases={PHASES_DEPLOY_ACTIVE}
                    phaseAriaLabel="staging pipeline"
                    targetVector="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                    activeVersionText="registry.konfidence.dev/checkout@v2.4.1"
                />
                <StageCard
                    title="canary"
                    href="#pipeline"
                    ariaLabel="canary — migrating"
                    statusRole="deploying"
                    phases={PHASES_ACTIVE}
                    phaseAriaLabel="canary pipeline"
                    targetVector="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                    activeVersionText="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                    selected
                />
                <StageCard
                    title="qa"
                    href="#pipeline"
                    ariaLabel="qa — failed"
                    statusRole="error"
                    phases={PHASES_FAILED}
                    phaseAriaLabel="qa pipeline"
                    targetVector="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                    activeVersionText="registry.konfidence.dev/checkout@v2.4.1"
                />
                <StageCard
                    title="dev"
                    href="#pipeline"
                    ariaLabel="dev — pending"
                    statusRole="neutral"
                    phases={PHASES_PENDING}
                    phaseAriaLabel="dev pipeline"
                    activeVersionText="No active version yet"
                />
                <StageCard
                    title="preview"
                    href="#pipeline"
                    ariaLabel="preview — warning"
                    statusRole="warning"
                    phases={PHASES_DONE}
                    phaseAriaLabel="preview pipeline"
                    targetVector="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                    activeVersionText="registry.konfidence.dev/checkout@v2.5.0-rc.1"
                />
            </div>
        </Sample>

        <NotYetImplemented
            note="Cross-stage composition (`Promotion from→to`, `PipelineTimeline`) requires an orchestrating layer on top of these primitives; still pending."
            planned={[
                "<Promotion from to>",
                "<PipelineTimeline>",
            ]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .phase-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 16px;
        width: 100%;
    }

    .phase-caption {
        margin: 0 0 6px;
        font-size: var(--text-meta);
        color: var(--text-tertiary, var(--text-secondary));
        text-transform: uppercase;
        letter-spacing: 0.04em;
    }

    .stage-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
        gap: 16px;
        width: 100%;
    }
</style>
