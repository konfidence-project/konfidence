import type * as stages from "$lib/stages";
import { getRequestEvent, query } from "$app/server";
import { error } from "@sveltejs/kit";

interface ApiStage {
  conditions: stages.StageCondition[];
  name: string;
  namespace: string;
  vector: string;
}

interface ApiStageList {
  items: ApiStage[];
}

const HTTP_UNAUTHORIZED = 401;
const HTTP_FORBIDDEN = 403;

const accessError = (status: number) => {
  if (status === HTTP_UNAUTHORIZED) {
    return {
      accessError: {
        message: "Your session is missing or has expired. Sign in again to view stages.",
        status,
        title: "Sign-in required",
      },
      items: [],
    };
  }
  return {
    accessError: {
      message: "You are signed in, but your account does not have permission to view stages.",
      status,
      title: "Access denied",
    },
    items: [],
  };
};

const toStage = (stage: ApiStage): stages.Stage => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    creationTimestamp: "",
    generation: 0,
    name: stage.name,
    namespace: stage.namespace,
  },
  spec: { vector: stage.vector },
  status: { conditions: stage.conditions },
});

export const getStages = query(async () => {
  const { fetch } = getRequestEvent();
  const response = await fetch("/api/v1/stages");

  if (response.status === HTTP_UNAUTHORIZED || response.status === HTTP_FORBIDDEN) {
    return accessError(response.status);
  }
  if (!response.ok) {
    error(response.status, "Failed to load stages");
  }

  const stages = (await response.json()) as ApiStageList;
  return {
    accessError: undefined,
    apiVersion: "star.konfidence.cloud/v1alpha1",
    items: stages.items.map(toStage),
    kind: "StageList",
  } satisfies stages.StageList & { accessError?: undefined };
});
