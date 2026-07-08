import { type ServerLoad, redirect } from "@sveltejs/kit";

import getIdpConfig from "$lib/server/idp-config";

const HTTP_SEE_OTHER = 303;

export const load: ServerLoad = ({ locals, url }) => {
  if (!locals.user) {
    const redirectTo = url.pathname + url.search;
    redirect(HTTP_SEE_OTHER, `/login?redirectTo=${encodeURIComponent(redirectTo)}`);
  }

  return {
    idpType: getIdpConfig().type,
    user: locals.user,
  };
};
