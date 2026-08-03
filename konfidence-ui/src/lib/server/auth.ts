import type { AuthRole, AuthSession, AuthUser } from "$lib/auth/types";
import { HTTP_BAD_GATEWAY, HTTP_FORBIDDEN, HTTP_UNAUTHORIZED } from "$lib/http-status";
import type { components } from "$lib/konfidence-api/schema";
import { type RequestEvent, error } from "@sveltejs/kit";
import { createRequestClient } from "$lib/server/konfidence-api/client";
import { hasCredentials } from "$lib/server/auth/credentials";

const IDENTITY_TIMEOUT_MS = 5000;
const AUTH_ROLES = ["ADMIN", "DEV", "PM"] as const;

type ApiIdentity = components["schemas"]["IdentityResponse"];
interface IdentityResult {
  data?: ApiIdentity;
  response: Response;
}

const toUser = (identity: ApiIdentity): AuthUser => ({
  ...identity,
  roles: identity.roles.filter((role): role is AuthRole => AUTH_ROLES.includes(role as AuthRole)),
});

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

const loadIdentity = async (event: RequestEvent, signal: AbortSignal): Promise<IdentityResult> => {
  const api = createRequestClient(event);
  const result = await api.GET("/api/identity", { signal });
  return { data: result.data, response: result.response };
};

const identityFromResult = (result: IdentityResult): ApiIdentity => {
  const { data, response } = result;
  if (!response.ok) {
    error(HTTP_BAD_GATEWAY, {
      code: "IDENTITY_LOAD_FAILED",
      message: "Failed to load the signed-in identity",
    });
  }

  if (!data) {
    error(HTTP_BAD_GATEWAY, {
      code: "IDENTITY_EMPTY",
      message: "The signed-in identity response was empty",
    });
  }

  return data;
};

const sessionFromResult = (result: IdentityResult): AuthSession | undefined => {
  if (result.response.status === HTTP_UNAUTHORIZED) {
    return undefined;
  }
  return { user: toUser(identityFromResult(result)) };
};

const resolveSession = async (event: RequestEvent): Promise<AuthSession | undefined> => {
  if (!hasCredentials(event.request)) {
    return undefined;
  }

  const { controller, timeout } = timeoutSignal();

  try {
    const result = await loadIdentity(event, controller.signal);
    return sessionFromResult(result);
  } catch (error_) {
    if (error_ instanceof Error && error_.name === "AbortError") {
      error(HTTP_BAD_GATEWAY, {
        code: "IDENTITY_TIMEOUT",
        message: "Timed out while loading the signed-in identity",
      });
    }
    throw error_;
  } finally {
    clearTimeout(timeout);
  }
};

const requireUser = (locals: App.Locals): AuthUser => {
  if (!locals.user) {
    error(HTTP_UNAUTHORIZED, { code: "UNAUTHORIZED", message: "Unauthorized" });
  }
  return locals.user;
};

const requireRole = (locals: App.Locals, roles: readonly AuthRole[]): AuthUser => {
  const user = requireUser(locals);
  if (!user.roles.some((role) => roles.includes(role))) {
    error(HTTP_FORBIDDEN, { code: "FORBIDDEN", message: "Forbidden" });
  }
  return user;
};

export { requireRole, requireUser, resolveSession };
