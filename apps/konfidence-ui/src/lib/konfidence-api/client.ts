import createOpenApiClient, { type Client, type Middleware } from "openapi-fetch";
import { HTTP_UNAUTHORIZED } from "$lib/http-status";
import type { paths } from "$lib/konfidence-api/schema";

type ApiClient = Client<paths>;

interface CreateApiClientOptions {
  onUnauthorized?: () => void;
}

const DEFAULT_API_BASE_URL = "/api";

/**
 * Resolves the API base URL from `VITE_KONFIDENCE_API_BASE_URL` at build time,
 * falling back to the same-origin `/api` proxy. When the value is a fully
 * qualified URL (http/https), the client treats the API as cross-origin and
 * sends credentialed requests; the backend must then respond with matching
 * CORS headers (`Access-Control-Allow-Origin` + `Access-Control-Allow-Credentials`).
 */
const resolveApiBaseUrl = (): string =>
  import.meta.env.VITE_KONFIDENCE_API_BASE_URL ?? DEFAULT_API_BASE_URL;

const isCrossOrigin = (baseUrl: string): boolean => /^https?:\/\//i.test(baseUrl);

const createApiClient = ({ onUnauthorized }: CreateApiClientOptions = {}): ApiClient => {
  const baseUrl = resolveApiBaseUrl();
  const client = createOpenApiClient<paths>({
    baseUrl,
    credentials: isCrossOrigin(baseUrl) ? "include" : "same-origin",
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

export { createApiClient, resolveApiBaseUrl };
export type { ApiClient };
