import type { AddressInfo } from "node:net";
import type { Server } from "node:http";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createMockApi, createMockServer, type MockServerOptions } from "../src/server.js";

const SESSION_COOKIE = "kden-session=mock-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";

let baseUrl: string;
let server: Server;

const startServer = async (options: MockServerOptions = {}): Promise<void> => {
  server = await createMockServer(options);
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address() as AddressInfo;
  baseUrl = `http://127.0.0.1:${address.port}`;
};

const stopServer = async (): Promise<void> => {
  await new Promise<void>((resolve, reject) =>
    server.close((error) => (error ? reject(error) : resolve())),
  );
};

const request = async (path: string, init?: RequestInit): Promise<Response> => {
  const response = await fetch(`${baseUrl}${path}`, { redirect: "manual", ...init });
  expect(response.headers.get("x-konfidence-mock-validation-error")).toBeNull();
  return response;
};

beforeEach(() => startServer());

afterEach(() => stopServer());

describe("Konfidence mock API", () => {
  test("registers and serves every OpenAPI operation", async () => {
    const { api, operationHandlers } = await createMockApi();
    const documentedOperations = api.router
      .getOperations()
      .map((operation) => operation.operationId)
      .toSorted();
    expect(Object.keys(operationHandlers).toSorted()).toEqual(documentedOperations);

    const returnUrl = "http://127.0.0.1:4173/projects";
    const login = await request(`/api/v1/login?return_url=${encodeURIComponent(returnUrl)}`);
    expect(login.status).toBe(302);
    const callbackLocation = login.headers.get("location");
    expect(callbackLocation).toBeTruthy();

    const callback = await request(callbackLocation!);
    expect(callback.status).toBe(302);
    expect(callback.headers.get("location")).toBe(returnUrl);
    expect(callback.headers.get("set-cookie")).toContain(SESSION_COOKIE);

    const identity = await request("/api/v1/identity", {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(identity.status).toBe(200);
    await expect(identity.json()).resolves.toMatchObject({ email: "alex@example.com" });

    const projects = await request("/api/v1/projects", {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(projects.status).toBe(200);
    const projectList = (await projects.json()) as { data: unknown[] };
    expect(projectList.data[0]).toMatchObject({ id: "payments-platform" });

    const landscapes = await request("/api/v1/projects/payments-platform/landscapes", {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(landscapes.status).toBe(200);

    const stages = await request(
      "/api/v1/projects/payments-platform/stages?landscapeId=production",
      { headers: { cookie: SESSION_COOKIE } },
    );
    expect(stages.status).toBe(200);
    await expect(stages.json()).resolves.toMatchObject({
      data: [{ id: "prod-eu30", landscapeId: "production" }],
    });

    const vectors = await request(
      "/api/v1/projects/payments-platform/vectorDeployments?landscapeId=development",
      { headers: { cookie: SESSION_COOKIE } },
    );
    expect(vectors.status).toBe(200);
    await expect(vectors.json()).resolves.toMatchObject({
      data: [{ id: "vector-dev-us30-1" }],
    });

    const artifacts = await request(
      "/api/v1/projects/payments-platform/artifactDeployments?vectorDeploymentId=vector-dev-us30-1",
      { headers: { cookie: SESSION_COOKIE } },
    );
    expect(artifacts.status).toBe(200);
    await expect(artifacts.json()).resolves.toMatchObject({
      data: [{ id: "artifact-dev-us30-1" }, { id: "artifact-dev-us30-2" }],
    });

    const exchange = await request("/api/v1/exchange", {
      body: JSON.stringify({ code: "mock-code", verifier: "mock-verifier" }),
      headers: { "content-type": "application/json" },
      method: "POST",
    });
    expect(exchange.status).toBe(200);
    expect(exchange.headers.get("set-cookie")).toContain(SESSION_COOKIE);
    await expect(exchange.text()).resolves.toBe("");

    const logout = await request("/api/v1/logout", {
      headers: { cookie: SESSION_COOKIE },
      method: "POST",
    });
    expect(logout.status).toBe(200);
    expect(logout.headers.get("set-cookie")).toContain("Max-Age=0");
  });

  test("supports authentication, authorization, empty, and failure scenarios", async () => {
    const unauthorized = await request("/api/v1/projects");
    expect(unauthorized.status).toBe(401);

    const forbidden = await request("/api/v1/projects/payments-platform/landscapes", {
      headers: {
        cookie: `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=forbidden`,
      },
    });
    expect(forbidden.status).toBe(403);

    const unknownProject = await request("/api/v1/projects/unknown/landscapes", {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(unknownProject.status).toBe(403);

    const noProjects = await request("/api/v1/projects", {
      headers: {
        cookie: `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=no-projects`,
      },
    });
    await expect(noProjects.json()).resolves.toEqual({ data: [] });

    const oneProject = await request("/api/v1/projects", {
      headers: {
        cookie: `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=one-project`,
      },
    });
    await expect(oneProject.json()).resolves.toMatchObject({
      data: [{ id: "payments-platform" }],
    });

    const failure = await request("/api/v1/projects", {
      headers: {
        cookie: `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=internal-error`,
      },
    });
    expect(failure.status).toBe(500);

    const forcedUnauthenticated = await request("/api/v1/projects", {
      headers: {
        cookie: `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=unauthenticated`,
      },
    });
    expect(forcedUnauthenticated.status).toBe(401);
  });

  test("rejects invalid requests and detects invalid response bodies", async () => {
    const invalidLogin = await request("/api/v1/login");
    expect(invalidLogin.status).toBe(400);

    const invalidExchange = await request("/api/v1/exchange", {
      body: JSON.stringify({ code: "wrong", verifier: "wrong" }),
      headers: { "content-type": "application/json" },
      method: "POST",
    });
    expect(invalidExchange.status).toBe(401);

    const malformedExchange = await request("/api/v1/exchange", {
      body: "{",
      headers: { "content-type": "application/json" },
      method: "POST",
    });
    expect(malformedExchange.status).toBe(500);

    const notFound = await request("/api/not-found");
    expect(notFound.status).toBe(404);

    const methodNotAllowed = await request("/api/v1/projects", {
      headers: { cookie: SESSION_COOKIE },
      method: "POST",
    });
    expect(methodNotAllowed.status).toBe(405);

    const { api } = await createMockApi();
    const validation = api.validateResponse({ unexpected: true }, "listProjectsV1", 200);
    expect(validation.errors).not.toBeNull();

    await stopServer();
    await startServer({
      operationHandlers: {
        listProjectsV1: () => ({ body: { unexpected: true }, status: 200 }),
      },
    });
    const consoleError = vi.spyOn(globalThis.console, "error").mockImplementation(() => undefined);
    const invalidResponse = await fetch(`${baseUrl}/api/v1/projects`, {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(invalidResponse.status).toBe(500);
    expect(invalidResponse.headers.get("x-konfidence-mock-validation-error")).toBe("true");
    expect(consoleError).toHaveBeenCalledWith(
      "Mock response violates the OpenAPI contract",
      expect.any(Object),
    );
    consoleError.mockRestore();
  });
});
