import type { Stage } from "$lib/stages";

type StageHealth = "deploying";
type StagePhaseState = "cur" | "done" | "idle";
type StagePhaseId = "Active" | "DeploymentCreated" | "MigrationTasks";

interface StageChip {
  label: string;
  tone?: "" | "alert" | "info" | "warn";
  value: number | string;
}

interface StagePhase {
  id: StagePhaseId;
  label: string;
  message?: string;
  reason?: string;
  state: StagePhaseState;
}

interface VectorParts {
  hash?: string;
  version: string;
}

const STATUS_ORDER: StagePhaseId[] = ["DeploymentCreated", "MigrationTasks", "Active"];
const STATUS_LABEL: Record<StagePhaseId, string> = {
  Active: "Active",
  DeploymentCreated: "Deploy",
  MigrationTasks: "Tasks",
};
const HASH_SUFFIX_PATTERN = /^(?<version>.+?)-(?<hash>[0-9a-f]{6,})$/i;
const NUMERIC_VERSION_PATTERN = /^\d/;

const getStageHealth = (_stage: Stage): StageHealth => "deploying";

const getStageStatusLabel = (stage: Stage): { label: "Deploying"; tone: StageHealth } => ({
  label: "Deploying",
  tone: getStageHealth(stage),
});

const getPhases = (stage: Stage): StagePhase[] => {
  const currentIndex = STATUS_ORDER.indexOf(stage.status);
  return STATUS_ORDER.map((status, index) => {
    let state: StagePhaseState = "idle";
    if (index < currentIndex) {
      state = "done";
    } else if (index === currentIndex) {
      state = "cur";
    }
    return { id: status, label: STATUS_LABEL[status], state };
  });
};

const getChips = (stage: Stage): StageChip[] => [{ label: "generation", value: stage.generation }];

const getLandscapeLabel = (stage: Stage): string => stage.landscapeName.toUpperCase();

const vectorTag = (vector: string): string => {
  if (!vector.includes(":")) {
    return vector;
  }
  return vector.split(":").at(-1) ?? vector;
};

const normalizeVersion = (version: string): string => {
  if (NUMERIC_VERSION_PATTERN.test(version)) {
    return `v${version}`;
  }
  return version;
};

const toVectorParts = (version: string, hash?: string): VectorParts => {
  if (hash) {
    return { hash, version };
  }
  return { version };
};

const splitVector = (vector: string): VectorParts => {
  const trimmed = vector.trim();
  if (!trimmed) {
    return { version: "—" };
  }

  const tag = vectorTag(trimmed);
  const [rawVersion, digest] = tag.split("@");
  const match = rawVersion.match(HASH_SUFFIX_PATTERN);
  const version = normalizeVersion(match?.groups?.version ?? rawVersion);
  const hash = digest ?? match?.groups?.hash;
  return toVectorParts(version, hash);
};

export { getChips, getLandscapeLabel, getPhases, getStageHealth, getStageStatusLabel, splitVector };
export type { StageChip, StageHealth, StagePhase, StagePhaseState, VectorParts };
