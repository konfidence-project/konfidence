<script lang="ts">
    import CopyIcon from "@lucide/svelte/icons/copy";
    import EllipsisVerticalIcon from "@lucide/svelte/icons/ellipsis-vertical";
    import FileTextIcon from "@lucide/svelte/icons/file-text";
    import ScrollTextIcon from "@lucide/svelte/icons/scroll-text";
    import { Menu, Portal } from "@skeletonlabs/skeleton-svelte";

    import type { Stage } from "$lib/stages.js";
    import {
        getChips,
        getLandscapeLabel,
        getPhases,
        getStageStatusLabel,
        splitVector,
    } from "../utils/stage-view.js";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
    const status = $derived(getStageStatusLabel(stage));
    const landscape = $derived(getLandscapeLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.vector));
    let actionAnnouncement = $state("");

    const selectAction = async (value: string): Promise<void> => {
        if (value === "copy") {
            await globalThis.navigator.clipboard.writeText(stage.name);
            actionAnnouncement = `${stage.name} copied to clipboard`;
        }
    };
</script>

<article
    data-testid="stage-card"
    data-health={status.tone}
    class={["card relative w-full overflow-hidden border border-app-border bg-app-card text-app-text shadow-lg", selected && "outline-2 outline-offset-2 outline-app-focus"]}
    aria-label={`Stage ${stage.name}`}
>
    <div class={["absolute inset-x-0 top-0 h-[0.2rem]", status.tone === "healthy" ? "bg-app-success" : "bg-app-accent"]} aria-hidden="true"></div>

    <div class="grid gap-[0.35rem] border-b border-app-border pt-[0.7rem] pr-[2.4rem] pb-[0.65rem] pl-[0.7rem]">
        <div class="flex items-center justify-between gap-[0.55rem]">
            <span class="truncate text-[0.82rem] font-bold">{stage.name}</span>
            <span class={["inline-flex items-center gap-[0.3rem] rounded-full px-[0.48rem] py-[0.24rem] text-[0.6rem] font-[750] tracking-[0.04em] whitespace-nowrap uppercase", status.tone === "healthy" ? "bg-app-success/14 text-app-success" : "bg-app-accent/13 text-app-accent-strong"]}>
                <i class={["size-[0.36rem] rounded-full bg-current", status.tone === "deploying" && "animate-status-pulse"]} aria-hidden="true"></i>
                {status.label}
            </span>
        </div>
        <div class="truncate text-[0.65rem] tracking-[0.05em] text-app-muted uppercase">{landscape}</div>
        <div class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
            <code class="rounded border border-app-border bg-app-bg px-[0.3rem] py-[0.1rem] text-[0.66rem] [overflow-wrap:anywhere]">{vector.version}</code>
            {#if vector.hash}<code class="rounded border border-app-border bg-app-bg px-[0.3rem] py-[0.1rem] text-[0.66rem] text-app-muted [overflow-wrap:anywhere]">{vector.hash}</code>{/if}
        </div>
    </div>

    <div class="grid gap-[0.5rem] p-[0.7rem]">
        <div class="grid grid-cols-3 gap-[0.3rem]">
            {#each phases as phase (phase.id)}
                <div
                    class={[
                        "h-[0.3rem] rounded-full",
                        phase.state === "done" && "bg-app-success",
                        phase.state === "cur" && "bg-app-accent animate-status-pulse",
                        phase.state === "idle" && "bg-app-border",
                    ]}
                    title={phase.reason ? `${phase.label}: ${phase.reason}` : phase.label}
                ></div>
            {/each}
        </div>
        <div class="grid grid-cols-3 gap-[0.3rem]">
            {#each phases as phase (phase.id)}
                <div
                    class={[
                        "truncate text-[0.58rem] font-[650] tracking-[0.04em] uppercase",
                        phase.state === "done" && "text-app-success",
                        phase.state === "cur" && "text-app-accent-strong",
                        phase.state === "idle" && "text-app-muted",
                    ]}
                >{phase.label}</div>
            {/each}
        </div>
        {#if chips.length > 0}
            <div class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
                {#each chips as chip, index (`${chip.label}-${index}`)}
                    <span class="inline-flex items-center gap-[0.3rem] rounded-full border border-app-border bg-app-bg px-[0.4rem] py-[0.18rem] text-[0.59rem] whitespace-nowrap text-app-muted"><strong class="text-app-text">{chip.value}</strong> {chip.label}</span>
                {/each}
            </div>
        {/if}
    </div>

    <Menu
        aria-label={`Actions for ${stage.name}`}
        positioning={{ placement: "bottom-end", gutter: 4 }}
        onSelect={(details) => void selectAction(details.value)}
    >
        <Menu.Trigger class="absolute top-[0.55rem] right-[0.45rem] grid size-[1.8rem] cursor-pointer place-items-center rounded-[0.35rem] border-0 bg-transparent p-0 text-app-muted hover:bg-app-accent/10" aria-label={`More actions for ${stage.name}`}>
            <EllipsisVerticalIcon class="size-4" aria-hidden="true" />
        </Menu.Trigger>
        <Portal target={globalThis.document?.getElementById("app-overlays") ?? undefined}>
            <Menu.Positioner class="pointer-events-auto z-10">
                <Menu.Content class="w-40 rounded-[0.45rem] border border-app-border bg-app-card p-[0.35rem] text-app-text shadow-xl [&_[data-part=item]]:flex [&_[data-part=item]]:w-full [&_[data-part=item]]:cursor-pointer [&_[data-part=item]]:items-center [&_[data-part=item]]:rounded-[0.3rem] [&_[data-part=item]]:p-2 [&_[data-part=item]]:text-[0.7rem] [&_[data-part=item]]:text-app-text [&_[data-part=item][data-highlighted]]:bg-app-accent/9 [&_[data-part=item-text]]:flex [&_[data-part=item-text]]:items-center [&_[data-part=item-text]]:gap-[0.45rem] [&_svg]:size-[0.85rem]">
                    <Menu.Item value="copy">
                        <Menu.ItemText><CopyIcon aria-hidden="true" /> Copy stage name</Menu.ItemText>
                    </Menu.Item>
                    <Menu.Item value="yaml" disabled={true}>
                        <Menu.ItemText><FileTextIcon aria-hidden="true" /> View YAML</Menu.ItemText>
                    </Menu.Item>
                    <Menu.Item value="logs" disabled={true}>
                        <Menu.ItemText><ScrollTextIcon aria-hidden="true" /> Open logs</Menu.ItemText>
                    </Menu.Item>
                </Menu.Content>
            </Menu.Positioner>
        </Portal>
    </Menu>
    <span class="sr-only" aria-live="polite">{actionAnnouncement}</span>
</article>
