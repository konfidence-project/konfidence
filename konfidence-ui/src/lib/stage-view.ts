import type { Stage, StageCondition, StageConditionType } from "$lib/stages.js";

export type StageHealth = "healthy" | "deploying" | "warning" | "error";

export type StagePhaseState = "done" | "cur" | "err" | "idle";

export type StagePhase = {
  id: StageConditionType | "Tasks";
  label: string;
  state: StagePhaseState;
  reason?: string;
  message?: string;
};

export type StageChip = {
  label: string;
  value: string | number;
  tone?: "" | "info" | "warn" | "alert";
};

export type VectorParts = {
  version: string;
  hash?: string;
};

const findCondition = (stage: Stage, type: StageConditionType): StageCondition | undefined =>
  stage.status.conditions?.find((condition) => condition.type === type);

export const getStageHealth = (stage: Stage): StageHealth => {
  const ready = findCondition(stage, "Ready");
  const fetchFailed = findCondition(stage, "FetchFailed");

  if (fetchFailed?.status === "True") return "error";
  if (ready?.status === "False") return "error";
  if (ready?.status === "True") return "healthy";

  // No Ready=True yet, but something is progressing.
  const inFlight = stage.status.conditions?.some(
    (c) =>
      c.type !== "FetchFailed" &&
      c.type !== "Ready" &&
      (c.status === "True" || c.status === "Unknown"),
  );
  if (inFlight) return "deploying";

  return "warning";
};

export const getStageStatusLabel = (stage: Stage): { label: string; tone: StageHealth } => {
  const tone = getStageHealth(stage);
  const label =
    tone === "healthy"
      ? "Live"
      : tone === "deploying"
        ? "Deploying"
        : tone === "error"
          ? "Failed"
          : "Pending";
  return { label, tone };
};

const phaseStateFrom = (condition: StageCondition | undefined, invert = false): StagePhaseState => {
  if (!condition) return "idle";
  const value = condition.status;
  if (invert) {
    if (value === "True") return "err";
    if (value === "False") return "done";
    return "cur";
  }
  if (value === "True") return "done";
  if (value === "False") return "err";
  return "cur";
};

export const getPhases = (stage: Stage): StagePhase[] => {
  const deploy = findCondition(stage, "VectorDeploymentCreated");
  const tasks = findCondition(stage, "VectorDeployed");
  const migrate = findCondition(stage, "VectorMigrated");
  const active = findCondition(stage, "Ready");
  const fetchFailed = findCondition(stage, "FetchFailed");

  // Fetch failure short-circuits every downstream phase into an error state.
  const deployState = fetchFailed?.status === "True" ? "err" : phaseStateFrom(deploy);

  return [
    {
      id: "VectorDeploymentCreated",
      label: "Deploy",
      state: deployState,
      reason: (fetchFailed?.status === "True" ? fetchFailed : deploy)?.reason,
      message: (fetchFailed?.status === "True" ? fetchFailed : deploy)?.message,
    },
    {
      id: "Tasks",
      label: "Tasks",
      state: phaseStateFrom(tasks),
      reason: tasks?.reason,
      message: tasks?.message,
    },
    {
      id: "VectorMigrated",
      label: "Migrate",
      state: phaseStateFrom(migrate),
      reason: migrate?.reason,
      message: migrate?.message,
    },
    {
      id: "Ready",
      label: "Active",
      state: phaseStateFrom(active),
      reason: active?.reason,
      message: active?.message,
    },
  ];
};

/**
 * Parse a vector reference into a `version + hash` pair, mirroring the visual
 * split in the design mockup (e.g. `v2.14.0` + `a3f2c9`).
 *
 * Accepts common shapes we see today:
 *   - `github.com/org/repo:1.18.3`        → { version: "v1.18.3" }
 *   - `github.com/org/repo:1.18.3-a3f2c9` → { version: "v1.18.3", hash: "a3f2c9" }
 *   - `v2.14.0@a3f2c9`                    → { version: "v2.14.0", hash: "a3f2c9" }
 *   - bare `1.2.3`                         → { version: "v1.2.3" }
 */
export const splitVector = (vector: string): VectorParts => {
  const trimmed = vector.trim();
  if (!trimmed) return { version: "—" };

  // Take the trailing tag after the last ':' if present, otherwise the whole thing.
  const tag = trimmed.includes(":") ? (trimmed.split(":").at(-1) ?? trimmed) : trimmed;

  // Split off an optional hash suffix delimited by '@' or a '-' followed by hex.
  let version = tag;
  let hash: string | undefined;

  const atSplit = tag.split("@");
  if (atSplit.length === 2 && atSplit[1]) {
    version = atSplit[0];
    hash = atSplit[1];
  } else {
    const hashMatch = tag.match(/^(.+?)-([0-9a-f]{6,})$/i);
    if (hashMatch) {
      version = hashMatch[1];
      hash = hashMatch[2];
    }
  }

  if (/^\d/.test(version)) {
    version = `v${version}`;
  }

  return { version, hash };
};

export const getChips = (stage: Stage): StageChip[] => {
  const chips: StageChip[] = [];
  const historyCount = stage.status.vectorHistory?.length ?? 0;
  chips.push({
    label: historyCount === 1 ? "version" : "versions",
    value: historyCount + 1, // include current vector
  });

  const latest = stage.status.latestVectorDeploymentRef?.name;
  if (latest) {
    chips.push({
      label: "deployment",
      value: latest.length > 22 ? `${latest.slice(0, 20)}…` : latest,
    });
  }

  const fetchFailed = findCondition(stage, "FetchFailed");
  if (fetchFailed?.status === "True") {
    chips.push({ label: "fetch failed", value: "!", tone: "alert" });
  }

  return chips;
};

export const getLandscapeLabel = (stage: Stage): string => stage.metadata.namespace.toUpperCase();
