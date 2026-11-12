import { expect, test, vi } from "vitest";
import { createApiClient } from "#lib/konfidence-api/client.ts";

const requestCredentials = async (baseUrl: string): Promise<RequestCredentials> => {
  const fetch = vi.fn(async (_request: Request) =>
    globalThis.Response.json({ data: [] }, { status: 200 }),
  );
  const client = createApiClient({ baseUrl, fetch });

  await client.request("get", "/v1/projects");

  return fetch.mock.calls[0]?.[0].credentials;
};

test("uses same-origin credentials with the default API path", async () => {
  await expect(requestCredentials("/api")).resolves.toBe("same-origin");
});

test("includes credentials with an absolute API URL", async () => {
  await expect(requestCredentials("https://api.example.test/api")).resolves.toBe("include");
});

test("retries a promotion conflict through the shared client", async () => {
  const fetch = vi
    .fn<(request: Request) => Promise<Response>>()
    .mockResolvedValueOnce(
      globalThis.Response.json({ error: { message: "concurrent change" } }, { status: 409 }),
    )
    .mockResolvedValueOnce(new globalThis.Response(null, { status: 204 }));
  const client = createApiClient({ fetch });

  const { response } = await client.POST(
    "/v1/projects/{projectId}/vectorPromotions/{vectorPromotionId}/approve",
    {
      params: { path: { projectId: "project", vectorPromotionId: "promotion" } },
    },
  );

  expect(response.status).toBe(204);
  expect(fetch).toHaveBeenCalledTimes(2);
});
