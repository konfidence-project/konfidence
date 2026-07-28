<script lang="ts">
    import CopyIcon from "@lucide/svelte/icons/copy";
    import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
    import FileCodeIcon from "@lucide/svelte/icons/file-code";
    import ScrollTextIcon from "@lucide/svelte/icons/scroll-text";
    import { Badge } from "$lib/components/ui/badge/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as Card from "$lib/components/ui/card/index.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import type { Stage } from "$lib/stages.js";
    import { getChips, getPhases, getStageStatusLabel, splitVector } from "../utils/stage-view.js";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
    const status = $derived(getStageStatusLabel(stage));
    const phases = $derived(getPhases(stage));
    const chips = $derived(getChips(stage));
    const vector = $derived(splitVector(stage.vector));

    const copyName = async (): Promise<void> => {
        await globalThis.navigator?.clipboard?.writeText(stage.name);
    };
</script>

<Card.Root
    class={[
        "stage-card relative w-80 gap-0 overflow-hidden border-border bg-card p-0 text-card-foreground shadow-[0_0.35rem_1.2rem_color-mix(in_oklch,var(--foreground)_8%,transparent)]",
        selected &&
            "border-primary shadow-[0_0_0_2px_color-mix(in_oklch,var(--primary)_25%,transparent)]",
    ]}
    data-health={status.tone}
    role="article"
    aria-label={`Stage ${stage.name}`}
>
    <span
        class={["h-[0.24rem] bg-information", status.tone === "healthy" && "bg-success"]}
        aria-hidden="true"
    ></span>
    <Card.Header class="relative gap-[0.65rem] px-[0.9rem] pt-[0.85rem] pb-[0.7rem]">
        <div class="flex items-start justify-between gap-2 pr-[1.4rem]">
            <div class="min-w-0">
                <Card.Title class="truncate text-[0.95rem]">{stage.name}</Card.Title>
                <Card.Description class="mt-[0.1rem] text-[0.68rem] tracking-[0.06em] uppercase">
                    {stage.landscapeName}
                </Card.Description>
            </div>
            <Badge
                variant="outline"
                class={[
                    "gap-[0.3rem] border-current text-[0.62rem] tracking-[0.05em] uppercase",
                    status.tone === "healthy"
                        ? "bg-success-background text-success"
                        : "bg-information-background text-information",
                ]}
            >
                <span
                    class={[
                        "size-[0.38rem] rounded-full bg-current",
                        status.tone === "deploying" &&
                            "animate-status-pulse motion-reduce:animate-none",
                    ]}
                ></span>
                {status.label}
            </Badge>
        </div>
        <div class="flex min-w-0 items-center gap-[0.35rem] text-[0.68rem] text-muted-foreground uppercase">
            <span>Vector</span>
            <code class="truncate rounded-[0.3rem] border bg-muted px-[0.3rem] py-[0.1rem] text-[0.72rem] text-foreground normal-case">
                {vector.version}
            </code>
            {#if vector.hash}
                <code
                    class="truncate rounded-[0.3rem] border bg-muted px-[0.3rem] py-[0.1rem] text-[0.72rem] text-muted-foreground normal-case"
                >{vector.hash}</code>
            {/if}
        </div>
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <Button
                        {...props}
                        class="absolute top-[0.55rem] right-[0.55rem]"
                        variant="ghost"
                        size="icon-xs"
                        aria-label={`Actions for stage ${stage.name}`}
                    >
                        <EllipsisIcon />
                    </Button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end" class="w-44">
                <DropdownMenu.Item onSelect={copyName}><CopyIcon /> Copy stage name</DropdownMenu.Item>
                <DropdownMenu.Item><FileCodeIcon /> View YAML</DropdownMenu.Item>
                <DropdownMenu.Item><ScrollTextIcon /> Open logs</DropdownMenu.Item>
            </DropdownMenu.Content>
        </DropdownMenu.Root>
    </Card.Header>

    <Card.Content class="px-[0.9rem] pt-0 pb-[0.85rem]">
        <div class="mb-[0.3rem] text-[0.62rem] font-bold tracking-[0.06em] text-muted-foreground uppercase">
            Deployment phases
        </div>
        <ol class="grid list-none grid-cols-3 gap-1 p-0">
            {#each phases as phase (phase.id)}
                <li
                    class={[
                        "grid gap-[0.22rem] text-center text-[0.58rem] text-muted-foreground uppercase",
                        phase.state === "done" && "text-success",
                        phase.state === "cur" && "font-bold text-information",
                    ]}
                    title={phase.label}
                >
                    <span
                        class={[
                            "h-[0.22rem] rounded-full bg-border",
                            phase.state === "done" && "bg-success opacity-65",
                            phase.state === "cur" && "bg-information",
                        ]}
                        aria-hidden="true"
                    ></span>
                    <span>{phase.label}<span class="sr-only">: {phase.state}</span></span>
                </li>
            {/each}
        </ol>
        <div class="mt-[0.6rem] flex flex-wrap gap-[0.3rem]">
            {#each chips as chip, index (`${chip.label}-${index}`)}
                <Badge class="text-[0.65rem]" variant="outline">
                    <strong>{chip.value}</strong> {chip.label}
                </Badge>
            {/each}
        </div>
    </Card.Content>
</Card.Root>
