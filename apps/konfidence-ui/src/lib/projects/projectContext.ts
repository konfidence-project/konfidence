import { createContext } from "svelte";
import type { components } from "@konfidence/api-client/schema";

type Project = components["schemas"]["Project"];

interface ProjectsContext {
  readonly projects: readonly Project[];
  readonly selectedProject?: Project;
  readonly status: ProjectsStatus;
  readonly error?: string;
  retry: () => Promise<void>;
  selectProject: (projectId: string) => Promise<void>;
  getEntryProject: () => Project | undefined;
}

type ProjectsStatus = "idle" | "loading" | "ready" | "error";

const [useProjects, provideProjects] = createContext<ProjectsContext>();

export { provideProjects, useProjects };
export type { Project, ProjectsStatus, ProjectsContext };
