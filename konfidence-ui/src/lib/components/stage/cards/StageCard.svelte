<script lang="ts">
    import CopyIcon from "@lucide/svelte/icons/copy";
    import EllipsisVerticalIcon from "@lucide/svelte/icons/ellipsis-vertical";
    import FileTextIcon from "@lucide/svelte/icons/file-text";
    import ScrollTextIcon from "@lucide/svelte/icons/scroll-text";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import type { Stage } from "$lib/stages.js";
    import { getChips, getPhases, getStageStatusLabel, splitVector } from "../utils/stage-view.js";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
    const status = $derived(getStageStatusLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.vector));

    let actionAnnouncement = $state("");

    const copyName = async (): Promise<void> => {
        await globalThis.navigator?.clipboard?.writeText(stage.name);
        actionAnnouncement = `${stage.name} copied to clipboard`;
    };
</script>

<article
    data-testid="stage-card"
    data-health={status.tone}
    class={[
        "relative w-80 overflow-hidden rounded-xl border bg-card text-card-foreground shadow-lg",
        selected && "outline-2 outline-offset-2 outline-primary",
    ]}
    aria-label={`Stage ${stage.name}`}
>
    <div
        class={[
            "absolute inset-x-0 top-0 h-[0.2rem]",
            status.tone === "healthy" ? "bg-success" : "bg-information",
        ]}
        aria-hidden="true"
    ></div>

    <div class="grid gap-[0.35rem] border-b pt-[0.7rem] pr-[2.4rem] pb-[0.65rem] pl-[0.7rem]">
        <div class="flex items-center justify-between gap-[0.55rem]">
            <span class="truncate text-[0.82rem] font-bold">{stage.name}</span>
            <span
                class={[
                    "inline-flex items-center gap-[0.3rem] rounded-full px-[0.48rem] py-[0.24rem] text-[0.6rem] font-[750] tracking-[0.04em] whitespace-nowrap uppercase",
                    status.tone === "healthy"
                        ? "bg-success-background text-success"
                        : "bg-information-background text-information",
                ]}
            >
                <i
                    class={[
                        "size-[0.36rem] rounded-full bg-current",
                        status.tone === "deploying" &&
                            "animate-status-pulse motion-reduce:animate-none",
                    ]}
                    aria-hidden="true"
                ></i>
                {status.label}
            </span>
        </div>
        <div class="truncate text-[0.65rem] tracking-[0.05em] text-muted-foreground uppercase">
            {stage.landscapeName}
        </div>
        <div class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
            <code
                class="rounded border bg-muted px-[0.3rem] py-[0.1rem] text-[0.66rem] text-foreground [overflow-wrap:anywhere]"
            >{vector.version}</code>
            {#if vector.hash}
                <code
                    class="rounded border bg-muted px-[0.3rem] py-[0.1rem] text-[0.66rem] text-muted-foreground [overflow-wrap:anywhere]"
                >{vector.hash}</code>
            {/if}
        </div>
    </div>

    <div class="grid gap-[0.5rem] p-[0.7rem]">
        <div class="grid grid-cols-3 gap-[0.3rem]">
            {#each phases as phase (phase.id)}
                <div
                    class={[
                        "h-[0.3rem] rounded-full",
                        phase.state === "done" && "bg-success",
                        phase.state === "cur" &&
                            "bg-information animate-status-pulse motion-reduce:animate-none",
                        phase.state === "idle" && "bg-border",
                    ]}
                    title={phase.reason ? `${phase.label}: ${phase.reason}` : phase.label}
                ></div>
            {/each}
        </div>
        <div class="grid grid-cols-3 gap-[0.3rem]">
            {#each phases as phase (phase.id)}
                <div
                    class={[
                        "truncate text-center text-[0.58rem] font-[650] tracking-[0.04em] uppercase",
                        phase.state === "done" && "text-success",
                        phase.state === "cur" && "text-information",
                        phase.state === "idle" && "text-muted-foreground",
                    ]}
                >{phase.label}</div>
            {/each}
        </div>
        {#if chips.length > 0}
            <div class="flex min-w-0 flex-wrap items-center gap-[0.3rem]">
                {#each chips as chip, index (`${chip.label}-${index}`)}
                    <span
                        class="inline-flex items-center gap-[0.3rem] rounded-full border bg-muted/50 px-[0.4rem] py-[0.18rem] text-[0.59rem] whitespace-nowrap text-muted-foreground"
                    >
                        <strong class="text-foreground">{chip.value}</strong> {chip.label}
                    </span>
                {/each}
            </div>
        {/if}
    </div>

    <DropdownMenu.Root>
        <DropdownMenu.Trigger>
            {#snippet child({ props })}
                <Button
                    {...props}
                    class="absolute top-[0.55rem] right-[0.45rem] size-[1.8rem] rounded-[0.35rem]"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={`More actions for ${stage.name}`}
                >
                    <EllipsisVerticalIcon aria-hidden="true" />
                </Button>
            {/snippet}
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="end" class="w-40">
            <DropdownMenu.Item onSelect={copyName}>
                <CopyIcon aria-hidden="true" /> Copy stage name
            </DropdownMenu.Item>
            <DropdownMenu.Item disabled>
                <FileTextIcon aria-hidden="true" /> View YAML
            </DropdownMenu.Item>
            <DropdownMenu.Item disabled>
                <ScrollTextIcon aria-hidden="true" /> Open logs
            </DropdownMenu.Item>
        </DropdownMenu.Content>
    </DropdownMenu.Root>
    <span class="sr-only" aria-live="polite">{actionAnnouncement}</span>
</article>
