import type { Stage } from "$lib/landscape/api";

type StageVersion = NonNullable<Stage["targetStageVersion"]>;

// Mirrors internal/stage/state.go; Failed is reserved and not emitted by the backend yet.
const stageStatuses: Record<StageVersion["status"], { label: string; badge: string }> = {
  ActivatingVector: { badge: "deploying", label: "Activating vector" },
  DeployingVector: { badge: "deploying", label: "Deploying vector" },
  Failed: { badge: "error", label: "Failed" },
  MigratingVector: { badge: "deploying", label: "Migrating vector" },
  PendingDeployment: { badge: "queued", label: "Pending deployment" },
  Ready: { badge: "healthy", label: "Ready" },
};

export { stageStatuses };
export type { StageVersion };
