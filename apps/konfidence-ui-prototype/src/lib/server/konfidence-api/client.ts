import { KONFIDENCE_API_URL } from "$app/env/private";
import type { RequestEvent } from "@sveltejs/kit";
import createOpenApiClient from "openapi-fetch";
import type { paths } from "$lib/konfidence-api/schema";

type ClientEvent = Pick<RequestEvent, "fetch" | "request">;

const createRequestClient = ({ fetch, request }: ClientEvent) => {
  const requestHeaders = new globalThis.Headers(request.headers);
  requestHeaders.delete("host");

  const authenticatedFetch = (apiRequest: Request): Promise<Response> => {
    const headers = new globalThis.Headers(apiRequest.headers);
    requestHeaders.forEach((value, name) => headers.set(name, value));

    return fetch(new globalThis.Request(apiRequest, { headers }));
  };

  return createOpenApiClient<paths>({
    baseUrl: `${KONFIDENCE_API_URL}/api`,
    fetch: authenticatedFetch,
  });
};

export { createRequestClient };
