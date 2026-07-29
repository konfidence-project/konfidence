import { KONFIDENCE_API_URL } from "$app/env/private";
import type { RequestEvent } from "@sveltejs/kit";
import createOpenApiClient from "openapi-fetch";
import type { paths } from "$lib/konfidence-api/schema";
import { credentialHeaders } from "$lib/server/auth/credentials";

type ClientEvent = Pick<RequestEvent, "fetch" | "request">;

const createRequestClient = ({ fetch, request }: ClientEvent) => {
  const credentials = credentialHeaders(request);

  const authenticatedFetch = (apiRequest: Request): Promise<Response> => {
    const headers = new globalThis.Headers(apiRequest.headers);
    credentials.forEach((value, name) => headers.set(name, value));

    return fetch(new globalThis.Request(apiRequest, { headers }));
  };

  return createOpenApiClient<paths>({
    baseUrl: KONFIDENCE_API_URL,
    fetch: authenticatedFetch,
  });
};

export { createRequestClient };
