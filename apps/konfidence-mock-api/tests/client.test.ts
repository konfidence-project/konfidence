import type { FastifyInstance } from "fastify";
import { afterAll, beforeAll, describe, expect, test } from "vitest";
import { createKonfidenceApiClient, type KonfidenceApiClient } from "@konfidence/api-client";
import { createMockServer } from "../src/server.js";

const SESSION_COOKIE = "kden-session=mock-session";

describe("typed API client", () => {
  let client: KonfidenceApiClient;
  let server: FastifyInstance;

  beforeAll(async () => {
    server = await createMockServer();
    const address = await server.listen({ host: "127.0.0.1", port: 0 });
    client = createKonfidenceApiClient({
      baseUrl: `${address}/api`,
      headers: { cookie: SESSION_COOKIE },
    });
  });

  afterAll(async () => server.close());

  test("reads projects through the generated contract", async () => {
    const { data, error, response } = await client.request("get", "/v1/projects");

    expect(response.status).toBe(200);
    expect(error).toBeUndefined();
    expect(data?.data).toContainEqual({ id: "payments-platform", name: "Payments Platform" });
  });

  test("returns a typed error response", async () => {
    const anonymousClient = createKonfidenceApiClient({ baseUrl: `${server.listeningOrigin}/api` });
    const { data, error, response } = await anonymousClient.request("get", "/v1/projects");

    expect(response.status).toBe(401);
    expect(data).toBeUndefined();
    expect(error).toEqual({
      error: { code: "401", message: "Authentication required" },
    });
  });
});
