import { type Actions, type ServerLoad, error, redirect } from "@sveltejs/kit";
import { SESSION_COOKIE_NAME, SESSION_TTL_MS, createSession } from "$lib/server/session";
import { dev } from "$app/environment";
import generateSillyName from "$lib/server/silly-name";
import getIdpConfig from "$lib/server/idp-config";

const HTTP_SEE_OTHER = 303;
const HTTP_NOT_IMPLEMENTED = 501;
const MS_PER_SECOND = 1000;

/** Where signed-in users land when no explicit redirectTo is supplied. */
const DEFAULT_POST_LOGIN_TARGET = "/landscape";

/**
 * Only allow relative redirects that stay on this origin — never trust the
 * value from the URL directly.
 */
const safeRedirectTarget = (input: string | null | undefined): string => {
  if (!input) {
    return DEFAULT_POST_LOGIN_TARGET;
  }
  if (!input.startsWith("/")) {
    return DEFAULT_POST_LOGIN_TARGET;
  }
  if (input.startsWith("//")) {
    return DEFAULT_POST_LOGIN_TARGET;
  }
  return input;
};

const slugify = (value: string): string => value.toLowerCase().replaceAll(/[^a-z0-9]+/g, "-");

/**
 * If the visitor is already signed in, bounce them straight into the app —
 * hitting `/login` while authenticated should not force them to re-login.
 * Also surfaces the currently active IdP type to the login page so it can
 * label the sign-in button accordingly.
 */
export const load: ServerLoad = ({ locals, url }) => {
  if (locals.user) {
    const target = safeRedirectTarget(url.searchParams.get("redirectTo"));
    redirect(HTTP_SEE_OTHER, target);
  }

  const idp = getIdpConfig();

  return {
    idpType: idp.type,
    redirectTo: safeRedirectTarget(url.searchParams.get("redirectTo")),
  };
};

export const actions: Actions = {
  default: async ({ cookies, request, url }) => {
    const formData = await request.formData();
    const redirectTo = safeRedirectTarget(
      (formData.get("redirectTo") as string | null) ?? url.searchParams.get("redirectTo"),
    );

    const idp = getIdpConfig();

    if (idp.type === "oidc") {
      // TODO: kick off the OIDC authorization-code flow here (redirect to
      //       `${idp.issuer}/authorize?...`). Not implemented yet.
      error(
        HTTP_NOT_IMPLEMENTED,
        "OIDC login is not implemented yet. Set IDP_TYPE=stub for local development.",
      );
    }

    // Stub identity — replaced by claims from the real IdP once OIDC ships.
    const name = generateSillyName();
    const sub = `stub:${slugify(name)}:${Date.now()}`;

    const sid = createSession({ name, sub });

    cookies.set(SESSION_COOKIE_NAME, sid, {
      httpOnly: true,
      maxAge: Math.floor(SESSION_TTL_MS / MS_PER_SECOND),
      path: "/",
      sameSite: "lax",
      secure: !dev,
    });

    redirect(HTTP_SEE_OTHER, redirectTo);
  },
};
