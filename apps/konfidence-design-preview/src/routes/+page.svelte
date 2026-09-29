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
        { id: "forms", label: "Forms", status: "partial" },
        { id: "navigation", label: "Navigation", status: "partial" },
        { id: "shell", label: "App shell", status: "implemented" },
        { id: "shell-patterns", label: "Shell patterns", status: "partial" },
        { id: "data-blocks", label: "Data blocks", status: "partial" },
        { id: "lists", label: "Lists", status: "placeholder" },
        { id: "pipeline", label: "Pipeline", status: "partial" },
        { id: "hero", label: "Page header", status: "partial" },
        { id: "logs-code", label: "Logs & code", status: "placeholder" },
        { id: "feedback", label: "Feedback", status: "partial" },
        { id: "message-strips", label: "Message strips", status: "placeholder" },
        { id: "activity-feed", label: "Activity feed", status: "placeholder" },
        { id: "charts", label: "Charts", status: "placeholder" },
        { id: "misc", label: "Misc.", status: "partial" },
        { id: "overlays", label: "Overlays", status: "partial" },
    ];
</script>

<div class="page grid grid-cols-[220px_1fr] items-start gap-8 [@media(max-width:900px)]:grid-cols-1">
    <aside class="toc sticky top-[68px] max-h-[calc(100vh-88px)] overflow-y-auto py-2 pr-1 [@media(max-width:900px)]:static" aria-label="Sections">
        <p class="toc__title mb-2 px-2 text-meta font-ui-semibold tracking-[0.06em] text-content-tertiary uppercase">Sections</p>
        <nav>
            <ol class="m-0 list-none p-0">
                {#each TOC as entry (entry.id)}
                    <li>
                        <a class="flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-ui-sm text-content-secondary no-underline hover:bg-surface-subtle hover:text-content-primary" href={`#${entry.id}`}>
                            <span>{entry.label}</span>
                            <span class={[
                                "toc__dot inline-block size-2 rounded-full",
                                entry.status === "implemented" && "bg-status-healthy-solid",
                                entry.status === "partial" && "bg-status-warning-solid",
                                entry.status === "placeholder" && "bg-outline-default",
                            ]} data-status={entry.status} aria-hidden="true"></span>
                        </a>
                    </li>
                {/each}
            </ol>
        </nav>
        <div class="toc__legend mt-4 flex flex-col gap-1 px-2.5 py-2 text-meta text-content-tertiary [&>span]:inline-flex [&>span]:items-center [&>span]:gap-1.5">
            <span><span class="toc__dot inline-block size-2 rounded-full bg-status-healthy-solid" data-status="implemented"></span>Live</span>
            <span><span class="toc__dot inline-block size-2 rounded-full bg-status-warning-solid" data-status="partial"></span>Partial</span>
            <span><span class="toc__dot inline-block size-2 rounded-full bg-outline-default" data-status="placeholder"></span>Pending</span>
        </div>
    </aside>

    <div class="gallery">
        <header class="gallery__header mb-8">
            <p class="gallery__eyebrow mb-1.5 text-meta font-ui-semibold tracking-[0.06em] text-content-tertiary uppercase">Style guide</p>
            <h1 class="gallery__title mb-3 text-ui-display font-ui-display tracking-[-1px] text-content-primary">Konfidence design system</h1>
            <p class="gallery__lead m-0 max-w-[720px] text-ui-body leading-[1.6] text-content-secondary">
                Explore the design tokens, components and patterns below. Pending components
                include a list of planned primitives and a link to the roadmap.
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
