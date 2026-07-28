<script lang="ts">
    import CopyIcon from "@lucide/svelte/icons/copy";
    import FileTextIcon from "@lucide/svelte/icons/file-text";
    import MoreHorizontalIcon from "@lucide/svelte/icons/ellipsis";
    import NetworkIcon from "@lucide/svelte/icons/network";
    import ScrollTextIcon from "@lucide/svelte/icons/scroll-text";
    import { Menu, Portal } from "@skeletonlabs/skeleton-svelte";

    import type { Stage } from "$lib/stages.js";
    import { getChips, getPhases, getStageStatusLabel, splitVector } from "../utils/stage-view.js";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
    const status = $derived(getStageStatusLabel(stage));
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

<article class={["card w-full overflow-visible border border-app-border border-t-[3px] border-t-app-accent bg-app-card text-app-text shadow-lg", selected && "outline-2 outline-offset-2 outline-app-focus"]} aria-label={`Stage ${stage.name}`}>
    <header class="grid grid-cols-[auto_minmax(0,1fr)_auto_auto] items-center gap-[0.55rem] border-b border-app-border px-[0.7rem] py-[0.65rem]">
        <span class="grid size-[1.85rem] place-items-center rounded-[0.4rem] bg-app-accent/13 text-app-accent-strong"><NetworkIcon class="size-4" aria-hidden="true" /></span>
        <span class="grid min-w-0">
            <strong class="truncate text-[0.82rem]">{stage.name}</strong>
            <small class="truncate text-[0.65rem] text-app-muted uppercase">{stage.landscapeName}</small>
        </span>
        <span class={["inline-flex items-center gap-[0.3rem] rounded-full px-[0.48rem] py-[0.24rem] text-[0.6rem] font-[750] tracking-[0.04em] whitespace-nowrap uppercase", status.tone === "healthy" ? "bg-app-success/14 text-app-success" : "bg-app-accent/13 text-app-accent-strong"]}>
            <i class={["size-[0.36rem] rounded-full bg-current", status.tone === "deploying" && "animate-status-pulse"]} aria-hidden="true"></i>
            {status.label}
        </span>
        <Menu
            aria-label={`Actions for ${stage.name}`}
            positioning={{ placement: "bottom-end", gutter: 4 }}
            onSelect={(details) => void selectAction(details.value)}
        >
            <Menu.Trigger class="grid size-[1.8rem] cursor-pointer place-items-center rounded-[0.35rem] border-0 bg-transparent p-0 text-app-muted hover:bg-app-accent/10" aria-label={`More actions for ${stage.name}`}>
                <MoreHorizontalIcon class="size-4" aria-hidden="true" />
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
    </header>
    <span class="sr-only" aria-live="polite">{actionAnnouncement}</span>

    <div class="grid gap-[0.55rem] p-[0.7rem]">
        <div class="grid grid-cols-[3.5rem_minmax(0,1fr)] items-center gap-[0.45rem]">
            <span class="text-[0.58rem] font-[750] tracking-[0.06em] text-app-muted uppercase">Vector</span>
            <span class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
                <code class="rounded border border-app-border bg-app-bg px-[0.3rem] py-[0.1rem] text-[0.66rem] [overflow-wrap:anywhere]">{vector.version}</code>
                {#if vector.hash}<code class="rounded border border-app-border bg-app-bg px-[0.3rem] py-[0.1rem] text-[0.66rem] text-app-muted [overflow-wrap:anywhere]">{vector.hash}</code>{/if}
            </span>
        </div>
        <div class="grid grid-cols-[3.5rem_minmax(0,1fr)] items-center gap-[0.45rem]">
            <span class="text-[0.58rem] font-[750] tracking-[0.06em] text-app-muted uppercase">Phases</span>
            <span class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
                {#each phases as phase (phase.id)}
                    <span
                        class={[
                            "inline-flex items-center gap-[0.3rem] rounded-full border border-app-border px-[0.4rem] py-[0.18rem] text-[0.59rem] whitespace-nowrap",
                            phase.state === "done" && "border-app-success/35 text-app-success",
                            phase.state === "cur" && "border-app-accent/40 text-app-accent-strong",
                            phase.state === "idle" && "text-app-muted",
                        ]}
                        title={phase.reason ? `${phase.label}: ${phase.reason}` : phase.label}
                    >
                        <i class="size-[0.36rem] rounded-full bg-current" aria-hidden="true"></i>{phase.label}
                    </span>
                {/each}
            </span>
        </div>
        {#if chips.length > 0}
            <div class="grid grid-cols-[3.5rem_minmax(0,1fr)] items-center gap-[0.45rem]">
                <span class="text-[0.58rem] font-[750] tracking-[0.06em] text-app-muted uppercase">Details</span>
                <span class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
                    {#each chips as chip, index (`${chip.label}-${index}`)}
                        <span class="inline-flex items-center gap-[0.3rem] rounded-full border border-app-border bg-app-bg px-[0.4rem] py-[0.18rem] text-[0.59rem] whitespace-nowrap text-app-muted"><strong class="text-app-text">{chip.value}</strong> {chip.label}</span>
                    {/each}
                </span>
            </div>
        {/if}
    </div>
</article>
