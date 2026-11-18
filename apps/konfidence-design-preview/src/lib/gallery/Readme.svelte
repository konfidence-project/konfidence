<script lang="ts">
    import DOMPurify from "dompurify";
    import { marked } from "marked";

    let { name }: { name: string } = $props();
    let expanded = $state(false);
    const id = $props.id();
    const readmes = import.meta.glob<string>(
        "../../../../../packages/konfidence-design-system/src/components/*/README.md",
        { eager: true, import: "default", query: "?raw" },
    );
    const markdown = $derived(Object.entries(readmes).find(([path]) => path.endsWith(`/${name}/README.md`))?.[1]);
    const html = $derived(markdown ? DOMPurify.sanitize(marked.parse(markdown, { async: false })) : "");
    const previewTokens = $derived(marked.lexer(markdown ?? "").filter((token) => token.type !== "space").slice(0, 2));
    const teaser = $derived(DOMPurify.sanitize(marked.parser(previewTokens)));
</script>

{#if markdown?.trim()}
    <section class="mb-5" aria-label={`Documentation for ${name}`}>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <h2 class="text-body font-semibold">{name} documentation</h2>
            <button class="shrink-0 text-compact text-content-link underline" aria-expanded={expanded} aria-controls={id} onclick={() => expanded = !expanded}>{expanded ? "Collapse documentation" : "Expand documentation"}</button>
        </div>
        <div id={id} class="readme text-body leading-relaxed text-content-secondary">
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- Repository Markdown is sanitized before rendering. -->
            {@html expanded ? html : teaser}
        </div>
    </section>
{/if}

<style>
    .readme :global(h1), .readme :global(h2), .readme :global(h3) {
        margin: 1.25rem 0 0.5rem;
        color: var(--text-primary);
        font-weight: 600;
    }
    .readme :global(p), .readme :global(ul), .readme :global(ol) { margin: 0.75rem 0; }
    .readme :global(ul) { list-style: disc; padding-left: 1.5rem; }
    .readme :global(ol) { list-style: decimal; padding-left: 1.5rem; }
    .readme :global(pre) {
        overflow-x: auto;
        padding: 1rem;
        border-radius: var(--radius-sm);
        background: var(--surface-subtle);
        font-size: var(--font-size-meta);
    }
    .readme :global(code) { font-family: var(--font-mono); }
    .readme :global(a) { color: var(--text-link); text-decoration: underline; }
    .readme :global(table) { display: block; overflow-x: auto; border-collapse: collapse; }
    .readme :global(th), .readme :global(td) { border: 1px solid var(--border-subtle); padding: 0.5rem; }
</style>
