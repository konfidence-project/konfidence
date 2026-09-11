import { resolve } from "$app/paths";

const projectLandscapeUrl = (projectId: string, embedded: boolean): string => {
  const path = resolve("/(shell)/projects/[projectId]/landscape", { projectId });
  return embedded ? `${path}?embedded=1` : path;
};

const projectsUrl = (embedded: boolean): string => {
  const path = resolve("/(shell)/projects");
  return embedded ? `${path}?embedded=1` : path;
};

export { projectLandscapeUrl, projectsUrl };
