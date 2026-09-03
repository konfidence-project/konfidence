import { getContext, setContext } from "svelte";
import type { AuthStatus, AuthUser } from "$lib/auth/types";
import { toAuthUser } from "$lib/auth/types";
import type { ApiClient } from "$lib/konfidence-api/client";
import { resolveApiBaseUrl } from "$lib/konfidence-api/client";
import { goto } from "$app/navigation";
import { HTTP_UNAUTHORIZED } from "$lib/http-status";
import type { paths } from "$lib/konfidence-api/schema";

// SvelteKit UI routes (not part of the OpenAPI surface).
const LOGIN_PATH = "/login";
const LOGOUT_PATH = "/logout";

// OpenAPI-typed operation paths. `satisfies keyof paths` guarantees these break
// at build time if the schema is regenerated with a different route.
const LOGIN_API_ROUTE = "/v1/login" satisfies keyof paths;
const LOGOUT_API_ROUTE = "/v1/logout" satisfies keyof paths;
const IDENTITY_API_ROUTE = "/v1/identity" satisfies keyof paths;

type IdentityResult = Awaited<ReturnType<ApiClient["GET"]>>;

/**
 * Reactive store for the authenticated session. State is exposed as `$state`
 * fields so consumers can read them reactively inside Svelte components.
 *
 * Provide an instance via `provideSession()` in the root layout and read it
 * with `useSession()` further down the tree. Using context (rather than a
 * module-level singleton) keeps the state scoped to the component tree so
 * SSR — should we ever re-enable it — cannot leak state between concurrent
 * server-side users.
 */
class SessionStore {
  user = $state<AuthUser | undefined>(undefined);
  status = $state<AuthStatus>("idle");
  error = $state<string | undefined>(undefined);

  readonly #client: ApiClient;
  #inflight: Promise<void> | undefined = undefined;

  constructor(client: ApiClient) {
    this.#client = client;
  }

  /** Called by the API client middleware when it sees a 401 response. */
  handleUnauthorized(): void {
    this.user = undefined;
    this.status = "unauthenticated";
  }

  clearError(): void {
    this.error = undefined;
  }

  buildLoginUrl(returnTo?: string): string {
    const returnUrl = this.#buildReturnUrl(returnTo);
    return `${resolveApiBaseUrl()}${LOGIN_API_ROUTE}?return_url=${encodeURIComponent(returnUrl)}`;
  }

  signIn(returnTo?: string): void {
    globalThis.location.assign(this.buildLoginUrl(returnTo));
  }

  async refresh(): Promise<void> {
    if (this.#inflight) {
      return this.#inflight;
    }
    this.status = "loading";
    this.error = undefined;
    this.#inflight = this.#runRefresh();
    return this.#inflight;
  }

  async signOut(): Promise<void> {
    try {
      await this.#client.POST(LOGOUT_API_ROUTE);
    } catch {
      // Ignore logout failures; local state is cleared regardless.
    }
    this.user = undefined;
    this.status = "unauthenticated";
    this.error = undefined;
    await goto(LOGIN_PATH);
  }

  #buildReturnUrl(returnTo: string | undefined): string {
    const { origin } = globalThis.location;
    const target = returnTo?.startsWith("/") === true ? returnTo : "/";
    return new globalThis.URL(target, origin).href;
  }

  async #runRefresh(): Promise<void> {
    try {
      const result = await this.#client.GET(IDENTITY_API_ROUTE);
      this.#applyIdentity(result);
    } catch (fetchError) {
      this.user = undefined;
      this.status = "unauthenticated";
      this.error = fetchError instanceof Error ? fetchError.message : "Failed to reach the API";
    } finally {
      this.#inflight = undefined;
    }
  }

  #applyIdentity(result: IdentityResult): void {
    if (result.data) {
      this.user = toAuthUser(result.data);
      this.status = "authenticated";
      return;
    }
    this.user = undefined;
    this.status = "unauthenticated";
    if (result.response.status !== HTTP_UNAUTHORIZED) {
      this.error = `Unable to load identity (status ${result.response.status})`;
    }
  }
}

const SESSION_CONTEXT_KEY = Symbol("konfidence-session");

/** Registers the store on the current component's context. */
const provideSession = (store: SessionStore): SessionStore => {
  setContext(SESSION_CONTEXT_KEY, store);
  return store;
};

/** Reads the session store from context. Throws if none has been provided. */
const useSession = (): SessionStore => {
  const store = getContext<SessionStore | undefined>(SESSION_CONTEXT_KEY);
  if (!store) {
    throw new Error(
      "useSession() called outside of a SessionStore context. Ensure provideSession() runs in +layout.svelte before children read the store.",
    );
  }
  return store;
};

export { LOGIN_PATH, LOGOUT_PATH, provideSession, SessionStore, useSession };
