interface ArtifactDeployment {
  apiVersion: "star.konfidence.cloud/v1alpha1";
  kind: "ArtifactDeployment";
  metadata: {
    creationTimestamp?: string;
    generation?: number;
    name: string;
    namespace: string;
  };
  spec: {
    component?: {
      name: string;
      resources?: {
        content: unknown;
        name: string;
        type: string;
      }[];
      version?: string;
    };
    manifest: {
      allowReuse: boolean;
      type: string;
    };
    taskManifests: {
      dependsOn?: string[];
      name: string;
      spec: unknown;
      type: string;
    }[];
  };
  status: {
    conditions?: {
      lastTransitionTime: string;
      message: string;
      observedGeneration?: number;
      reason: string;
      status: "True" | "False" | "Unknown";
      type:
        | "ArtifactFetched"
        | "ArtifactDeployed"
        | "AppHealthy"
        | "DeploymentResultCreated"
        | "Ready";
    }[];
  };
}

interface ArtifactSummary {
  createdAt: string;
  displayName: string;
  manifestType: string;
  reuse: string;
  status: string;
  version: string;
}

export { type ArtifactDeployment, type ArtifactSummary };
