import { env } from "$env/dynamic/private";

/**
 * Which identity provider drives the login flow.
 *
 * - `stub`: local dev fallback. The login form creates a session with a
 *   randomly generated silly display name. No external calls.
 * - `oidc`: full OpenID Connect authorization-code flow against `IDP_ISSUER`.
 *   Not yet implemented — see the TODO in `resolveOidcConfig`.
 */
type IdpType = "oidc" | "stub";

interface StubIdpConfig {
  type: "stub";
}

interface OidcIdpConfig {
  clientId: string;
  clientSecret: string;
  issuer: string;
  redirectUri: string;
  /** OAuth scopes to request (space-separated on the wire, parsed here). */
  scopes: string[];
  type: "oidc";
}

type IdpConfig = OidcIdpConfig | StubIdpConfig;

const DEFAULT_SCOPES = "openid profile email";

const stripTrailingSlash = (value: string): string => {
  if (value.endsWith("/")) {
    return value.slice(0, -1);
  }
  return value;
};

const requireEnv = (name: string, missing: string[]): string => {
  const value = env[name];
  if (!value || value.trim() === "") {
    missing.push(name);
    return "";
  }
  return value;
};

const resolveOidcConfig = (): OidcIdpConfig => {
  const missing: string[] = [];
  const issuer = requireEnv("IDP_ISSUER", missing);
  const clientId = requireEnv("IDP_CLIENT_ID", missing);
  const clientSecret = requireEnv("IDP_CLIENT_SECRET", missing);
  const redirectUri = requireEnv("IDP_REDIRECT_URI", missing);

  if (missing.length > 0) {
    throw new Error(
      `IDP_TYPE=oidc requires ${missing.join(", ")} to be set. ` +
        `See .env.example for the full list.`,
    );
  }

  const scopes = (env.IDP_SCOPES ?? DEFAULT_SCOPES).split(/\s+/).filter(Boolean);

  // TODO: implement the OIDC authorization-code flow. Two endpoints need to
  //       Be added under src/routes/(auth)/oidc/ (login + callback) that
  //       Consume this config. See notes in src/lib/server/session.ts —
  //       `SessionUser` is already OIDC-shaped, so only the login side
  //       Changes when this ships.
  return {
    clientId,
    clientSecret,
    issuer: stripTrailingSlash(issuer),
    redirectUri,
    scopes,
    type: "oidc",
  };
};

/**
 * Reads the current IdP configuration from environment variables at runtime.
 *
 * Called on every request that touches the auth flow — cheap, since it's
 * just object construction. If you'd rather cache it, wrap in a
 * `let cached | undefined` here.
 */
const getIdpConfig = (): IdpConfig => {
  const rawType = (env.IDP_TYPE ?? "stub").trim().toLowerCase();

  if (rawType === "stub") {
    return { type: "stub" };
  }

  if (rawType === "oidc") {
    return resolveOidcConfig();
  }

  throw new Error(`Unknown IDP_TYPE '${env.IDP_TYPE}'. Expected 'stub' or 'oidc'.`);
};

export default getIdpConfig;
export type { IdpConfig, IdpType, OidcIdpConfig, StubIdpConfig };
