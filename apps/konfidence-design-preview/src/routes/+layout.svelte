<script lang="ts">
    import "../app.css";

    import type { Snippet } from "svelte";
    import { Brandbar, ColorModeSelect } from "@konfidence/design-system/components";
    import { resolve } from "$app/paths";
    import { ModeWatcher, mode, setMode, userPrefersMode } from "mode-watcher";

    interface Props {
        children?: Snippet;
    }

    let { children }: Props = $props();

    const MAIN_ID = "preview-main";

</script>

<ModeWatcher defaultTheme="konfidence" />

<a class="preview-skip-link" href={`#${MAIN_ID}`}>Skip to main content</a>

<div class="preview-shell">
    <Brandbar />
    <header class="preview-topbar">
        <a href={resolve("/")} class="preview-topbar__brand">
            <strong>Konfidence</strong>
            <span class="preview-topbar__dim">Design System</span>
        </a>
        <div class="preview-topbar__actions">
            <ColorModeSelect value={userPrefersMode.current} resolvedMode={mode.current} onValueChange={setMode} />
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
    .preview-skip-link {
        position: absolute;
        top: 12px;
        left: 12px;
        z-index: 500;
        padding: 6px 10px;
        border-radius: var(--radius-md, 8px);
        border: 1px solid var(--border-focus);
        background: var(--surface-default);
        color: var(--text-primary);
        font-size: var(--text-sm);
        transform: translateY(-200%);
        transition: transform var(--motion-fast, 120ms) var(--ease);
    }

    .preview-skip-link:focus-visible {
        transform: translateY(0);
    }

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

    .preview-topbar__actions {
        margin-left: auto;
        display: inline-flex;
        align-items: center;
        gap: 8px;
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
