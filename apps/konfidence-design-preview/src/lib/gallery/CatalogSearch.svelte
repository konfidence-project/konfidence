<script lang="ts">
    import { onMount } from "svelte";

    let { query = $bindable(""), onkeydown }: { query?: string; onkeydown?: (event: KeyboardEvent) => void } = $props();
    let input = $state<HTMLInputElement>();
    let isMac = $state(false);
    const shortcut = $derived(isMac ? "⌘K" : "Ctrl+K");
    onMount(() => {
        isMac = /Macintosh|Mac OS|iPhone|iPad|iPod/.test(navigator.userAgent);
        globalThis.addEventListener("keydown", shortcutKey, { capture: true });
        globalThis.addEventListener("preview:focus-search", focusSearch);
        return () => {
            globalThis.removeEventListener("keydown", shortcutKey, { capture: true });
            globalThis.removeEventListener("preview:focus-search", focusSearch);
        };
    });

    const focusSearch = (): void => { input?.focus(); input?.select(); };
    const shortcutKey = (event: KeyboardEvent): void => {
        if ((isMac ? event.metaKey : event.ctrlKey) && event.key.toLowerCase() === "k" && !event.altKey) {
            event.preventDefault();
            event.stopPropagation();
            focusSearch();
        }
    };
</script>

<label for="catalog-search" class="mb-2 block text-meta font-semibold text-content-secondary">Search components</label>
<div class="relative mb-4">
    <input bind:this={input} id="catalog-search" type="search" bind:value={query} {onkeydown} placeholder="Component or keyword…" class="w-full rounded-base border border-outline-default bg-surface-card py-2 pr-20 pl-3 text-compact" />
    <button class="absolute top-1/2 right-3 -translate-y-1/2 rounded border border-outline-subtle px-1.5 py-0.5 font-mono text-meta text-content-tertiary hover:bg-surface-subtle" aria-label="Focus catalog search" aria-keyshortcuts={isMac ? "Meta+K" : "Control+K"} onclick={focusSearch}>{shortcut}</button>
</div>
