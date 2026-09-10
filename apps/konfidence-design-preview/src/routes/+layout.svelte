<script lang="ts">
    import "../app.css";

    import type { Snippet } from "svelte";
    import { Brandbar, SkipLink } from "@konfidence/design-system/components";
    import { resolve } from "$app/paths";
    import { themeStore } from "$lib/theme";

    interface Props {
        children?: Snippet;
    }

    let { children }: Props = $props();

    const MAIN_ID = "preview-main";

    const MODE_LABEL: Record<"light" | "dark" | "system", string> = {
        dark: "Dark",
        light: "Light",
        system: "System",
    };

    const nextMode = $derived(themeStore.mode === "dark" ? "light" : "dark");
    const modeLabel = $derived(MODE_LABEL[themeStore.mode]);
</script>

<SkipLink target={MAIN_ID}>Skip to main content</SkipLink>

<div class="preview-shell">
    <Brandbar />
    <header class="preview-topbar">
        <a href={resolve("/")} class="preview-topbar__brand">
            <strong>Konfidence</strong>
            <span class="preview-topbar__dim">Design System</span>
        </a>
        <nav class="preview-topbar__nav" aria-label="Design system">
            <a href={resolve("/")}>Overview</a>
            <a href={resolve("/design-system")}>Style guide</a>
        </nav>
        <div class="preview-topbar__actions">
            <button
                type="button"
                class="preview-toggle"
                onclick={() => themeStore.setMode(nextMode)}
                aria-label="Toggle color mode (currently {modeLabel})"
                title="Toggle color mode"
            >
                <span class="preview-toggle__label">Mode</span>
                <span class="preview-toggle__value">{modeLabel}</span>
            </button>
        </div>
    </header>

    <main id={MAIN_ID} class="preview-main">
        {@render children?.()}
    </main>

    <footer class="preview-footer">
        <span>Konfidence Design Preview · reads live from <code>@konfidence/design-system</code>.</span>
    </footer>
</div>

<style>
    .preview-shell {
        min-height: 100vh;
        display: flex;
        flex-direction: column;
        background: var(--surface-canvas, var(--surface-default));
        color: var(--text-primary);
    }

    .preview-topbar {
        position: sticky;
        top: 0;
        z-index: 20;
        display: flex;
        align-items: center;
        gap: 20px;
        padding: 10px 20px;
        border-bottom: 1px solid var(--border-subtle);
        background: color-mix(in oklab, var(--surface-default) 92%, transparent);
        backdrop-filter: blur(6px);
    }

    .preview-topbar__brand {
        display: inline-flex;
        align-items: baseline;
        gap: 6px;
        color: var(--text-primary);
        text-decoration: none;
        font-size: var(--text-body);
    }

    .preview-topbar__brand strong {
        font-weight: var(--weight-display, 600);
        letter-spacing: -0.02em;
    }

    .preview-topbar__dim {
        color: var(--text-tertiary, var(--text-secondary));
        font-size: var(--text-sm);
    }

    .preview-topbar__nav {
        display: flex;
        gap: 4px;
        margin-left: 12px;
    }

    .preview-topbar__nav a {
        padding: 6px 10px;
        border-radius: 6px;
        color: var(--text-secondary);
        text-decoration: none;
        font-size: var(--text-sm);
    }

    .preview-topbar__nav a:hover {
        background: var(--surface-subtle);
        color: var(--text-primary);
    }

    .preview-topbar__actions {
        margin-left: auto;
        display: inline-flex;
        align-items: center;
        gap: 8px;
    }

    .preview-toggle {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 6px 10px;
        border-radius: 999px;
        border: 1px solid var(--border-subtle);
        background: var(--surface-default);
        color: var(--text-secondary);
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        cursor: pointer;
        transition: background var(--motion-fast) var(--ease);
    }

    .preview-toggle:hover {
        background: var(--surface-subtle);
        color: var(--text-primary);
    }

    .preview-toggle__label {
        text-transform: uppercase;
        letter-spacing: 0.04em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .preview-toggle__value {
        color: var(--text-primary);
        text-transform: capitalize;
    }

    .preview-main {
        flex: 1;
        width: 100%;
        max-width: 1200px;
        margin: 0 auto;
        padding: 32px 24px 64px;
    }

    .preview-footer {
        border-top: 1px solid var(--border-subtle);
        padding: 16px 20px;
        color: var(--text-tertiary, var(--text-secondary));
        font-size: var(--text-meta);
        text-align: center;
    }

    .preview-footer code {
        font-family: var(--font-mono);
        padding: 1px 5px;
        border-radius: 4px;
        background: var(--surface-subtle);
        color: var(--text-primary);
    }
</style>
