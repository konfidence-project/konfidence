<script lang="ts">
    import { untrack } from "svelte";
    import type { Component } from "svelte";
    import { Tabs } from "@skeletonlabs/skeleton-svelte";
    import { Button } from "@konfidence/design-system/components";
    import { createFramePreview } from "./frame-preview.ts";
    import type { FramePreview, PreviewState } from "./frame-preview.ts";

    let { source, title = "Example", previewHeight = 320 }: { source: string; title?: string; previewHeight?: number } = $props();
    let code = $state(untrack(() => source));
    let revision = $state(0);
    let preview = $state<PreviewState>({ status: "loading" });
    let copyStatus = $state("");
    let host = $state<HTMLDivElement>();
    let active = $state(false);
    let tab = $state<"preview" | "code">("preview");
    let Editor = $state<Component<{ value?: string; label: string; onChange?: () => void }>>();
    let editorError = $state("");
    let renderer = $state<FramePreview>();

    // The editor is loaded once on first use and remains mounted across tab
    // switches, preserving selections and undo history without hiding other examples.
    $effect(() => {
        if (tab !== "code" || Editor) {return;}
        let cancelled = false;
        editorError = "";
        void import("./CodeEditor.svelte")
            .then((module) => { if (!cancelled) {Editor = module.default;} })
            .catch((error: unknown) => { if (!cancelled) {editorError = error instanceof Error ? error.message : String(error);} });
        return () => { cancelled = true; };
    });

    $effect(() => {
        if (!host) {return;}
        const session = createFramePreview(host, { title }, (state) => { preview = state; });
        renderer = session;
        return () => session.dispose();
    });

    // Compile only examples near the viewport. Source remains copyable
    // before activation, without loading a compiler for the whole catalog.
    $effect(() => {
        if (!host) {return;}
        const observer = new IntersectionObserver((entries) => {
            if (entries.some((entry) => entry.isIntersecting)) { active = true; observer.disconnect(); }
        }, { rootMargin: "200px" });
        observer.observe(host);
        return () => observer.disconnect();
    });

    $effect(() => {
        if (!active || !renderer) {return;}
        // Revision retriggers Reset even when the source is unchanged. Keep
        // every edit debounced, including edits after the first reset.
        const request = { revision, source: code };
        const session = renderer;
        const controller = new AbortController();
        const timer = setTimeout(() => session.render(request.source, controller.signal), 300);
        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    });

    const copy = async (): Promise<void> => {
        try { await navigator.clipboard.writeText(code); copyStatus = "Copied"; }
        catch { tab = "code"; copyStatus = "Clipboard unavailable. Select the code and copy it manually."; }
    };
</script>

<div class="w-full min-w-0">
    <Tabs value={tab} onValueChange={({ value }) => { if (value === "preview" || value === "code") tab = value; }}>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <Tabs.List aria-label={`Example tabs: ${title}`} class="flex gap-1 rounded-base bg-surface-subtle p-1">
                <Tabs.Trigger value="preview" class="rounded px-3 py-1.5 text-compact data-[selected]:bg-surface-card data-[selected]:font-semibold">Preview</Tabs.Trigger>
                <Tabs.Trigger value="code" class="rounded px-3 py-1.5 text-compact data-[selected]:bg-surface-card data-[selected]:font-semibold">Code</Tabs.Trigger>
            </Tabs.List>
            <div class="playground-actions flex items-center gap-2">
                <Button variant="secondary" icon="copy" onclick={copy}>Copy code</Button>
                <Button variant="ghost" onclick={() => { code = source; revision += 1; copyStatus = ""; active = true; }}>Reset</Button>
            </div>
        </div>
        {#if copyStatus}<p class="mb-2 text-meta text-content-secondary" role="status">{copyStatus}</p>{/if}
        <Tabs.Content value="preview">
            <!-- Reserve the viewport before lazy loading. Auto-growing frames
                 would move hash destinations during native smooth scrolling. -->
            <div bind:this={host} style:height={`${previewHeight}px`} class="min-h-32 resize-y overflow-hidden rounded-base border border-outline-subtle bg-surface-default"></div>
        </Tabs.Content>
        <Tabs.Content value="code">
            {#if Editor}
                <Editor bind:value={code} label={`Svelte code: ${title}`} onChange={() => { active = true; copyStatus = ""; }} />
            {:else if editorError}
                <p class="mb-3 text-compact text-status-error-fg" role="alert">Could not load the editor: {editorError}</p>
                <!-- Browsers cache failed ES module imports; retrying the same
                     dynamic import cannot recover without a new document. -->
                <Button variant="secondary" onclick={() => location.reload()}>Reload preview</Button>
                <p class="mt-2 text-meta text-content-tertiary">Reloading resets local example edits.</p>
            {:else}<p class="text-compact text-content-tertiary">Loading editor…</p>{/if}
        </Tabs.Content>
    </Tabs>
    <p class="mt-3 text-meta text-content-tertiary" role="status">{preview.status === "ready" ? "Preview ready" : preview.status === "error" ? "Preview failed" : "Preview loads when visible"}</p>
    {#if preview.status === "error"}<pre class="mt-3 overflow-x-auto rounded-base bg-status-error-bg p-3 text-meta whitespace-pre-wrap text-status-error-fg" role="alert">{preview.message}</pre>{/if}
    <p class="mt-3 text-meta text-content-tertiary">Edits update the preview automatically. Example state resets on recompilation. Demo links do not navigate.</p>
</div>

<style>
    /* UI5's unlayered icon color otherwise overrides the toolbar button's
       foreground in dark mode. Keep this override local to preview controls. */
    .playground-actions :global([ui5-icon]) { color: inherit; }
</style>
