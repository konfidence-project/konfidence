import { toAuthUser, type AuthStatus, type AuthUser } from "$lib/auth/types";
import { createApiClient, type ApiClient } from "$lib/konfidence-api/client";
import { goto } from "$app/navigation";
import { HTTP_UNAUTHORIZED } from "$lib/http-status";

const LOGIN_PATH = "/login";
const LOGOUT_PATH = "/logout";
const LOGIN_API_PATH = "/api/v1/login";

// If SSR is ever re-enabled, move this into a class held via
// setContext/getContext to avoid leaking session state between
// concurrent server-side users.
let user = $state<AuthUser | undefined>(undefined);
let status = $state<AuthStatus>("idle");
let error = $state<string | undefined>(undefined);

let inflight: Promise<void> | undefined = undefined;
let apiClient: ApiClient | undefined = undefined;

const handleUnauthorized = (): void => {
  user = undefined;
  status = "unauthenticated";
};

const getClient = (): ApiClient => {
  apiClient ??= createApiClient({ onUnauthorized: handleUnauthorized });
  return apiClient;
};

const buildReturnUrl = (returnTo: string | undefined): string => {
  const { origin } = globalThis.location;
  const target = returnTo?.startsWith("/") === true ? returnTo : "/";
  return new globalThis.URL(target, origin).href;
};

const buildLoginUrl = (returnTo?: string): string =>
  `${LOGIN_API_PATH}?return_url=${encodeURIComponent(buildReturnUrl(returnTo))}`;

const applyIdentity = (result: Awaited<ReturnType<ApiClient["GET"]>>): void => {
  if (result.data) {
    user = toAuthUser(result.data);
    status = "authenticated";
    return;
  }
  user = undefined;
  status = "unauthenticated";
  if (result.response.status !== HTTP_UNAUTHORIZED) {
    error = `Unable to load identity (status ${result.response.status})`;
  }
};

const runRefresh = async (): Promise<void> => {
  try {
    const result = await getClient().GET("/v1/identity");
    applyIdentity(result);
  } catch (fetchError) {
    user = undefined;
    status = "unauthenticated";
    error = fetchError instanceof Error ? fetchError.message : "Failed to reach the API";
  } finally {
    inflight = undefined;
  }
};

const refresh = async (): Promise<void> => {
  if (inflight) {
    return inflight;
  }
  status = "loading";
  error = undefined;
  inflight = runRefresh();
  return inflight;
};

const signIn = (returnTo?: string): void => {
  globalThis.location.assign(buildLoginUrl(returnTo));
};

const signOut = async (): Promise<void> => {
  try {
    await getClient().POST("/v1/logout");
  } catch {
    // Ignore logout failures; local state is cleared regardless.
  }
  user = undefined;
  status = "unauthenticated";
  error = undefined;
  await goto(LOGIN_PATH);
};

const clearError = (): void => {
  error = undefined;
};

const session = {
  buildLoginUrl,
  clearError,
  get error(): string | undefined {
    return error;
  },
  refresh,
  signIn,
  signOut,
  get status(): AuthStatus {
    return status;
  },
  get user(): AuthUser | undefined {
    return user;
  },
};

const resetSessionForTest = (): void => {
  user = undefined;
  status = "idle";
  error = undefined;
  inflight = undefined;
  apiClient = undefined;
};

export { LOGIN_PATH, LOGOUT_PATH, resetSessionForTest, session };
