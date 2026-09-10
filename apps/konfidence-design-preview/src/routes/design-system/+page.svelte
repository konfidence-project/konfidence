<script lang="ts">
    import ActivityFeed from "$lib/gallery/sections/ActivityFeed.svelte";
    import Atoms from "$lib/gallery/sections/Atoms.svelte";
    import Buttons from "$lib/gallery/sections/Buttons.svelte";
    import Charts from "$lib/gallery/sections/Charts.svelte";
    import Colors from "$lib/gallery/sections/Colors.svelte";
    import DataBlocks from "$lib/gallery/sections/DataBlocks.svelte";
    import Feedback from "$lib/gallery/sections/Feedback.svelte";
    import Forms from "$lib/gallery/sections/Forms.svelte";
    import Hero from "$lib/gallery/sections/Hero.svelte";
    import Icons from "$lib/gallery/sections/Icons.svelte";
    import Lists from "$lib/gallery/sections/Lists.svelte";
    import LogsAndCode from "$lib/gallery/sections/LogsAndCode.svelte";
    import MessageStrips from "$lib/gallery/sections/MessageStrips.svelte";
    import Misc from "$lib/gallery/sections/Misc.svelte";
    import Navigation from "$lib/gallery/sections/Navigation.svelte";
    import Overlays from "$lib/gallery/sections/Overlays.svelte";
    import Pipeline from "$lib/gallery/sections/Pipeline.svelte";
    import Shell from "$lib/gallery/sections/Shell.svelte";
    import ShellPatterns from "$lib/gallery/sections/ShellPatterns.svelte";
    import Signature from "$lib/gallery/sections/Signature.svelte";
    import StatusBadges from "$lib/gallery/sections/StatusBadges.svelte";
    import Tags from "$lib/gallery/sections/Tags.svelte";
    import Typography from "$lib/gallery/sections/Typography.svelte";

    const TOC: readonly { id: string; label: string; status: "implemented" | "partial" | "placeholder" }[] = [
        { id: "signature", label: "Signature", status: "implemented" },
        { id: "foundation-colors", label: "Colors", status: "implemented" },
        { id: "foundation-tags", label: "Tags", status: "placeholder" },
        { id: "foundation-typography", label: "Typography", status: "implemented" },
        { id: "foundation-atoms", label: "Atoms", status: "placeholder" },
        { id: "icons", label: "Icons", status: "implemented" },
        { id: "buttons", label: "Buttons", status: "implemented" },
        { id: "status-badges", label: "Status badges", status: "implemented" },
        { id: "forms", label: "Forms", status: "placeholder" },
        { id: "navigation", label: "Navigation", status: "placeholder" },
        { id: "shell", label: "App shell", status: "implemented" },
        { id: "shell-patterns", label: "Shell patterns", status: "partial" },
        { id: "data-blocks", label: "Data blocks", status: "placeholder" },
        { id: "lists", label: "Lists", status: "placeholder" },
        { id: "pipeline", label: "Pipeline", status: "placeholder" },
        { id: "hero", label: "Hero", status: "placeholder" },
        { id: "logs-code", label: "Logs & code", status: "placeholder" },
        { id: "feedback", label: "Feedback", status: "placeholder" },
        { id: "message-strips", label: "Message strips", status: "placeholder" },
        { id: "activity-feed", label: "Activity feed", status: "placeholder" },
        { id: "charts", label: "Charts", status: "placeholder" },
        { id: "misc", label: "Misc.", status: "partial" },
        { id: "overlays", label: "Overlays", status: "placeholder" },
    ];
</script>

<div class="page">
    <aside class="toc" aria-label="Sections">
        <p class="toc__title">Sections</p>
        <nav>
            <ol>
                {#each TOC as entry (entry.id)}
                    <li>
                        <a href={`#${entry.id}`}>
                            <span>{entry.label}</span>
                            <span class="toc__dot" data-status={entry.status} aria-hidden="true"></span>
                        </a>
                    </li>
                {/each}
            </ol>
        </nav>
        <div class="toc__legend">
            <span><span class="toc__dot" data-status="implemented"></span>Live</span>
            <span><span class="toc__dot" data-status="partial"></span>Partial</span>
            <span><span class="toc__dot" data-status="placeholder"></span>Pending</span>
        </div>
    </aside>

    <div class="gallery">
        <header class="gallery__header">
            <p class="gallery__eyebrow">Style guide</p>
            <h1 class="gallery__title">Konfidence design system</h1>
            <p class="gallery__lead">
                All sections from the reference design-system.html are listed below. Where the
                Svelte component surface has caught up, real components render live. Where it has
                not, a placeholder card lists the planned primitives and links to the roadmap.
            </p>
        </header>

        <Signature />
        <Colors />
        <Tags />
        <Typography />
        <Atoms />
        <Icons />
        <Buttons />
        <StatusBadges />
        <Forms />
        <Navigation />
        <Shell />
        <ShellPatterns />
        <DataBlocks />
        <Lists />
        <Pipeline />
        <Hero />
        <LogsAndCode />
        <Feedback />
        <MessageStrips />
        <ActivityFeed />
        <Charts />
        <Misc />
        <Overlays />
    </div>
</div>

<style>
    .page {
        display: grid;
        grid-template-columns: 220px 1fr;
        gap: 32px;
        align-items: start;
    }

    @media (max-width: 900px) {
        .page {
            grid-template-columns: 1fr;
        }
        .toc {
            position: static !important;
        }
    }

    .toc {
        position: sticky;
        top: 68px;
        max-height: calc(100vh - 88px);
        overflow-y: auto;
        padding: 8px 4px 8px 0;
    }

    .toc__title {
        margin: 0 0 8px;
        padding: 0 8px;
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        text-transform: uppercase;
        letter-spacing: 0.06em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .toc nav ol {
        list-style: none;
        margin: 0;
        padding: 0;
    }

    .toc nav a {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        padding: 6px 10px;
        border-radius: 6px;
        color: var(--text-secondary);
        text-decoration: none;
        font-size: var(--text-sm);
    }

    .toc nav a:hover {
        background: var(--surface-subtle);
        color: var(--text-primary);
    }

    .toc__dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;
        background: var(--text-tertiary, var(--text-secondary));
    }

    .toc__dot[data-status="implemented"] {
        background: var(--status-healthy-solid, #17b26a);
    }

    .toc__dot[data-status="partial"] {
        background: var(--status-warning-solid, #f79009);
    }

    .toc__dot[data-status="placeholder"] {
        background: var(--border-default, var(--border-subtle));
    }

    .toc__legend {
        margin-top: 16px;
        padding: 8px 10px;
        display: flex;
        flex-direction: column;
        gap: 4px;
        font-size: var(--text-meta);
        color: var(--text-tertiary, var(--text-secondary));
    }

    .toc__legend span {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .gallery__header {
        margin-bottom: 32px;
    }

    .gallery__eyebrow {
        margin: 0 0 6px;
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        text-transform: uppercase;
        letter-spacing: 0.06em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .gallery__title {
        margin: 0 0 12px;
        font-size: var(--text-display, 36px);
        font-weight: var(--weight-display, 600);
        letter-spacing: -1px;
        color: var(--text-primary);
    }

    .gallery__lead {
        margin: 0;
        max-width: 720px;
        color: var(--text-secondary);
        font-size: var(--text-body);
        line-height: 1.6;
    }
</style>
