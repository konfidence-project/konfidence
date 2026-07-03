<script lang="ts">
    import "@ui5/webcomponents/dist/Card.js";
    import "@ui5/webcomponents/dist/CardHeader.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents/dist/Tag.js";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Popover.js";
    import "@ui5/webcomponents/dist/List.js";
    import "@ui5/webcomponents/dist/ListItemStandard.js";
    import "@ui5/webcomponents-icons/dist/upstacked-chart.js";
    import "@ui5/webcomponents-icons/dist/overflow.js";

    import {
        getChips,
        getPhases,
        getStageStatusLabel,
        splitVector,
    } from "$lib/stage-view.js";
    import type { Stage } from "$lib/stages.js";

    type StageChip = import("$lib/stage-view.js").StageChip;
    type StageHealth = import("$lib/stage-view.js").StageHealth;
    type StagePhaseState = import("$lib/stage-view.js").StagePhaseState;

    type TagDesign = "Critical" | "Information" | "Negative" | "Neutral" | "Positive";

    const STATUS_DESIGN: Record<StageHealth, TagDesign> = {
        deploying: "Information",
        error: "Negative",
        healthy: "Positive",
        warning: "Critical",
    };

    const PHASE_DESIGN: Record<StagePhaseState, TagDesign> = {
        cur: "Information",
        done: "Positive",
        err: "Negative",
        idle: "Neutral",
    };

    const CHIP_DESIGN: Record<NonNullable<StageChip["tone"]>, TagDesign> = {
        "": "Neutral",
        alert: "Negative",
        info: "Information",
        warn: "Critical",
    };

    const { stage } = $props<{ stage: Stage }>();

    const status = $derived(getStageStatusLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.spec.vector));

    const menuBtn = $state<HTMLElement>();
    const popover = $state<HTMLElement & { open?: boolean }>();

    const btnId = $derived(`stage-card-fiori-menu-${stage.metadata.name}`);

    const shouldPulseStatus = $derived(
        status.tone === "deploying" || status.tone === "error",
    );

    const tagDesign = (tone: StageHealth) => STATUS_DESIGN[tone];
    const phaseTagDesign = (state: StagePhaseState) => PHASE_DESIGN[state];
    const chipDesign = (tone: StageChip["tone"]) => CHIP_DESIGN[tone ?? ""];

    const openMenu = () => {
        if (popover && menuBtn) {
            popover.setAttribute("opener", btnId);
            popover.open = true;
        }
    };

    const closeMenu = () => {
        if (popover) {popover.open = false;}
    };

    const onMenuKey = (event: KeyboardEvent) => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            openMenu();
        }
    };
</script>

<ui5-card class="stage-fiori-card">
    <ui5-card-header
        slot="header"
        title-text={stage.metadata.name}
        subtitle-text={stage.metadata.namespace}
        interactive
    >
        <ui5-icon slot="avatar" name="upstacked-chart"></ui5-icon>
        <div slot="action" class="header-action">
            <ui5-tag design={tagDesign(status.tone)} hide-state-icon>
                <span class="pill-inner" class:pulse={shouldPulseStatus}>
                    <span class="dot" class:pulse={shouldPulseStatus}></span>
                    {status.label}
                </span>
            </ui5-tag>
            <ui5-button
                bind:this={menuBtn}
                id={btnId}
                design="Transparent"
                icon="overflow"
                tooltip="More actions"
                role="button"
                tabindex="0"
                aria-label="Stage actions"
                onclick={openMenu}
                onkeydown={onMenuKey}
            ></ui5-button>
        </div>
    </ui5-card-header>

    <div class="card-body">
        <div class="row">
            <span class="row-label">Vector</span>
            <span class="row-value">
                <code>{vector.version}</code>
                {#if vector.hash}
                    <code class="hash">{vector.hash}</code>
                {/if}
            </span>
        </div>

        <div class="row">
            <span class="row-label">Phases</span>
            <span class="phase-row">
                {#each phases as phase (phase.id)}
                    <ui5-tag
                        design={phaseTagDesign(phase.state)}
                        hide-state-icon
                        title={phase.reason
                            ? `${phase.label}: ${phase.reason}${phase.message ? ` — ${phase.message}` : ""}`
                            : phase.label}
                    >
                        {phase.label}
                    </ui5-tag>
                {/each}
            </span>
        </div>

        {#if chips.length > 0}
            <div class="row">
                <span class="row-label">Details</span>
                <span class="chip-row">
                    {#each chips as chip, i (`${chip.label}-${i}`)}
                        <ui5-tag design={chipDesign(chip.tone)} hide-state-icon>
                            <strong>{chip.value}</strong>&nbsp;{chip.label}
                        </ui5-tag>
                    {/each}
                </span>
            </div>
        {/if}
    </div>
</ui5-card>

<ui5-popover
    bind:this={popover}
    placement="Bottom"
    horizontal-align="End"
    accessible-name="Stage actions"
>
    <ui5-list
        separators="None"
        accessible-name="Stage actions"
        onitem-click={closeMenu}
    >
        <ui5-li icon="copy" type="Active">Copy stage name</ui5-li>
        <ui5-li icon="document-text" type="Active">View YAML</ui5-li>
        <ui5-li icon="log" type="Active">Open logs</ui5-li>
    </ui5-list>
</ui5-popover>

<style>
    .stage-fiori-card {
        min-width: 0;
        width: 100%;
        /* Compact density — shrink UI5 font/spacing tokens locally. */
        --sapFontSize: 0.75rem;
        --sapContent_ElementHeight_Regular: 1.625rem;
        --sapContent_ContainerPadding_Regular: 0.5rem;
        --_ui5-v2_23_2-card-header-padding: 0.5rem 0.75rem;
    }

    .header-action {
        display: inline-flex;
        align-items: center;
        gap: 0.25rem;
    }

    .card-body {
        display: grid;
        gap: 0.5rem;
        padding: 0.75rem;
    }

    .row {
        display: grid;
        grid-template-columns: 4.25rem 1fr;
        align-items: center;
        gap: 0.5rem;
        min-height: 1.5rem;
    }

    .row-label {
        color: var(--sapContent_LabelColor);
        font-size: 0.6875rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .row-value {
        display: inline-flex;
        gap: 0.375rem;
        align-items: center;
        min-width: 0;
        overflow-wrap: anywhere;
    }

    code {
        padding: 0.0625rem 0.375rem;
        border: 1px solid var(--sapList_BorderColor);
        border-radius: 0.25rem;
        background: var(--sapList_Background);
        font-family: var(--sapFontMonospaceFamily, "SF Mono", Menlo, monospace);
        font-size: 0.75rem;
    }
    code.hash {
        color: var(--sapContent_LabelColor);
    }

    .phase-row,
    .chip-row {
        display: flex;
        flex-wrap: wrap;
        gap: 0.25rem;
    }

    /* ── Pulsing status dot (deploying / error only) ────────── */
    .pill-inner {
        display: inline-flex;
        align-items: center;
        gap: 0.3125rem;
        line-height: 1;
    }
    .dot {
        width: 0.375rem;
        height: 0.375rem;
        border-radius: 50%;
        background: currentColor;
    }
    .dot.pulse {
        animation: fiori-status-pulse 0.8s ease-in-out infinite alternate;
    }
    @keyframes fiori-status-pulse {
        from {
            opacity: 1;
        }
        to {
            opacity: 0.25;
        }
    }
</style>
