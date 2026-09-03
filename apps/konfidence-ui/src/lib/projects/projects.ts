import { getContext, setContext } from "svelte";
import type { components } from "$lib/konfidence-api/schema";

type Project = components["schemas"]["Project"];

interface ProjectsContext {
  readonly projects: readonly Project[];
  /** First-project default until real project selection lands. */
  readonly selectedProjectId: string;
}

const KEY = Symbol("konfidence-projects");

const provideProjects = (context: ProjectsContext): ProjectsContext => {
  setContext(KEY, context);
  return context;
};

const useProjects = (): ProjectsContext => {
  const context = getContext<ProjectsContext | undefined>(KEY);
  if (!context) {
    throw new Error(
      "useProjects() called outside a projects context. Ensure provideProjects() runs in (shell)/+layout.svelte before children read it.",
    );
  }
  return context;
};

export { provideProjects, useProjects };
export type { Project, ProjectsContext };
