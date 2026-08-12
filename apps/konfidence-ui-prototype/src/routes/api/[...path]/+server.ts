import { KONFIDENCE_API_URL } from "$app/env/private";
import type { RequestHandler } from "./$types";

const proxy: RequestHandler = async ({ fetch, request, url }) => {
  const target = new URL(url.pathname + url.search, KONFIDENCE_API_URL);
  const startedAt = Date.now();
  const headers = new Headers(request.headers);
  headers.delete("host");
  const upstreamRequest = new Request(target, {
    body:
      request.method === "GET" || request.method === "HEAD"
        ? undefined
        : await request.arrayBuffer(),
    headers,
    method: request.method,
    redirect: "manual",
  });

  // oxlint-disable-next-line no-undef -- Console output is intentional server-side proxy diagnostics.
  console.info("[api-proxy] request", {
    method: request.method,
    path: url.pathname,
    targetOrigin: target.origin,
    targetPath: target.pathname,
  });

  try {
    const response = await fetch(upstreamRequest);

    // oxlint-disable-next-line no-undef -- Console output is intentional server-side proxy diagnostics.
    console.info("[api-proxy] response", {
      durationMs: Date.now() - startedAt,
      method: request.method,
      path: url.pathname,
      status: response.status,
    });

    return response;
  } catch (error) {
    // oxlint-disable-next-line no-undef -- Console output is intentional server-side proxy diagnostics.
    console.error("[api-proxy] error", {
      durationMs: Date.now() - startedAt,
      error,
      method: request.method,
      path: url.pathname,
      targetOrigin: target.origin,
    });
    throw error;
  }
};

export const GET = proxy;
export const fallback = proxy;
