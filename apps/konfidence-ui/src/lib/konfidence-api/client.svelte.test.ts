import { expect, test, vi } from "vitest";
import { createApiClient, resolveApiExplorerUrl } from "#lib/konfidence-api/client.ts";

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

test.each([
  ["/api", "/docs/"],
  ["/api/", "/docs/"],
  ["https://api.example.test/api", "https://api.example.test/docs/"],
  ["https://api.example.test/api/", "https://api.example.test/docs/"],
])("resolves the API explorer URL from %s", (apiBaseUrl, expected) => {
  expect(resolveApiExplorerUrl(apiBaseUrl)).toBe(expected);
});

test("falls back to same-origin docs for an unconventional API base URL", () => {
  expect(resolveApiExplorerUrl("https://api.example.test/custom")).toBe("/docs/");
});
