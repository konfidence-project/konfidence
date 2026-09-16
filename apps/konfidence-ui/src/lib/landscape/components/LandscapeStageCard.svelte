<script lang="ts">
    import { StageCard } from "@konfidence/design-system/components";
    import type {
        StageCardStatusRole,
        StagePhaseItem,
    } from "@konfidence/design-system/components";
    import type { Stage } from "$lib/landscape/landscapeApi";
    import type { StageVersion } from "$lib/landscape/stageStatus";
    import { stageStatuses } from "$lib/landscape/stageStatus";

    interface Props {
        stage: Stage;
        landscapeName: string;
        category?: string;
        href: string;
        selected?: boolean;
    }
    let {
        stage,
        landscapeName,
        category,
        href,
        selected = false,
    }: Props = $props();

    const target = $derived(stage.targetStageVersion);
    const active = $derived(stage.activeStageVersion);
    const targetIsActive = $derived(
        target !== undefined && target.id === active?.id,
    );

    const statusRole: StageCardStatusRole = $derived.by((): StageCardStatusRole => {
        if (!target) {
            return "neutral";
        }
        switch (target.status) {
            case "Ready": {
                return targetIsActive ? "healthy" : "neutral";
            }
            case "DeployingVector":
            case "MigratingVector":
            case "ActivatingVector": {
                return "deploying";
            }
            case "Failed": {
                return "error";
            }
            case "PendingDeployment": {
                return "neutral";
            }
            default: {
                return "neutral";
            }
        }
    });

    type PhaseState = StagePhaseItem["state"];
    const buildPhases = (
        deploy: PhaseState,
        migrate: PhaseState,
        activate: PhaseState,
    ): StagePhaseItem[] => [
        { label: "Deploy", state: deploy },
        { label: "Migrate", state: migrate },
        { label: "Activate", state: activate },
    ];
    const phases: StagePhaseItem[] = $derived.by(() => {
        if (!target) {
            return buildPhases("pending", "pending", "pending");
        }
        switch (target.status) {
            case "PendingDeployment": {
                return buildPhases("pending", "pending", "pending");
            }
            case "DeployingVector": {
                return buildPhases("active", "pending", "pending");
            }
            case "MigratingVector": {
                return buildPhases("done", "active", "pending");
            }
            case "ActivatingVector": {
                return buildPhases("done", "done", "active");
            }
            case "Ready": {
                return buildPhases("done", "done", "done");
            }
            case "Failed": {
                // The API does not report which phase failed; infer from the
                // presence of an active version.
                return active
                    ? buildPhases("done", "done", "failed")
                    : buildPhases("failed", "pending", "pending");
            }
            default: {
                return buildPhases("pending", "pending", "pending");
            }
        }
    });

    const statusLabel = (version: StageVersion): string =>
        stageStatuses[version.status].label;

    const phaseAriaLabel = $derived(target ? statusLabel(target) : "No target");
    const activeVersionText = $derived(
        targetIsActive
            ? "Matches target"
            : (active?.id ?? "Nothing active yet"),
    );
    const ariaLabel = $derived(
        `View details for stage ${stage.name} in ${landscapeName}${category ? `, ${category}` : ""}`,
    );
</script>

<StageCard
    title={stage.name}
    {href}
    {ariaLabel}
    {statusRole}
    {phases}
    {phaseAriaLabel}
    targetVector={target?.vector}
    {activeVersionText}
    live={targetIsActive}
    {selected}
/>
