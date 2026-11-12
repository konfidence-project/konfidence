import type {
  KonfidenceApiClient,
  KonfidenceApiClientOptions,
  Middleware,
} from "@konfidence/api-client";
import { createKonfidenceApiClient } from "@konfidence/api-client";
import { HTTP_UNAUTHORIZED } from "#lib/http-status.ts";

type ApiClient = KonfidenceApiClient;

interface CreateApiClientOptions {
  baseUrl?: string;
  fetch?: KonfidenceApiClientOptions["fetch"];
  onUnauthorized?: () => void;
}

const DEFAULT_API_BASE_URL = "/api";

/**
 * Resolves the API base URL from `VITE_KONFIDENCE_API_BASE_URL` at build time,
 * falling back to the same-origin `/api` path. When the value is a fully
 * qualified URL (http/https), the client treats the API as cross-origin and
 * sends credentialed requests; the backend must then respond with matching
 * CORS headers (`Access-Control-Allow-Origin` + `Access-Control-Allow-Credentials`).
 */
const resolveApiBaseUrl = (): string =>
  import.meta.env.VITE_KONFIDENCE_API_BASE_URL ?? DEFAULT_API_BASE_URL;

const API_PATH_SUFFIX = /\/api\/?$/;

/**
 * Resolves the Swagger UI URL next to the configured API base URL.
 *
 * Examples:
 * - `/api` becomes `/docs/`
 * - `https://api.example.test/api` becomes
 *   `https://api.example.test/docs/`
 */
const resolveApiExplorerUrl = (apiBaseUrl = resolveApiBaseUrl()): string => {
  const explorerUrl = apiBaseUrl.replace(API_PATH_SUFFIX, "/docs/");

  // Preserve useful behavior for an invalid or unconventional API base instead
  // of rendering a link to an API endpoint as if it were Swagger UI.
  return explorerUrl === apiBaseUrl ? "/docs/" : explorerUrl;
};

const isCrossOrigin = (baseUrl: string): boolean => /^https?:\/\//i.test(baseUrl);

const credentialsFor = (baseUrl: string): RequestCredentials => {
  if (isCrossOrigin(baseUrl)) {
    return "include";
  }
  return "same-origin";
};

const createApiClient = ({
  baseUrl = resolveApiBaseUrl(),
  fetch,
  onUnauthorized,
}: CreateApiClientOptions = {}): ApiClient => {
  const client = createKonfidenceApiClient({
    baseUrl,
    credentials: credentialsFor(baseUrl),
    fetch,
  });

  if (onUnauthorized) {
    const unauthorizedMiddleware: Middleware = {
      onResponse: ({ response }) => {
        if (response.status === HTTP_UNAUTHORIZED) {
          onUnauthorized();
        }
        return response;
      },
    };
    client.use(unauthorizedMiddleware);
  }

  return client;
};

export { createApiClient, resolveApiBaseUrl, resolveApiExplorerUrl };
export type { ApiClient };
