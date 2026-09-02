import { expect, test, vi } from "vitest";
import { createApiClient } from "$lib/konfidence-api/client";

const requestCredentials = async (baseUrl: string): Promise<RequestCredentials> => {
  const fetch = vi.fn(async (_request: Request) =>
    globalThis.Response.json({ data: [] }, { status: 200 }),
  );
  const client = createApiClient({ baseUrl, fetch });

  await client.request("get", "/v1/projects");

  return fetch.mock.calls[0]?.[0].credentials;
};

test("uses same-origin credentials with the dashboard proxy", async () => {
  await expect(requestCredentials("/api")).resolves.toBe("same-origin");
});

test("includes credentials with an absolute API URL", async () => {
  await expect(requestCredentials("https://api.example.test/api")).resolves.toBe("include");
});
