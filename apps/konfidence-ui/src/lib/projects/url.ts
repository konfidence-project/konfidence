import { resolve } from "$app/paths";

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

export { projectLandscapeUrl, projectsUrl, stageDetailsUrl };
