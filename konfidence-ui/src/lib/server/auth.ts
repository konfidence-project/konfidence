import * as valibot from "valibot";
import type { AuthRole, AuthSession, AuthUser } from "$lib/auth/types";
import { HTTP_BAD_GATEWAY, HTTP_FORBIDDEN, HTTP_UNAUTHORIZED } from "$lib/http-status";
import { type RequestEvent, error } from "@sveltejs/kit";
import { KONFIDENCE_API_URL } from "$app/env/private";

const IDENTITY_TIMEOUT_MS = 5000;
const AUTH_CREDENTIAL_HEADERS = ["authorization", "cookie", "x-session-id"] as const;
const AUTH_ROLES = ["", "ADMIN", "DEV", "PM"] as const;

const apiIdentitySchema = valibot.object({
  email: valibot.optional(valibot.string()),
  emailVerified: valibot.boolean(),
  name: valibot.optional(valibot.string()),
  role: valibot.optional(valibot.picklist(AUTH_ROLES)),
  subject: valibot.string(),
  username: valibot.string(),
});

type ApiIdentity = valibot.InferOutput<typeof apiIdentitySchema>;

const credentialHeaders = (request: Request): Headers => {
  const headers = new Headers();
  for (const name of AUTH_CREDENTIAL_HEADERS) {
    const value = request.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }
  return headers;
};

const hasCredentials = (request: Request): boolean =>
  AUTH_CREDENTIAL_HEADERS.some((name) => request.headers.has(name));

const toUser = (identity: ApiIdentity): AuthUser => {
  const user: AuthUser = {
    email: identity.email || undefined,
    emailVerified: identity.emailVerified,
    id: identity.subject,
    name: identity.name || identity.username,
    username: identity.username,
  };
  if (identity.role) {
    user.role = identity.role as AuthRole;
  }
  return user;
};

const timeoutSignal = (): {
  controller: AbortController;
  timeout: ReturnType<typeof setTimeout>;
} => {
  const controller = new AbortController();
  return {
    controller,
    timeout: setTimeout(() => controller.abort(), IDENTITY_TIMEOUT_MS),
  };
};

const loadIdentity = async (event: RequestEvent, signal: AbortSignal): Promise<Response> =>
  fetch(new URL("/api/identity", KONFIDENCE_API_URL), {
    headers: credentialHeaders(event.request),
    signal,
  });

const identityFromResponse = async (response: Response): Promise<ApiIdentity> => {
  if (!response.ok) {
    error(HTTP_BAD_GATEWAY, "Failed to load the signed-in identity");
  }

  const result = valibot.safeParse(apiIdentitySchema, await response.json());
  if (!result.success) {
    error(HTTP_BAD_GATEWAY, "Received an invalid signed-in identity");
  }

  return result.output;
};

const sessionFromResponse = async (response: Response): Promise<AuthSession | undefined> => {
  if (response.status === HTTP_UNAUTHORIZED) {
    return undefined;
  }
  return { user: toUser(await identityFromResponse(response)) };
};

const resolveSession = async (event: RequestEvent): Promise<AuthSession | undefined> => {
  if (!hasCredentials(event.request)) {
    return undefined;
  }

  const { controller, timeout } = timeoutSignal();

  try {
    const response = await loadIdentity(event, controller.signal);
    return await sessionFromResponse(response);
  } catch (error_) {
    if (error_ instanceof Error && error_.name === "AbortError") {
      error(HTTP_BAD_GATEWAY, "Timed out while loading the signed-in identity");
    }
    throw error_;
  } finally {
    clearTimeout(timeout);
  }
};

const requireUser = (locals: App.Locals): AuthUser => {
  if (!locals.user) {
    error(HTTP_UNAUTHORIZED, "Unauthorized");
  }
  return locals.user;
};

const requireRole = (locals: App.Locals, roles: readonly AuthRole[]): AuthUser => {
  const user = requireUser(locals);
  if (!user.role || !roles.includes(user.role)) {
    error(HTTP_FORBIDDEN, "Forbidden");
  }
  return user;
};

export { requireRole, requireUser, resolveSession };
