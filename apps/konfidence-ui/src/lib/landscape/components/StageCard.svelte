<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { Stage } from "$lib/landscape/landscapeApi";
    import { stageStatuses } from "$lib/landscape/stageStatus";

    interface Props {
        stage: Stage;
        landscapeName: string;
        category?: string;
        href: string;
    }
    let { stage, landscapeName, category, href }: Props = $props();
    const statusId = $props.id();
    const target = $derived(stage.targetStageVersion);
    const active = $derived(stage.activeStageVersion);
    const targetIsActive = $derived(
        target !== undefined && target.id === active?.id,
    );
    const status = $derived(target ? stageStatuses[target.status] : undefined);
    const steps = [
        { label: "Deploy", status: "DeployingVector" },
        { label: "Migrate", status: "MigratingVector" },
        { label: "Activate", status: "ActivatingVector" },
    ] as const;
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- The route helper supplies the resolved details URL. -->
<a {href}
    class="box-border flex h-64 w-full flex-col gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 text-[var(--text-primary)] no-underline shadow-[var(--shadow-sm)] hover:border-[var(--border-focus)] focus-visible:outline-2 focus-visible:outline-offset-3 focus-visible:outline-[var(--border-focus)]"
    aria-label={`View details for stage ${stage.name} in ${landscapeName}${category ? `, ${category}` : ""}`}
    aria-describedby={statusId}
>
    <header class="flex items-baseline gap-3">
        <h4
            class="m-0 truncate text-[length:var(--text-body)] font-bold"
            title={stage.name}
        >
            {stage.name}
        </h4>
        {#if targetIsActive}<span class="ml-auto shrink-0"
                ><StatusBadge status="healthy">Active</StatusBadge></span
            >{/if}
    </header>
    <div class="flex min-h-14 flex-col gap-1">
        <span class="text-[length:var(--text-meta)] text-[var(--text-tertiary)]"
            >Target vector</span
        >
        {#if target}
            <span
                class="line-clamp-2 font-[family-name:var(--font-mono)] text-[length:var(--text-sm)] leading-normal [overflow-wrap:anywhere]"
                title={target.vector}>{target.vector}</span
            >
        {:else}
            <span
                class="text-[length:var(--text-sm)] text-[var(--text-secondary)]"
                >No target version yet</span
            >
        {/if}
    </div>
    <div class="flex flex-col gap-1">
        <span class="text-[length:var(--text-meta)] text-[var(--text-tertiary)]"
            >Active version</span
        >
        <span
            class="truncate text-[length:var(--text-sm)] text-[var(--text-secondary)]"
            title={active?.id}
            >{targetIsActive
                ? "Matches target"
                : (active?.id ?? "Nothing active yet")}</span
        >
    </div>
    <footer
        class="mt-auto flex flex-col items-start gap-2 border-t border-[var(--border-subtle)] pt-3"
        id={statusId}
    >
        <div
            class="flex w-full items-center justify-between gap-2"
            role="group"
            aria-label={status?.label ?? "No target"}
        >
            {#each steps as step (step.status)}
                {@const isCurrentPhase = target?.status === step.status}
                <span aria-current={isCurrentPhase ? "step" : undefined}>
                    <StatusBadge
                        status={isCurrentPhase ? "deploying" : "queued"}
                        showDot={isCurrentPhase}>{step.label}</StatusBadge
                    >
                </span>
            {/each}
        </div>
    </footer>
</a>
