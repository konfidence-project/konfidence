export type ConditionStatus = "True" | "False" | "Unknown";

export type StageConditionType =
  | "FetchFailed"
  | "VectorDeploymentCreated"
  | "VectorDeployed"
  | "VectorMigrated"
  | "Ready";

export type StageCondition = {
  type: StageConditionType;
  status: ConditionStatus;
  observedGeneration?: number;
  lastTransitionTime: string;
  reason: string;
  message: string;
};

export type TypedObjectReference = {
  apiGroup?: string;
  kind: string;
  name: string;
  namespace?: string;
};

export type Stage = {
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
};

export type StageList = {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  kind: "StageList";
  items: Stage[];
};
