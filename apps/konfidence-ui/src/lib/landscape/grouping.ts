import type { Stage } from "$lib/landscape/api";

type StageGroups = Record<"dev" | "test" | "prod" | "other", Stage[]>;

const groupStages = (stages: readonly Stage[]): StageGroups =>
  stages.reduce<StageGroups>(
    (groups, stage) => {
      const prefix = stage.name
        .match(/^(?<category>dev|test|prod)-/i)
        ?.groups?.category?.toLowerCase();
      const category =
        prefix === "dev" || prefix === "test" || prefix === "prod" ? prefix : "other";
      groups[category].push(stage);
      return groups;
    },
    { dev: [], other: [], prod: [], test: [] },
  );

export { groupStages };
