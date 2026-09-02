import type { components } from "$lib/konfidence-api/schema";

type ApiLandscape = components["schemas"]["Landscape"];
type ApiStage = components["schemas"]["Stage"];
type ApiStageVersionStatus = NonNullable<ApiStage["targetStageVersion"]>["status"];
type StageStatus = "Active" | "DeploymentCreated" | "MigrationTasks";

interface Landscape {
  id: string;
  name: string;
}

interface Stage {
  generation: number;
  id: string;
  landscapeId: string;
  landscapeName: string;
  name: string;
  status: StageStatus;
  vector: string;
}

/* The prototype's phase UI predates the real StageVersion states, so map the
   backend enum down to the placeholder 3-phase model until the UI is redesigned. */
const toStageStatus = (status: ApiStageVersionStatus | undefined): StageStatus => {
  switch (status) {
    case "Ready": {
      return "Active";
    }
    case "ActivatingVector":
    case "MigratingVector": {
      return "MigrationTasks";
    }
    default: {
      return "DeploymentCreated";
    }
  }
};

const toLandscapeView = (
  landscapes: ApiLandscape[],
  stages: ApiStage[],
): { landscapes: Landscape[]; stages: Stage[] } => {
  const landscapeNames = new Map(landscapes.map((landscape) => [landscape.id, landscape.name]));
  return {
    landscapes: landscapes.map((landscape) => ({
      id: landscape.id,
      name: landscape.name,
    })),
    stages: stages.map((stage) => {
      // Both are absent while nothing is created or activated yet — prefer target, fall back to active.
      const version = stage.targetStageVersion ?? stage.activeStageVersion;
      return {
        generation: version?.stageGeneration ?? 0,
        id: stage.id,
        landscapeId: stage.landscapeId,
        landscapeName: landscapeNames.get(stage.landscapeId) ?? stage.landscapeId,
        name: stage.name,
        status: toStageStatus(version?.status),
        vector: version?.vector ?? "",
      };
    }),
  };
};

export { toLandscapeView };
export type { Landscape, Stage, StageStatus };
