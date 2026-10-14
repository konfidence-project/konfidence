import { resolve } from "$app/paths";

import { STAGE_PARAM } from "$lib/landscape/params";

const projectLandscapeUrl = (projectId: string, embedded: boolean): string => {
  const path = resolve("/(shell)/projects/[projectId]/landscape", {
    projectId,
  });
  return embedded ? `${path}?embedded=1` : path;
};

const projectsUrl = (embedded: boolean): string => {
  const path = resolve("/(shell)/projects");
  return embedded ? `${path}?embedded=1` : path;
};

interface StageOverlayUrlOptions {
  projectId: string;
  stageId: string;
  embedded: boolean;
}

const stageOverlayUrl = ({ projectId, stageId, embedded }: StageOverlayUrlOptions): string => {
  const base = projectLandscapeUrl(projectId, embedded);
  const url = new URL(base, "http://x");
  url.searchParams.set(STAGE_PARAM, stageId);
  return `${url.pathname}?${url.searchParams.toString()}`;
};

interface StageDetailsUrlOptions {
  projectId: string;
  landscapeId: string;
  stageId: string;
  embedded: boolean;
}

const stageDetailsUrl = ({
  projectId,
  landscapeId,
  stageId,
  embedded,
}: StageDetailsUrlOptions): string => {
  const path = resolve("/(shell)/projects/[projectId]/landscape/[landscapeId]/stages/[stageId]", {
    landscapeId,
    projectId,
    stageId,
  });
  return embedded ? `${path}?embedded=1` : path;
};

export { projectLandscapeUrl, projectsUrl, stageDetailsUrl, stageOverlayUrl };
