import type { AddressInfo } from "node:net";
import type { Server } from "node:http";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { createMockServer } from "../src/server.js";

const SESSION_COOKIE = "kden-session=mock-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";

let baseUrl: string;
let server: Server;

const startServer = async (): Promise<void> => {
  server = await createMockServer();
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address() as AddressInfo;
  baseUrl = `http://127.0.0.1:${address.port}`;
};

const stopServer = async (): Promise<void> => {
  await new Promise<void>((resolve, reject) =>
    server.close((error) => (error ? reject(error) : resolve())),
  );
};

const request = async (path: string, init?: RequestInit): Promise<Response> =>
  fetch(`${baseUrl}${path}`, { redirect: "manual", ...init });

beforeEach(() => startServer());

afterEach(() => stopServer());

describe("Konfidence mock API", () => {
  test("serves the OpenAPI operations", async () => {
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
    await expect(identity.json()).resolves.toMatchObject({
      email: "alex.admin@example.com",
      projectRoles: {
        "identity-service": ["admin"],
        "payments-platform": ["admin", "dev"],
      },
    });

    const projects = await request("/api/v1/projects", {
      headers: { cookie: SESSION_COOKIE },
    });
    expect(projects.status).toBe(200);
    const projectList = (await projects.json()) as { data: unknown[] };
    expect(projectList.data[0]).toMatchObject({ id: "payments-platform" });

    const emptyProject = await request("/api/v1/projects/identity-service/landscapes", {
      headers: { cookie: SESSION_COOKIE },
    });
    await expect(emptyProject.json()).resolves.toEqual({ data: [] });

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

  test("serves the developer persona with limited and sparse project data", async () => {
    const unauthorized = await request("/api/v1/projects");
    expect(unauthorized.status).toBe(401);

    const developerCookie = `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=developer`;
    const identity = await request("/api/v1/identity", {
      headers: { cookie: developerCookie },
    });
    await expect(identity.json()).resolves.toMatchObject({
      email: "dana.developer@example.com",
      projectRoles: { "payments-platform": ["dev"] },
    });

    const projects = await request("/api/v1/projects", {
      headers: { cookie: developerCookie },
    });
    await expect(projects.json()).resolves.toEqual({
      data: [{ id: "payments-platform", name: "Payments Platform" }],
    });

    const landscapes = await request("/api/v1/projects/payments-platform/landscapes", {
      headers: { cookie: developerCookie },
    });
    expect(landscapes.status).toBe(200);
    await expect(landscapes.json()).resolves.toEqual({
      data: [
        { id: "development", name: "Development" },
        { id: "test", name: "Test" },
      ],
    });

    const inaccessible = await request("/api/v1/projects/identity-service/landscapes", {
      headers: { cookie: developerCookie },
    });
    expect(inaccessible.status).toBe(403);

    const artifacts = await request("/api/v1/projects/payments-platform/artifactDeployments", {
      headers: { cookie: developerCookie },
    });
    await expect(artifacts.json()).resolves.toEqual({ data: [] });
  });

  test("serves an authenticated persona during degraded resource operations", async () => {
    const degradedCookie = `${SESSION_COOKIE}; ${SCENARIO_COOKIE}=degraded`;
    const identity = await request("/api/v1/identity", {
      headers: { cookie: degradedCookie },
    });
    await expect(identity.json()).resolves.toMatchObject({
      email: "riley.operator@example.com",
    });

    const projects = await request("/api/v1/projects", {
      headers: { cookie: degradedCookie },
    });
    expect(projects.status).toBe(200);

    const landscapes = await request("/api/v1/projects/payments-platform/landscapes", {
      headers: { cookie: degradedCookie },
    });
    expect(landscapes.status).toBe(500);
  });

  test("rejects invalid requests", async () => {
    const invalidLogin = await request("/api/v1/login");
    expect(invalidLogin.status).toBe(400);

    const state = Buffer.from("http://127.0.0.1:4173/login").toString("base64url");
    const deniedLogin = await request(
      `/api/v1/auth/callback?state=${state}&error=access_denied&error_description=Login%20denied`,
    );
    expect(deniedLogin.status).toBe(401);
    await expect(deniedLogin.json()).resolves.toMatchObject({
      error: { message: "Login denied" },
    });

    const malformedExchange = await request("/api/v1/exchange", {
      body: "{",
      headers: { "content-type": "application/json" },
      method: "POST",
    });
    expect(malformedExchange.status).toBe(400);

    const notFound = await request("/api/not-found");
    expect(notFound.status).toBe(404);

    const methodNotAllowed = await request("/api/v1/projects", {
      headers: { cookie: SESSION_COOKIE },
      method: "POST",
    });
    expect(methodNotAllowed.status).toBe(405);
  });
});
