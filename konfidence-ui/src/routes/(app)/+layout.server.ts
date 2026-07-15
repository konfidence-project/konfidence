import { type ServerLoad, error, redirect } from "@sveltejs/kit";

const HTTP_SEE_OTHER = 303;
const HTTP_UNAUTHORIZED = 401;
const HTTP_BAD_GATEWAY = 502;

interface ApiIdentity {
  email?: string;
  name?: string;
  subject: string;
  username: string;
}

export const load: ServerLoad = async ({ fetch, url }) => {
  const response = await fetch("/api/identity");
  if (response.status === HTTP_UNAUTHORIZED) {
    const redirectTo = url.pathname + url.search;
    redirect(HTTP_SEE_OTHER, `/api/login?returnTo=${encodeURIComponent(redirectTo)}`);
  }
  if (!response.ok) {
    error(HTTP_BAD_GATEWAY, "Failed to load the signed-in identity");
  }

  const identity = (await response.json()) as ApiIdentity;
  return {
    user: {
      email: identity.email,
      name: identity.name || identity.username,
      sub: identity.subject,
    },
  };
};
