import type { ApiClient } from "$lib/konfidence-api/client";
import { createApiClient } from "$lib/konfidence-api/client";

/*
 * Singleton API client shared across the app. Safe as a module-level singleton
 * because the app is client-only (see `apps/konfidence-ui/src/routes/+layout.ts`
 * `export const ssr = false`). If SSR is ever re-enabled, provide this via
 * Svelte context instead so each request gets its own instance.
 */
let cachedClient: ApiClient | undefined = undefined;
let unauthorizedHandler: (() => void) | undefined = undefined;

/**
 * Register a callback invoked whenever the API returns 401. Typically wired
 * from the SessionStore in the root layout so a lost session propagates to
 * the auth state.
 */
const setOnUnauthorized = (handler: (() => void) | undefined): void => {
  unauthorizedHandler = handler;
};

const getApiClient = (): ApiClient => {
  cachedClient ??= createApiClient({
    onUnauthorized: () => unauthorizedHandler?.(),
  });
  return cachedClient;
};

/** Test-only: drop the cached client and unauthorized handler. */
const resetApiClientForTest = (): void => {
  cachedClient = undefined;
  unauthorizedHandler = undefined;
};

export { getApiClient, resetApiClientForTest, setOnUnauthorized };
