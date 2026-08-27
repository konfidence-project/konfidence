import createOpenApiClient, { type Client, type Middleware } from "openapi-fetch";
import { HTTP_UNAUTHORIZED } from "$lib/http-status";
import type { paths } from "$lib/konfidence-api/schema";

type ApiClient = Client<paths>;

interface CreateApiClientOptions {
  onUnauthorized?: () => void;
}

const createApiClient = ({ onUnauthorized }: CreateApiClientOptions = {}): ApiClient => {
  const client = createOpenApiClient<paths>({
    baseUrl: "/api",
    credentials: "same-origin",
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

export { createApiClient };
export type { ApiClient };
