import { afterEach, expect, test, vi } from "vitest";
import { createKonfidenceApiClient } from "./client.js";
import { withConflictRetries } from "./retry.js";

const { AbortController, Request, Response } = globalThis;

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

test.each([
  { name: "success", statuses: [204] },
  { name: "invalid request", statuses: [400] },
  { name: "unauthorized", statuses: [401] },
  { name: "forbidden", statuses: [403] },
  { name: "not found", statuses: [404] },
  { name: "permanent error", statuses: [422] },
  { name: "server error", statuses: [500] },
  { name: "conflict then success", statuses: [409, 204] },
  { name: "conflict then permanent error", statuses: [409, 422] },
  { name: "exhausted retries", statuses: [409, 409, 409] },
])("returns the final response for $name", async ({ statuses }) => {
  const responses = statuses.map(
    (status) => new Response(status === 204 ? null : "server message", { status }),
  );
  let attempts = 0;
  const fetch = vi.fn(async () => responses[Math.min(attempts++, responses.length - 1)]);

  const response = await withConflictRetries(fetch)(new Request("https://api.example.test"));

  expect(fetch).toHaveBeenCalledTimes(statuses.length);
  expect(response).toBe(responses.at(-1));
  expect(await response.text()).toBe(response.status === 204 ? "" : "server message");
});

test("waits 100 ms and cancels only discarded response bodies", async () => {
  vi.useFakeTimers();
  const conflict = new Response("conflict", { status: 409 });
  const success = new Response("success");
  const cancelConflict = vi.spyOn(conflict.body!, "cancel");
  const cancelSuccess = vi.spyOn(success.body!, "cancel");
  const fetch = vi.fn().mockResolvedValueOnce(conflict).mockResolvedValueOnce(success);

  const pending = withConflictRetries(fetch)(new Request("https://api.example.test"));
  await vi.advanceTimersByTimeAsync(99);
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(cancelConflict).toHaveBeenCalledOnce();
  await vi.advanceTimersByTimeAsync(1);

  expect(await pending).toBe(success);
  expect(fetch).toHaveBeenCalledTimes(2);
  expect(cancelSuccess).not.toHaveBeenCalled();
  expect(vi.getTimerCount()).toBe(0);
});

test("does not retry transport failures", async () => {
  const failure = new TypeError("network failure");
  const fetch = vi.fn().mockRejectedValue(failure);

  await expect(withConflictRetries(fetch)(new Request("https://api.example.test"))).rejects.toBe(
    failure,
  );
  expect(fetch).toHaveBeenCalledOnce();
});

test("lets fetch reject an already-aborted request", async () => {
  const controller = new AbortController();
  const reason = new Error("navigation cancelled");
  controller.abort(reason);
  const fetch = vi.fn(globalThis.fetch);

  await expect(
    withConflictRetries(fetch)(
      new Request("https://api.example.test", {
        signal: controller.signal,
      }),
    ),
  ).rejects.toBe(reason);
  expect(fetch).toHaveBeenCalledOnce();
});

test("lets fetch reject an abort after the backoff finishes", async () => {
  vi.useFakeTimers();
  const controller = new AbortController();
  const reason = new Error("navigation cancelled");
  const fetch = vi
    .fn(globalThis.fetch)
    .mockResolvedValueOnce(new Response("conflict", { status: 409 }));
  const pending = withConflictRetries(fetch)(
    new Request("https://api.example.test", {
      signal: controller.signal,
    }),
  );
  const rejected = expect(pending).rejects.toBe(reason);

  await vi.advanceTimersByTimeAsync(50);
  expect(vi.getTimerCount()).toBe(1);
  controller.abort(reason);
  expect(fetch).toHaveBeenCalledOnce();

  await vi.advanceTimersByTimeAsync(50);
  await rejected;
  expect(fetch).toHaveBeenCalledTimes(2);
  expect(vi.getTimerCount()).toBe(0);
});

test("forwards cancellation to the active fetch", async () => {
  const controller = new AbortController();
  const reason = new Error("navigation cancelled");
  const fetch = vi.fn(
    (request: Request) =>
      new Promise<Response>((_resolve, reject) => {
        request.signal.addEventListener("abort", () => reject(request.signal.reason), {
          once: true,
        });
      }),
  );
  const pending = withConflictRetries(fetch)(
    new Request("https://api.example.test", {
      signal: controller.signal,
    }),
  );

  controller.abort(reason);

  await expect(pending).rejects.toBe(reason);
  expect(fetch).toHaveBeenCalledOnce();
});

test("the shared client replays JSON bodies and preserves request options", async () => {
  const bodies: unknown[] = [];
  const fetch = vi.fn(async (request: Request) => {
    bodies.push(await request.json());
    return bodies.length < 3
      ? Response.json({ error: { message: "concurrent change" } }, { status: 409 })
      : new Response(null, { status: 204 });
  });
  const client = createKonfidenceApiClient({
    baseUrl: "https://api.example.test",
    fetch,
    credentials: "include",
    headers: { Authorization: "Bearer test-token" },
  });

  const { response } = await client.POST(
    "/v1/projects/{projectId}/vectorPromotions/{vectorPromotionId}/reject",
    {
      params: { path: { projectId: "project", vectorPromotionId: "promotion" } },
      body: { reason: "known bad vector" },
    },
  );

  expect(response.status).toBe(204);
  expect(bodies).toEqual(Array.from({ length: 3 }, () => ({ reason: "known bad vector" })));
  for (const [request] of fetch.mock.calls) {
    expect(request.url).toBe(
      "https://api.example.test/v1/projects/project/vectorPromotions/promotion/reject",
    );
    expect(request.method).toBe("POST");
    expect(request.credentials).toBe("include");
    expect(request.headers.get("Authorization")).toBe("Bearer test-token");
  }
});

test("the shared client also wraps the default fetch", async () => {
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(new Response(null, { status: 409 }))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  vi.stubGlobal("fetch", fetch);
  const client = createKonfidenceApiClient({ baseUrl: "https://api.example.test" });

  const { response } = await client.POST(
    "/v1/projects/{projectId}/vectorPromotions/{vectorPromotionId}/approve",
    {
      params: { path: { projectId: "project", vectorPromotionId: "promotion" } },
    },
  );

  expect(response.status).toBe(204);
  expect(fetch).toHaveBeenCalledTimes(2);
});
