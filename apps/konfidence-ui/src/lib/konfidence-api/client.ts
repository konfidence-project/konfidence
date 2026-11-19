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

/**
 * The configured value identifies the Konfidence server and does not include
 * the `/api` path or a trailing slash. An empty value uses the current origin.
 */
const KONFIDENCE_API_BASE_URL = import.meta.env.VITE_KONFIDENCE_API_BASE_URL ?? "";
const KONFIDENCE_API_URL = `${KONFIDENCE_API_BASE_URL}/api`;
const KONFIDENCE_API_DOCS_URL = `${KONFIDENCE_API_BASE_URL}/docs/`;

const isCrossOrigin = (baseUrl: string): boolean => /^https?:\/\//i.test(baseUrl);

const credentialsFor = (baseUrl: string): RequestCredentials => {
  if (isCrossOrigin(baseUrl)) {
    return "include";
  }
  return "same-origin";
};

const createApiClient = ({
  baseUrl = KONFIDENCE_API_URL,
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

export { KONFIDENCE_API_URL, KONFIDENCE_API_BASE_URL, KONFIDENCE_API_DOCS_URL, createApiClient };
export type { ApiClient };
