<script lang="ts">
    import { onDestroy } from "svelte";
    import { SvelteURL } from "svelte/reactivity";
    import { resolve } from "$app/paths";
    import { entryId } from "./catalog.ts";
    import type { CatalogEntry } from "./catalog.ts";

    let { entry }: { entry: CatalogEntry } = $props();
    let feedback = $state("");
    let timeout: ReturnType<typeof setTimeout> | undefined = undefined;
    const href = $derived(resolve(`/#${entryId(entry)}`));

    const copyLink = async (event: MouseEvent): Promise<void> => {
        // Preserve native new-tab and context-menu behavior for this real link.
        if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.defaultPrevented) {return;}
        event.preventDefault();
        clearTimeout(timeout);
        feedback = "";
        try {
            await navigator.clipboard.writeText(new SvelteURL(href, location.href).href);
            feedback = "Link copied";
        } catch {
            feedback = "Clipboard unavailable. Use Copy link address from the anchor's context menu.";
        }
        timeout = setTimeout(() => { feedback = ""; }, 3000);
    };
    onDestroy(() => clearTimeout(timeout));
</script>

<a {href} onclick={copyLink} aria-label={`Copy link to ${entry.name}`} title={`Copy link to ${entry.name}`} class="inline-flex shrink-0 items-center justify-center rounded px-1.5 py-1 font-mono text-body font-semibold text-content-tertiary no-underline hover:text-content-link focus-visible:outline-2 focus-visible:outline-outline-focus">#</a>
{#if feedback}<span class="text-meta font-normal text-content-tertiary" role="status">{feedback}</span>{/if}
