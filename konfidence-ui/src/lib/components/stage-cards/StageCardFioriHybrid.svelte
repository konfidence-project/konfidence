<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents/dist/Tag.js";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Popover.js";
    import "@ui5/webcomponents/dist/List.js";
    import "@ui5/webcomponents/dist/ListItemStandard.js";
    import "@ui5/webcomponents-icons/dist/overflow.js";

    import type { Stage } from "$lib/stages.js";
    import {
        getChips,
        getLandscapeLabel,
        getPhases,
        getStageStatusLabel,
        splitVector,
    } from "$lib/stage-view.js";

    const { stage, selected = false } = $props<{
        stage: Stage;
        selected?: boolean;
    }>();

    const status = $derived(getStageStatusLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.spec.vector));
    const landscape = $derived(getLandscapeLabel(stage));

    let menuBtn = $state<HTMLElement | null>(null);
    let popover = $state<(HTMLElement & { open?: boolean }) | null>(null);

    const btnId = $derived(
        `stage-card-hybrid-menu-${stage.metadata.name}`,
    );

    const statusDesign = $derived(
        status.tone === "healthy"
            ? "Positive"
            : status.tone === "deploying"
              ? "Information"
              : status.tone === "warning"
                ? "Critical"
                : status.tone === "error"
                  ? "Negative"
                  : "Neutral",
    );

    const chipDesign = (tone: string | undefined) =>
        tone === "alert"
            ? "Negative"
            : tone === "info"
              ? "Information"
              : tone === "warn"
                ? "Critical"
                : "Neutral";

    const openMenu = (event: Event) => {
        event.stopPropagation();
        if (popover && menuBtn) {
            popover.setAttribute("opener", btnId);
            popover.open = true;
        }
    };

    const closeMenu = () => {
        if (popover) popover.open = false;
    };

    const onMenuKey = (event: KeyboardEvent) => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            openMenu(event);
        }
    };

    const phaseTitle = (
        label: string,
        reason?: string,
        message?: string,
    ) => {
        if (reason && message) return `${label}: ${reason} — ${message}`;
        if (reason) return `${label}: ${reason}`;
        if (message) return `${label}: ${message}`;
        return label;
    };
</script>

<article
    class="stage"
    class:selected
    class:sel-err={selected && status.tone === "error"}
    data-health={status.tone}
    aria-label={`Stage ${stage.metadata.name}`}
>
    <div class="stripe {status.tone}" aria-hidden="true"></div>

    <div class="st-h">
        <div class="st-r1">
            <span class="st-nm" title={stage.metadata.name}>
                {stage.metadata.name}
            </span>
            <ui5-tag design={statusDesign} hide-state-icon>
                <span class="pill-inner">
                    <span
                        class="dot"
                        class:pulse={status.tone === "deploying" ||
                            status.tone === "error"}
                    ></span>
                    {status.label}
                </span>
            </ui5-tag>
        </div>
        <div class="st-ls">{landscape}</div>
        <div class="st-vec">
            <span class="st-vn">{vector.version}</span>
            {#if vector.hash}
                <span class="st-vh">{vector.hash}</span>
            {/if}
        </div>
    </div>

    <div class="st-body">
        <div class="st-phs">
            {#each phases as phase (phase.id)}
                <div
                    class={["st-ph", phase.state]}
                    title={phaseTitle(phase.label, phase.reason, phase.message)}
                ></div>
            {/each}
        </div>
        <div class="st-phl">
            {#each phases as phase (`${phase.id}-l`)}
                <div class={["st-pl", phase.state]}>{phase.label}</div>
            {/each}
        </div>

        {#if chips.length > 0}
            <div class="st-chips">
                {#each chips as chip, i (`${chip.label}-${i}`)}
                    <ui5-tag design={chipDesign(chip.tone)} hide-state-icon>
                        <strong>{chip.value}</strong>&nbsp;{chip.label}
                    </ui5-tag>
                {/each}
            </div>
        {/if}
    </div>

    <ui5-button
        bind:this={menuBtn}
        id={btnId}
        class="st-menu"
        design="Transparent"
        icon="overflow"
        tooltip="More actions"
        role="button"
        tabindex="0"
        aria-label="Stage actions"
        onclick={openMenu}
        onkeydown={onMenuKey}
    ></ui5-button>
</article>

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
    .stage {
        position: relative;
        width: 270px;
        background: var(--sapTile_Background, #fff);
        border: 1.5px solid var(--sapTile_BorderColor, #e4e4e2);
        border-radius: 10px;
        overflow: hidden;
        cursor: pointer;
        transition:
            transform 0.25s cubic-bezier(0.16, 1, 0.3, 1),
            box-shadow 0.25s cubic-bezier(0.16, 1, 0.3, 1),
            border-color 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        color: var(--sapTextColor, #111);
        font-family: var(
            --sapFontFamily,
            -apple-system,
            "Helvetica Neue",
            Arial,
            sans-serif
        );
    }
    .stage:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
    }
    .stage.selected {
        border-color: var(--sapInformativeElementColor, #1565c0);
        box-shadow:
            0 0 0 2.5px var(--sapInformativeElementColor, #1565c0),
            0 4px 16px rgba(21, 101, 192, 0.15);
        transform: translateY(-1px);
    }
    .stage.sel-err {
        border-color: var(--sapNegativeElementColor, #c62828);
        box-shadow:
            0 0 0 2.5px var(--sapNegativeElementColor, #c62828),
            0 4px 16px rgba(198, 40, 40, 0.15);
    }

    /* ── Health stripe ───────────────────────────────────────── */
    .stripe {
        height: 4px;
        width: 100%;
    }
    .stripe.healthy {
        background: var(--sapPositiveElementColor, #2e7d32);
    }
    .stripe.deploying {
        background: var(--sapInformativeElementColor, #1565c0);
        animation: hs-pulse 2s ease-in-out infinite;
    }
    .stripe.warning {
        background: var(--sapCriticalElementColor, #f57f17);
    }
    .stripe.error {
        background: var(--sapNegativeElementColor, #c62828);
        animation: hs-pulse 1.2s ease-in-out infinite;
    }
    @keyframes hs-pulse {
        0%,
        100% {
            opacity: 0.5;
        }
        50% {
            opacity: 1;
        }
    }

    /* ── Header ────────────────────────────────────────────────── */
    .st-h {
        /* Right padding reserves space for the absolutely-positioned kebab so
           the "Live" pill can never sit under it. */
        padding: 12px 40px 8px 14px;
    }
    .st-r1 {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 6px;
    }
    .st-nm {
        font-weight: 700;
        letter-spacing: -0.01em;
        font-size: 15px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .st-ls {
        font-size: 11px;
        color: var(--sapContent_LabelColor, #6b6b68);
        font-family: var(--sapFontMonospaceFamily, "SF Mono", Menlo, monospace);
        margin-top: 2px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .st-vec {
        display: flex;
        align-items: baseline;
        gap: 6px;
        margin-top: 6px;
    }
    .st-vn {
        font-family: var(--sapFontMonospaceFamily, "SF Mono", Menlo, monospace);
        font-weight: 700;
        font-size: 14px;
    }
    .st-vh {
        font-family: var(--sapFontMonospaceFamily, "SF Mono", Menlo, monospace);
        font-size: 11px;
        color: var(--sapContent_LabelColor, #6b6b68);
    }

    /* ── Status pill dot inside ui5-tag ─────────────────────── */
    .pill-inner {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        line-height: 1;
    }
    .dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: currentColor;
    }
    .dot.pulse {
        animation: pill-bdot 0.8s ease-in-out infinite alternate;
    }
    @keyframes pill-bdot {
        from {
            opacity: 1;
        }
        to {
            opacity: 0.25;
        }
    }

    /* ── Body ──────────────────────────────────────────────────── */
    .st-body {
        padding: 0 14px 12px;
    }
    .st-phs {
        display: flex;
        gap: 3px;
        margin: 6px 0 2px;
    }
    .st-ph {
        flex: 1;
        height: 4px;
        border-radius: 4px;
        background: var(--sapField_BorderColor, #e4e4e2);
    }
    .st-ph.done {
        background: var(--sapPositiveElementColor, #2e7d32);
        opacity: 0.55;
    }
    .st-ph.cur {
        background: var(--sapInformativeElementColor, #1565c0);
        animation: hs-pulse 2s ease-in-out infinite;
    }
    .st-ph.err {
        background: var(--sapNegativeElementColor, #c62828);
        opacity: 0.65;
    }

    .st-phl {
        display: flex;
        gap: 3px;
        margin-bottom: 6px;
    }
    .st-pl {
        flex: 1;
        font-size: 9px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--sapContent_LabelColor, #6b6b68);
        text-align: center;
    }
    .st-pl.done {
        color: var(--sapPositiveElementColor, #2e7d32);
    }
    .st-pl.cur {
        color: var(--sapInformativeElementColor, #1565c0);
        font-weight: 700;
    }
    .st-pl.err {
        color: var(--sapNegativeElementColor, #c62828);
        font-weight: 700;
    }

    .st-chips {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
        margin-top: 6px;
    }

    /* ── Kebab ─────────────────────────────────────────────────── */
    .st-menu {
        position: absolute;
        top: 6px;
        right: 6px;
        z-index: 3;
    }
</style>
