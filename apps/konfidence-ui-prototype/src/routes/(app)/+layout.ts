import { redirect } from "@sveltejs/kit";
import { getIdentity, getProjects } from "$lib/konfidence-api/queries.svelte";
import type { AuthRole, AuthUser } from "$lib/auth/types";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import type { LayoutLoad } from "./$types";

const AUTH_ROLES = ["ADMIN", "DEV", "PM"] as const;

export const load: LayoutLoad = async ({ url }) => {
  const identity = await getIdentity();
  const user: AuthUser = {
    ...identity,
    roles: identity.roles.filter((role): role is AuthRole => AUTH_ROLES.includes(role as AuthRole)),
  };
  const projects = await getProjects();
  if (url.pathname === "/") {
    if (projects.length === 1) {
      redirect(HTTP_SEE_OTHER, `/projects/${projects[0].id}/landscape`);
    }
    redirect(HTTP_SEE_OTHER, "/projects");
  }
  return { projects, user };
};
