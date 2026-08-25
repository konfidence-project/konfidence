import type { components } from "$lib/konfidence-api/schema";

type ApiLandscape = components["schemas"]["Landscape"];
type ApiStage = components["schemas"]["Stage"];
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
    stages: stages.map((stage) => ({
      generation: stage.targetStageVersion.stageGeneration,
      id: stage.id,
      landscapeId: stage.landscapeId,
      landscapeName: landscapeNames.get(stage.landscapeId) ?? stage.landscapeId,
      name: stage.name,
      status: stage.targetStageVersion.status,
      vector: stage.targetStageVersion.vector,
    })),
  };
};

export { toLandscapeView };
export type { Landscape, Stage, StageStatus };
