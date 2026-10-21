import { resolve } from "$app/paths";

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

export { stageDetailsUrl };
export type { StageDetailsUrlOptions };
