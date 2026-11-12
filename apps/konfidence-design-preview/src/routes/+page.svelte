<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { afterNavigate, beforeNavigate, goto } from "$app/navigation";
    import { SvelteURL } from "svelte/reactivity";
    import CatalogSearch from "#lib/gallery/CatalogSearch.svelte";
    import CatalogItem from "#lib/gallery/CatalogItem.svelte";
    import { CATALOG, entryId, searchCatalog, selectedEntry } from "#lib/gallery/catalog.ts";
    import type { CatalogEntry } from "#lib/gallery/catalog.ts";
    import { watchGalleryViewport } from "#lib/gallery/gallery-viewport.ts";
    import type { GalleryViewport } from "#lib/gallery/gallery-viewport.ts";

    let query = $state("");
    let navigation = $state<HTMLElement>();
    let gallery = $state<HTMLElement>();
    let viewport = $state<GalleryViewport>();
    let statuses = $state<Record<CatalogEntry["status"], boolean>>({ implemented: true, planned: false });
    const currentUrl = $derived(page.shallow?.url ?? page.url);
    const selected = $derived(selectedEntry(currentUrl));
    const searching = $derived(query.trim().length > 0);
    const matchesStatus = (entry: CatalogEntry): boolean => statuses[entry.status] || entry.name === selected?.name;
    const entries = $derived(searchCatalog(query).filter((entry) => searching || matchesStatus(entry)));

    // Normalize legacy bookmarks after hydration so SvelteKit, rather than an
    // effect racing its scroll restoration, owns the destination and history.
    beforeNavigate(({ shallow, type }) => { if (!shallow || type !== "goto") {viewport?.pause();} });
    afterNavigate(({ shallow, type }) => {
        if (shallow && type === "goto") {return;}
        const entry = selectedEntry(currentUrl);
        if (entry && currentUrl.hash !== `#${entryId(entry)}`) {
            void goto(resolve(`/#${entryId(entry)}`), { replace: true });
        }
        viewport?.settleNavigation();
    });

    $effect(() => {
        if (!gallery) {return;}
        const observer = watchGalleryViewport(gallery, (id) => {
            const entry = CATALOG.find((candidate) => entryId(candidate) === id);
            if (!entry || entry.name === selected?.name) {return;}
            const url = new SvelteURL(currentUrl.href);
            url.searchParams.delete("component");
            // Shallow replacement preserves focus, scroll, page state and history.
            void goto(resolve(url.search ? `/?${url.searchParams.toString()}#${id}` : `/#${id}`), { replace: true, reset: false, shallow: true, state: page.state });
        });
        viewport = observer;
        return () => observer.destroy();
    });

    // Clear search before native link navigation measures its hash destination.
    // Modified clicks open another tab/window without changing this page's search.
    const selectComponent = (event: MouseEvent): void => {
        if (event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey && !event.defaultPrevented) { viewport?.pause(); query = ""; }
    };

    const navigateResults = (event: KeyboardEvent): void => {
        const links = [...(navigation?.querySelectorAll<HTMLAnchorElement>("a") ?? [])];
        const index = links.findIndex((link) => link.matches(":focus"));
        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            let next = index + (event.key === "ArrowDown" ? 1 : -1);
            if (index === -1) {next = event.key === "ArrowDown" ? 0 : links.length - 1;}
            links[(next + links.length) % links.length]?.focus();
        } else if (event.key === "Enter" && event.target instanceof HTMLInputElement) {
            links[0]?.click();
        } else if (event.key === "Escape") { query = ""; }
    };
</script>

<svelte:head><title>{selected ? `${selected.name} · ` : ""}Konfidence design system</title></svelte:head>

<div class="grid grid-cols-[280px_minmax(0,1fr)] items-start gap-8 [@media(max-width:900px)]:grid-cols-1">
    <aside class="sticky top-[68px] max-h-[calc(100vh-88px)] overflow-y-auto py-2 pr-1 [@media(max-width:900px)]:static [@media(max-width:900px)]:max-h-[50vh]" aria-label="Design system catalog">
        <CatalogSearch bind:query onkeydown={navigateResults} />
        <div class="mb-3 flex flex-wrap items-center gap-x-4 gap-y-2 px-2 text-meta text-content-tertiary" role="group" aria-label="Component status legend">
            <button type="button" aria-pressed={statuses.implemented} onclick={() => statuses.implemented = !statuses.implemented} class={["inline-flex items-center gap-1.5 rounded-base border px-2 py-1 focus-visible:outline-2 focus-visible:outline-outline-focus", statuses.implemented ? "border-outline-default bg-surface-subtle font-semibold text-content-primary" : "border-transparent hover:bg-surface-subtle"]}><span class="inline-block size-2 rounded-full bg-status-healthy-solid" aria-hidden="true"></span>Implemented</button>
            <button type="button" aria-pressed={statuses.planned} onclick={() => statuses.planned = !statuses.planned} class={["inline-flex items-center gap-1.5 rounded-base border px-2 py-1 focus-visible:outline-2 focus-visible:outline-outline-focus", statuses.planned ? "border-outline-default bg-surface-subtle font-semibold text-content-primary" : "border-transparent hover:bg-surface-subtle"]}><span class="inline-block size-2 rounded-full bg-outline-default" aria-hidden="true"></span>Planned</button>
        </div>
        <nav bind:this={navigation} aria-label="Components">
            <ul class="m-0 list-none p-0">
                {#each entries as entry (entry.name)}
                    <li>
                        <a href={resolve(`/#${entryId(entry)}`)} aria-label={entry.name} aria-describedby={`catalog-status-${entry.name}`} aria-current={selected?.name === entry.name ? "location" : undefined} onclick={selectComponent} onkeydown={navigateResults} class={["flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-compact no-underline hover:bg-surface-subtle", selected?.name === entry.name ? "bg-surface-subtle font-semibold text-content-primary" : "text-content-secondary"]}>
                            <span>{entry.name}</span>
                            <span id={`catalog-status-${entry.name}`} role="img" aria-label={entry.status === "implemented" ? "Implemented" : "Planned"} title={entry.status === "implemented" ? "Implemented" : "Planned"} class={["inline-block size-2 shrink-0 rounded-full", entry.status === "implemented" ? "bg-status-healthy-solid" : "bg-outline-default"]}></span>
                        </a>
                    </li>
                {:else}<li class="px-2.5 py-3 text-compact text-content-tertiary" role="status">No matching components.</li>{/each}
            </ul>
        </nav>
    </aside>

    <div bind:this={gallery} class="min-w-0">
        <header class="mb-8">
            <p class="mb-1.5 text-meta font-semibold tracking-wide text-content-tertiary uppercase">Style guide</p>
            <h1 class="text-display font-display tracking-[-1px] text-content-primary">Konfidence design system</h1>
            <p class="mt-3 text-body text-content-secondary">Search across all components. Selecting a result restores the status filters; the URL-selected component always stays visible.</p>
        </header>
        {#each CATALOG as entry (entry.name)}
            <CatalogItem {entry} hidden={!searching && !matchesStatus(entry)} />
        {/each}
        {#if !searching && !statuses.implemented && !statuses.planned && !selected}
            <p class="rounded-base border border-outline-subtle bg-surface-card p-5 text-body text-content-secondary" role="status">Select Implemented or Planned to show components.</p>
        {/if}
    </div>
</div>
