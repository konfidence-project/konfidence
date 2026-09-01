import type { FastifyInstance } from "fastify";
import type { AddressInfo } from "node:net";
import { afterAll, beforeAll, expect, test } from "vitest";
import { createMockServer } from "../src/server.js";

const SESSION = "kden-session=mock-session";
const scenario = (name: string): string => `${SESSION}; konfidence_mock_scenario=${name}`;

let baseUrl: string;
let server: FastifyInstance;

beforeAll(async () => {
  server = await createMockServer();
  await server.listen({ host: "127.0.0.1", port: 0 });
  baseUrl = `http://127.0.0.1:${(server.server.address() as AddressInfo).port}`;
});

afterAll(() => server.close());

const get = (path: string, cookie = SESSION): Promise<Response> =>
  fetch(`${baseUrl}${path}`, { headers: { cookie }, redirect: "manual" });

const post = (path: string, init: RequestInit = {}): Promise<Response> =>
  fetch(`${baseUrl}${path}`, { method: "POST", redirect: "manual", ...init });

test("logs in through the callback and sets a session cookie", async () => {
  const returnUrl = "http://127.0.0.1:4173/projects";
  const login = await get(`/api/v1/login?return_url=${encodeURIComponent(returnUrl)}`);
  expect(login.status).toBe(302);

  const callback = await get(login.headers.get("location")!);
  expect(callback.status).toBe(302);
  expect(callback.headers.get("location")).toBe(returnUrl);
  expect(callback.headers.get("set-cookie")).toContain(SESSION);
});

test("requires a session cookie", async () => {
  const response = await fetch(`${baseUrl}/api/v1/projects`);
  expect(response.status).toBe(401);
  await expect(response.json()).resolves.toEqual({
    error: { code: "401", message: "Authentication required" },
  });
});

test("serves the identity with the roles the user holds per project", async () => {
  const identity = await get("/api/v1/identity");
  expect(identity.status).toBe(200);
  await expect(identity.json()).resolves.toMatchObject({
    email: "alex.admin@example.com",
    projectRoles: {
      "identity-service": ["admin"],
      "payments-platform": ["admin", "dev"],
    },
  });
});

test("lists the projects of the admin scenario", async () => {
  const projects = await get("/api/v1/projects");
  expect(projects.status).toBe(200);
  await expect(projects.json()).resolves.toMatchObject({
    data: [{ id: "payments-platform" }, { id: "identity-service" }],
  });
});

test("lists empty collections for a project without resources", async () => {
  const landscapes = await get("/api/v1/projects/identity-service/landscapes");
  expect(landscapes.status).toBe(200);
  await expect(landscapes.json()).resolves.toEqual({ data: [] });
});

test("filters stages by landscape", async () => {
  const stages = await get("/api/v1/projects/payments-platform/stages?landscapeId=production");
  await expect(stages.json()).resolves.toMatchObject({
    data: [{ id: "prod-eu30", landscapeId: "production" }],
  });
});

test("filters vector deployments by landscape", async () => {
  const vectors = await get(
    "/api/v1/projects/payments-platform/vectorDeployments?landscapeId=development",
  );
  await expect(vectors.json()).resolves.toMatchObject({ data: [{ id: "vector-dev-us30-1" }] });
});

test("filters artifact deployments by vector deployment", async () => {
  const artifacts = await get(
    "/api/v1/projects/payments-platform/artifactDeployments?vectorDeploymentId=vector-dev-us30-1",
  );
  await expect(artifacts.json()).resolves.toMatchObject({
    data: [{ id: "artifact-dev-us30-1" }, { id: "artifact-dev-us30-2" }],
  });
});

test("serves the developer scenario with one sparse project", async () => {
  const cookie = scenario("developer");

  const identity = await get("/api/v1/identity", cookie);
  await expect(identity.json()).resolves.toMatchObject({
    email: "dana.developer@example.com",
    projectRoles: { "payments-platform": ["dev"] },
  });

  const projects = await get("/api/v1/projects", cookie);
  await expect(projects.json()).resolves.toEqual({
    data: [{ id: "payments-platform", name: "Payments Platform" }],
  });

  const landscapes = await get("/api/v1/projects/payments-platform/landscapes", cookie);
  await expect(landscapes.json()).resolves.toEqual({
    data: [
      { id: "development", name: "Development" },
      { id: "test", name: "Test" },
    ],
  });

  const artifacts = await get("/api/v1/projects/payments-platform/artifactDeployments", cookie);
  await expect(artifacts.json()).resolves.toEqual({ data: [] });
});

test("denies access to a project the scenario does not include", async () => {
  const response = await get("/api/v1/projects/identity-service/landscapes", scenario("developer"));
  expect(response.status).toBe(403);
});

test("fails project resource requests in the degraded scenario", async () => {
  const cookie = scenario("degraded");

  const identity = await get("/api/v1/identity", cookie);
  await expect(identity.json()).resolves.toMatchObject({ email: "riley.operator@example.com" });

  const projects = await get("/api/v1/projects", cookie);
  expect(projects.status).toBe(200);

  const landscapes = await get("/api/v1/projects/payments-platform/landscapes", cookie);
  expect(landscapes.status).toBe(500);
});

test("exchanges a CLI code for a session", async () => {
  const exchange = await post("/api/v1/exchange", {
    body: JSON.stringify({ code: "mock-code", verifier: "mock-verifier" }),
    headers: { "content-type": "application/json" },
  });
  expect(exchange.status).toBe(200);
  expect(exchange.headers.get("set-cookie")).toContain(SESSION);
  await expect(exchange.text()).resolves.toBe("");
});

test("clears the session cookie on logout", async () => {
  const logout = await post("/api/v1/logout", { headers: { cookie: SESSION } });
  expect(logout.status).toBe(200);
  expect(logout.headers.get("set-cookie")).toContain("Max-Age=0");
});

test("rejects a login without a usable return URL", async () => {
  const missing = await get("/api/v1/login");
  expect(missing.status).toBe(400);

  const malformed = await get("/api/v1/login?return_url=nowhere");
  expect(malformed.status).toBe(400);
});

test("reports a login the identity provider denied", async () => {
  const denied = await get(
    "/api/v1/auth/callback?state=http%3A%2F%2F127.0.0.1%3A4173%2Flogin&error=access_denied&error_description=Login%20denied",
  );
  expect(denied.status).toBe(401);
  await expect(denied.json()).resolves.toMatchObject({ error: { message: "Login denied" } });
});

test("rejects a malformed request body", async () => {
  const exchange = await post("/api/v1/exchange", {
    body: "{",
    headers: { "content-type": "application/json" },
  });
  expect(exchange.status).toBe(400);
});

test("reports unknown routes in the error schema", async () => {
  const response = await get("/api/not-found");
  expect(response.status).toBe(404);
  await expect(response.json()).resolves.toEqual({
    error: { code: "404", message: "Not found" },
  });
});
