<script lang="ts">
    import {
        getChips,
        getLandscapeLabel,
        getPhases,
        getStageStatusLabel,
        splitVector,
    } from "$lib/stage-view.js";
    import type { Stage } from "$lib/stages.js";
    import StageStatusPill from "$lib/components/StageStatusPill.svelte";

    const { stage, selected = false } = $props<{
        stage: Stage;
        selected?: boolean;
    }>();

    const status = $derived(getStageStatusLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.spec.vector));
    const landscape = $derived(getLandscapeLabel(stage));

    let menuOpen = $state(false);
    let cardEl = $state<HTMLElement>();

    const toggleMenu = (event: MouseEvent) => {
        event.stopPropagation();
        menuOpen = !menuOpen;
    };

    const closeMenu = () => {
        menuOpen = false;
    };

    const copyName = async () => {
        try {
            await globalThis.navigator?.clipboard?.writeText(
                stage.metadata.name,
            );
        } catch {
            // Clipboard not available (e.g. SSR/tests) — silently ignore.
        }
        closeMenu();
    };

    const phaseTitle = (label: string, reason?: string, message?: string) => {
        if (reason && message) {
            return `${label}: ${reason} — ${message}`;
        }
        if (reason) {
            return `${label}: ${reason}`;
        }
        if (message) {
            return `${label}: ${message}`;
        }
        return label;
    };
</script>

<svelte:window on:click={closeMenu} />

<article
    bind:this={cardEl}
    class="stage"
    class:selected
    class:sel-err={selected && status.tone === "error"}
    data-health={status.tone}
    data-testid="stage-card"
    aria-label={`Stage ${stage.metadata.name}`}
>
    <div class="stripe {status.tone}"></div>

    <div class="st-h">
        <div class="st-r1">
            <span class="st-nm">{stage.metadata.name}</span>
            <StageStatusPill
                status={status.tone}
                label={status.label}
                pulse={status.tone === "deploying" || status.tone === "error"}
            />
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
                    <span class={["sc", chip.tone ?? ""]}>
                        <strong>{chip.value}</strong>
                        {chip.label}
                    </span>
                {/each}
            </div>
        {/if}
    </div>

    <button
        type="button"
        class="st-menu"
        class:open={menuOpen}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        aria-label="Stage actions"
        onclick={toggleMenu}
    >
        ⋮
    </button>

    {#if menuOpen}
        <div class="st-menu-pop" role="menu">
            <button
                type="button"
                role="menuitem"
                class="st-menu-item"
                onclick={copyName}
            >
                Copy stage name
            </button>
            <button
                type="button"
                role="menuitem"
                class="st-menu-item"
                onclick={closeMenu}
            >
                View YAML
            </button>
            <button
                type="button"
                role="menuitem"
                class="st-menu-item"
                onclick={closeMenu}
            >
                Open logs
            </button>
        </div>
    {/if}
</article>

<style>
    /* Local design tokens — pulled from the mockup so the tile keeps its
       look even when the surrounding Fiori theme changes. */
    .stage {
        --stage-surface: var(--sapTile_Background, #ffffff);
        --stage-border: var(--sapTile_BorderColor, #e4e4e2);
        --stage-border-strong: #d0d0cc;
        --stage-bg: var(--sapBackgroundColor, #f5f5f3);
        --stage-text: var(--sapTextColor, #111111);
        --stage-text-2: var(--sapContent_LabelColor, #6b6b68);
        --stage-text-3: #a8a8a4;
        --stage-radius: 10px;
        --stage-green: var(--sapPositiveElementColor, #2e7d32);
        --stage-green-bg: var(--sapSuccessBackground, #e8f5e9);
        --stage-blue: var(--sapInformativeElementColor, #1565c0);
        --stage-blue-bg: var(--sapInformationBackground, #e3f2fd);
        --stage-yellow: var(--sapCriticalElementColor, #f57f17);
        --stage-yellow-bg: var(--sapWarningBackground, #fff8e1);
        --stage-red: var(--sapNegativeElementColor, #c62828);
        --stage-red-bg: var(--sapErrorBackground, #ffebee);
        --stage-shadow-lg: 0 8px 32px rgba(0, 0, 0, 0.1);

        position: relative;
        width: 100%;
        box-sizing: border-box;
        background: var(--stage-surface);
        border: 1.5px solid var(--stage-border);
        border-radius: var(--stage-radius);
        cursor: pointer;
        overflow: hidden;
        transition:
            transform 0.25s cubic-bezier(0.16, 1, 0.3, 1),
            box-shadow 0.25s cubic-bezier(0.16, 1, 0.3, 1),
            border-color 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        color: var(--stage-text);
        font-family:
            -apple-system, "Helvetica Neue", "Segoe UI", Arial, sans-serif;
    }

    .stage:hover {
        box-shadow: var(--stage-shadow-lg);
        transform: translateY(-2px);
    }

    .stage.selected {
        border-color: var(--stage-blue);
        box-shadow:
            0 0 0 2.5px var(--stage-blue),
            0 4px 16px rgba(21, 101, 192, 0.15);
        transform: translateY(-1px);
    }

    .stage.sel-err {
        border-color: var(--stage-red);
        box-shadow:
            0 0 0 2.5px var(--stage-red),
            0 4px 16px rgba(198, 40, 40, 0.15);
    }

    /* ── Stripe ────────────────────────────────────────────────── */
    .stripe {
        height: 4px;
        width: 100%;
    }
    .stripe.healthy {
        background: var(--stage-green);
    }
    .stripe.deploying {
        background: var(--stage-blue);
        animation: stripe-pulse 2s ease-in-out infinite;
    }
    .stripe.warning {
        background: var(--stage-yellow);
    }
    .stripe.error {
        background: var(--stage-red);
        animation: stripe-pulse 1.2s ease-in-out infinite;
    }

    @keyframes stripe-pulse {
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
        padding: 12px 34px 8px 14px;
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
        color: var(--stage-text);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .st-ls {
        font-size: 11px;
        color: var(--stage-text-3);
        font-family: "SF Mono", "Menlo", "Consolas", ui-monospace, monospace;
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
        font-family: "SF Mono", "Menlo", "Consolas", ui-monospace, monospace;
        font-weight: 700;
        font-size: 14px;
        color: var(--stage-text);
    }
    .st-vh {
        font-family: "SF Mono", "Menlo", "Consolas", ui-monospace, monospace;
        font-size: 11px;
        color: var(--stage-text-3);
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
        background: var(--stage-border);
    }
    .st-ph.done {
        background: var(--stage-green);
        opacity: 0.55;
    }
    .st-ph.cur {
        background: var(--stage-blue);
        animation: stripe-pulse 2s ease-in-out infinite;
    }
    .st-ph.err {
        background: var(--stage-red);
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
        color: var(--stage-text-3);
        text-align: center;
    }
    .st-pl.done {
        color: var(--stage-green);
    }
    .st-pl.cur {
        color: var(--stage-blue);
        font-weight: 700;
    }
    .st-pl.err {
        color: var(--stage-red);
        font-weight: 700;
    }

    .st-chips {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
        margin-top: 6px;
    }
    .sc {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border-radius: 5px;
        font-size: 12px;
        background: var(--stage-bg);
        color: var(--stage-text-2);
        border: 1px solid var(--stage-border);
    }
    .sc strong {
        font-weight: 700;
        color: var(--stage-text);
    }
    .sc.alert {
        background: var(--stage-red-bg);
        border-color: var(--stage-red);
        color: var(--stage-red);
        font-weight: 600;
    }
    .sc.info {
        background: var(--stage-blue-bg);
        border-color: var(--stage-blue);
        color: var(--stage-blue);
        font-weight: 600;
    }
    .sc.warn {
        background: var(--stage-yellow-bg);
        border-color: var(--stage-yellow);
        color: var(--stage-yellow);
        font-weight: 600;
    }

    /* ── Kebab menu ────────────────────────────────────────────── */
    .st-menu {
        position: absolute;
        top: 8px;
        right: 8px;
        width: 22px;
        height: 22px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
        background: transparent;
        border: none;
        cursor: pointer;
        font-size: 16px;
        line-height: 1;
        color: var(--stage-text-3);
        transition:
            background 0.15s,
            color 0.15s;
        padding: 0;
    }
    .st-menu:hover {
        background: var(--stage-bg);
        color: var(--stage-text);
    }

    .st-menu-pop {
        position: absolute;
        top: 34px;
        right: 8px;
        min-width: 160px;
        background: var(--stage-surface);
        border: 1px solid var(--stage-border);
        border-radius: 6px;
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
        padding: 4px 0;
        z-index: 5;
    }
    .st-menu-item {
        display: block;
        width: 100%;
        text-align: left;
        padding: 7px 12px;
        background: transparent;
        border: none;
        font: inherit;
        font-size: 13px;
        color: var(--stage-text);
        cursor: pointer;
    }
    .st-menu-item:hover,
    .st-menu-item:focus-visible {
        background: var(--stage-bg);
        outline: none;
    }
</style>
