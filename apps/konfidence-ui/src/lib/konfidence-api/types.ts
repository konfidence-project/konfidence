import type { components } from "@konfidence/api-client/schema";

type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];
type StageVersion = components["schemas"]["StageVersion"];
type Project = components["schemas"]["Project"];
type Identity = components["schemas"]["Identity"];
type VectorDeployment = components["schemas"]["VectorDeployment"];
type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type VectorPromotionConfig = components["schemas"]["VectorPromotionConfig"];

export type {
  ArtifactDeployment,
  Identity,
  Landscape,
  Project,
  Stage,
  StageVersion,
  VectorDeployment,
  VectorPromotionConfig,
};
