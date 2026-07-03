import type { Stage, StageCondition, StageConditionType } from "$lib/stages.js";

type StageHealth = "healthy" | "deploying" | "warning" | "error";

type StagePhaseState = "done" | "cur" | "err" | "idle";

interface StagePhase {
  id: StageConditionType | "Tasks";
  label: string;
  message?: string;
  reason?: string;
  state: StagePhaseState;
}

interface StageChip {
  label: string;
  tone?: "" | "info" | "warn" | "alert";
  value: string | number;
}

interface VectorParts {
  hash?: string;
  version: string;
}

const DEPLOYMENT_TRUNCATE_LENGTH = 20;
const DEPLOYMENT_MAX_LENGTH = 22;
const HASH_SUFFIX_PATTERN = /^(?<version>.+?)-(?<hash>[0-9a-f]{6,})$/i;

const findCondition = (stage: Stage, type: StageConditionType): StageCondition | undefined =>
  stage.status.conditions?.find((condition) => condition.type === type);

const isInFlightCondition = (condition: StageCondition) =>
  condition.type !== "FetchFailed" &&
  condition.type !== "Ready" &&
  (condition.status === "True" || condition.status === "Unknown");

const stageHealthFromReady = (stage: Stage, ready: StageCondition | undefined): StageHealth => {
  if (ready?.status === "False") {
    return "error";
  }
  if (ready?.status === "True") {
    return "healthy";
  }
  if (stage.status.conditions?.some(isInFlightCondition)) {
    return "deploying";
  }

  return "warning";
};

const getStageHealth = (stage: Stage): StageHealth => {
  const fetchFailed = findCondition(stage, "FetchFailed");
  if (fetchFailed?.status === "True") {
    return "error";
  }

  return stageHealthFromReady(stage, findCondition(stage, "Ready"));
};

const STATUS_LABELS: Record<StageHealth, string> = {
  deploying: "Deploying",
  error: "Failed",
  healthy: "Live",
  warning: "Pending",
};

const getStageStatusLabel = (stage: Stage): { label: string; tone: StageHealth } => {
  const tone = getStageHealth(stage);
  return { label: STATUS_LABELS[tone], tone };
};

const PHASE_STATE: Record<StageCondition["status"], StagePhaseState> = {
  False: "err",
  True: "done",
  Unknown: "cur",
};

const phaseStateFrom = (condition: StageCondition | undefined): StagePhaseState => {
  if (!condition) {
    return "idle";
  }
  return PHASE_STATE[condition.status];
};

const deployCondition = (
  deploy: StageCondition | undefined,
  fetchFailed: StageCondition | undefined,
) => {
  if (fetchFailed?.status === "True") {
    return fetchFailed;
  }
  return deploy;
};

const deployPhaseState = (
  deploy: StageCondition | undefined,
  fetchFailed: StageCondition | undefined,
): StagePhaseState => {
  if (fetchFailed?.status === "True") {
    return "err";
  }
  return phaseStateFrom(deploy);
};

const getPhases = (stage: Stage): StagePhase[] => {
  const deploy = findCondition(stage, "VectorDeploymentCreated");
  const tasks = findCondition(stage, "VectorDeployed");
  const migrate = findCondition(stage, "VectorMigrated");
  const active = findCondition(stage, "Ready");
  const fetchFailed = findCondition(stage, "FetchFailed");
  const deployStatus = deployCondition(deploy, fetchFailed);

  return [
    {
      id: "VectorDeploymentCreated",
      label: "Deploy",
      message: deployStatus?.message,
      reason: deployStatus?.reason,
      state: deployPhaseState(deploy, fetchFailed),
    },
    {
      id: "Tasks",
      label: "Tasks",
      message: tasks?.message,
      reason: tasks?.reason,
      state: phaseStateFrom(tasks),
    },
    {
      id: "VectorMigrated",
      label: "Migrate",
      message: migrate?.message,
      reason: migrate?.reason,
      state: phaseStateFrom(migrate),
    },
    {
      id: "Ready",
      label: "Active",
      message: active?.message,
      reason: active?.reason,
      state: phaseStateFrom(active),
    },
  ];
};

const vectorTag = (trimmed: string): string => {
  if (trimmed.includes(":")) {
    return trimmed.split(":").at(-1) ?? trimmed;
  }
  return trimmed;
};

const splitHash = (tag: string): VectorParts => {
  const [version, hash] = tag.split("@");
  if (hash) {
    return { hash, version };
  }

  const hashMatch = tag.match(HASH_SUFFIX_PATTERN);
  if (hashMatch?.groups) {
    return {
      hash: hashMatch.groups.hash,
      version: hashMatch.groups.version,
    };
  }

  return { version: tag };
};

const normalizeVersion = (parts: VectorParts): VectorParts => {
  if (/^\d/.test(parts.version)) {
    return { hash: parts.hash, version: `v${parts.version}` };
  }

  return parts;
};

const splitVector = (vector: string): VectorParts => {
  const trimmed = vector.trim();
  if (!trimmed) {
    return { version: "—" };
  }
  return normalizeVersion(splitHash(vectorTag(trimmed)));
};

const versionLabel = (historyCount: number): string => {
  if (historyCount === 1) {
    return "version";
  }
  return "versions";
};

const deploymentValue = (latest: string): string => {
  if (latest.length > DEPLOYMENT_MAX_LENGTH) {
    return `${latest.slice(0, DEPLOYMENT_TRUNCATE_LENGTH)}…`;
  }

  return latest;
};

const getChips = (stage: Stage): StageChip[] => {
  const chips: StageChip[] = [];
  const historyCount = stage.status.vectorHistory?.length ?? 0;
  chips.push({
    label: versionLabel(historyCount),
    value: historyCount + 1,
  });

  const latest = stage.status.latestVectorDeploymentRef?.name;
  if (latest) {
    chips.push({
      label: "deployment",
      value: deploymentValue(latest),
    });
  }

  const fetchFailed = findCondition(stage, "FetchFailed");
  if (fetchFailed?.status === "True") {
    chips.push({ label: "fetch failed", tone: "alert", value: "!" });
  }

  return chips;
};

const getLandscapeLabel = (stage: Stage): string => stage.metadata.namespace.toUpperCase();

export { getChips, getLandscapeLabel, getPhases, getStageHealth, getStageStatusLabel, splitVector };

export type { StageChip, StageHealth, StagePhase, StagePhaseState, VectorParts };
