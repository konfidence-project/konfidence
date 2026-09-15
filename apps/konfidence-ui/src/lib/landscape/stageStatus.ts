import type { Stage } from "$lib/landscape/landscapeApi";

type StageVersion = NonNullable<Stage["targetStageVersion"]>;

// Badge tones this feature relies on. Keep in sync with the `.badge--<tone>`
// classes exposed by `<StatusBadge>` in the design system.
type StageBadgeTone = "deploying" | "error" | "healthy" | "queued";

interface StageStatusEntry {
  badge: StageBadgeTone;
  label: string;
}

// Mirrors internal/stage/state.go; Failed is reserved and not emitted by the backend yet.
const stageStatuses: Record<StageVersion["status"], StageStatusEntry> = {
  ActivatingVector: { badge: "deploying", label: "Activating vector" },
  DeployingVector: { badge: "deploying", label: "Deploying vector" },
  Failed: { badge: "error", label: "Failed" },
  MigratingVector: { badge: "deploying", label: "Migrating vector" },
  PendingDeployment: { badge: "queued", label: "Pending deployment" },
  Ready: { badge: "healthy", label: "Ready" },
};

export { stageStatuses };
export type { StageBadgeTone, StageStatusEntry, StageVersion };
