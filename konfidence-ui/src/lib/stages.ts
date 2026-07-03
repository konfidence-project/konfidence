type ConditionStatus = "True" | "False" | "Unknown";

type StageConditionType =
  | "FetchFailed"
  | "VectorDeploymentCreated"
  | "VectorDeployed"
  | "VectorMigrated"
  | "Ready";

interface StageCondition {
  lastTransitionTime: string;
  message: string;
  observedGeneration?: number;
  reason: string;
  status: ConditionStatus;
  type: StageConditionType;
}

interface TypedObjectReference {
  apiGroup?: string;
  kind: string;
  name: string;
  namespace?: string;
}

interface Stage {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  kind: "Stage";
  metadata: {
    name: string;
    namespace: string;
    creationTimestamp: string;
    generation: number;
  };
  spec: {
    vector: string;
  };
  status: {
    conditions?: StageCondition[];
    vectorHistory?: string[];
    latestVectorDeploymentRef?: TypedObjectReference;
  };
}

interface StageList {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  items: Stage[];
  kind: "StageList";
}

export type {
  ConditionStatus,
  Stage,
  StageCondition,
  StageConditionType,
  StageList,
  TypedObjectReference,
};
