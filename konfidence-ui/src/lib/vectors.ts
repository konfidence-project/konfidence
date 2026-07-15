type VectorHealth = "Healthy" | "Warning" | "Error";

interface Vector {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  kind: "Vector";
  metadata: {
    name: string;
    namespace: string;
    creationTimestamp: string;
    generation: number;
  };
  spec: {
    vector: string;
    hash: string;
    artifactCount: number;
    deployedOn: string[];
  };
  status: {
    health: VectorHealth;
  };
}

interface VectorList {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  items: Vector[];
  kind: "VectorList";
}

export type { Vector, VectorHealth, VectorList };
