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

<a class="preview-skip-link absolute top-3 left-3 z-[500] -translate-y-[200%] rounded-control border border-outline-focus bg-surface-default px-2.5 py-1.5 text-ui-sm text-content-primary transition-transform duration-[var(--motion-fast,120ms)] ease-[var(--ease)] focus-visible:translate-y-0" href={`#${MAIN_ID}`}>Skip to main content</a>

<div class="preview-shell flex min-h-screen flex-col bg-surface-canvas text-content-primary">
    <Brandbar />
    <header class="preview-topbar sticky top-0 z-20 flex items-center gap-5 border-b border-outline-subtle bg-[color-mix(in_oklab,var(--surface-default)_92%,transparent)] px-5 py-2.5 backdrop-blur-[6px]">
        <a href={resolve("/")} class="preview-topbar__brand inline-flex items-baseline gap-1.5 text-ui-body text-content-primary no-underline">
            <strong class="font-ui-display tracking-[-0.02em]">Konfidence</strong>
            <span class="preview-topbar__dim text-ui-sm text-content-tertiary">Design System</span>
        </a>
        <div class="preview-topbar__actions ml-auto inline-flex items-center gap-2">
            <ColorModeSelect value={userPrefersMode.current} resolvedMode={mode.current} onValueChange={setMode} />
        </div>
    </header>

    <main id={MAIN_ID} class="preview-main mx-auto w-full max-w-[1200px] flex-1 px-6 pt-8 pb-16">
        {@render children?.()}
    </main>

    <footer class="preview-footer border-t border-outline-subtle px-5 py-4 text-center text-meta text-content-tertiary">
        <span>Konfidence Design Preview · reads live from <code class="rounded bg-surface-subtle px-[5px] py-px font-ui-mono text-content-primary">@konfidence/design-system</code>.</span>
    </footer>
</div>
