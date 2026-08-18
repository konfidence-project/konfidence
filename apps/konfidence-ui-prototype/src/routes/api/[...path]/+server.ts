import { KONFIDENCE_API_URL } from "$app/env/private";
import type { RequestHandler } from "./$types";

const proxy: RequestHandler = ({ fetch, request, url }) => {
  const target = new URL(url.pathname + url.search, KONFIDENCE_API_URL);
  const headers = new Headers(request.headers);
  headers.delete("host");

  return fetch(target, {
    body: request.body,
    headers,
    method: request.method,
    redirect: "manual",
  });
};

export const GET = proxy;
export const fallback = proxy;
