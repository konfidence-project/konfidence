import type { components } from "@konfidence/api-client/schema";
import type { FastifyInstance } from "fastify";
import type { AddressInfo } from "node:net";
import { afterAll, beforeAll, expect, test } from "vitest";
import { createMockServer } from "../src/server.js";

type Stage = components["schemas"]["Stage"];
type StageVersion = components["schemas"]["StageVersion"];

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

const adminStages = async (query = ""): Promise<Stage[]> => {
  const response = await get(`/api/v1/projects/payments-platform/stages${query}`);
  expect(response.status).toBe(200);
  const body = (await response.json()) as { data: Stage[] };
  return body.data;
};

const adminStageIds = [
  "dev-us30",
  "test-eu20",
  "prod-eu30",
  "dev-eu10",
  "dev-canary",
  "dev-rollback",
  "dev-new",
  "sandbox",
  "test-us10",
  "test-hotfix",
  "test-broken",
  "test-green",
  "prod-us40",
  "prod-dr",
  "prod-legacy",
];

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

  const stages = await get("/api/v1/projects/identity-service/stages");
  expect(stages.status).toBe(200);
  await expect(stages.json()).resolves.toEqual({ data: [] });
});

test("lists every landscape of the admin project, including one without stages", async () => {
  const response = await get("/api/v1/projects/payments-platform/landscapes");
  expect(response.status).toBe(200);
  await expect(response.json()).resolves.toEqual({
    data: [
      { id: "development", name: "Development" },
      { id: "test", name: "Test" },
      { id: "production", name: "Production" },
      { id: "sandbox", name: "Sandbox" },
    ],
  });
});

test("lists every stage of the admin project across all landscapes", async () => {
  const stages = await adminStages();
  const ids = stages.map(({ id }) => id);
  expect(ids).toHaveLength(adminStageIds.length);
  expect(ids).toEqual(expect.arrayContaining(adminStageIds));
});

test("filters stages by landscape", async () => {
  const stages = await adminStages("?landscapeId=production");
  expect(stages.map(({ id }) => id)).toEqual(["prod-eu30", "prod-us40", "prod-dr", "prod-legacy"]);
  expect(stages.every(({ landscapeId }) => landscapeId === "production")).toBe(true);
});

test("returns an empty stage list for a landscape without stages", async () => {
  await expect(adminStages("?landscapeId=sandbox")).resolves.toEqual([]);
});

test("represents every StageVersion status in the admin project", async () => {
  const stages = await adminStages();
  const statuses = stages.flatMap(({ activeStageVersion, targetStageVersion }) =>
    [activeStageVersion, targetStageVersion]
      .filter((version): version is StageVersion => version !== undefined)
      .map(({ status }) => status),
  );
  expect(new Set(statuses)).toEqual(
    new Set([
      "PendingDeployment",
      "DeployingVector",
      "MigratingVector",
      "ActivatingVector",
      "Ready",
      "Failed",
    ]),
  );
});

test("represents the stage version target and active combinations", async () => {
  const stages = await adminStages();
  const byId = Object.fromEntries(stages.map((stage) => [stage.id, stage]));

  // Target equals active: the desired and deployed versions are the same.
  expect(byId["dev-us30"]?.targetStageVersion).toEqual(byId["dev-us30"]?.activeStageVersion);
  expect(byId["sandbox"]?.targetStageVersion).toEqual(byId["sandbox"]?.activeStageVersion);

  // Target differs from active while the vector is still deploying.
  const deploying = byId["dev-eu10"];
  expect(deploying?.targetStageVersion?.status).toBe("DeployingVector");
  expect(deploying?.activeStageVersion?.status).toBe("Ready");
  expect(deploying?.targetStageVersion?.id).not.toBe(deploying?.activeStageVersion?.id);

  // A Ready target that still differs from the active version.
  const converged = byId["test-green"];
  expect(converged?.targetStageVersion?.status).toBe("Ready");
  expect(converged?.activeStageVersion?.status).toBe("Ready");
  expect(converged?.targetStageVersion?.id).not.toBe(converged?.activeStageVersion?.id);

  // Target without an active version: the first rollout of a new stage.
  expect(byId["dev-canary"]?.targetStageVersion?.status).toBe("PendingDeployment");
  expect(byId["dev-canary"]?.activeStageVersion).toBeUndefined();

  // Neither target nor active: the stage has no versions yet.
  expect(byId["dev-new"]?.targetStageVersion).toBeUndefined();
  expect(byId["dev-new"]?.activeStageVersion).toBeUndefined();
});

test("names stage groups with case-insensitive dev, test, and prod prefixes", async () => {
  const stages = await adminStages();
  const byId = Object.fromEntries(stages.map((stage) => [stage.id, stage]));
  expect(byId["dev-canary"]?.name).toBe("DEV-canary");
  expect(byId["test-hotfix"]?.name).toBe("Test-hotfix");
  expect(byId["prod-dr"]?.name).toBe("PROD-dr");
  expect(byId["sandbox"]?.name).toBe("sandbox");
});

test("reports a stage filter naming a landscape the project does not have as not found", async () => {
  const unknown = await get("/api/v1/projects/payments-platform/stages?landscapeId=nowhere");
  expect(unknown.status).toBe(404);

  const empty = await get("/api/v1/projects/payments-platform/stages?landscapeId=");
  expect(empty.status).toBe(404);
});

test("lists no stages for a landscape of the project that holds none", async () => {
  const stages = await get(
    "/api/v1/projects/payments-platform/stages?landscapeId=test",
    scenario("developer"),
  );
  expect(stages.status).toBe(200);
  await expect(stages.json()).resolves.toEqual({ data: [] });
});

test("filters vector deployments by landscape", async () => {
  const vectors = await get(
    "/api/v1/projects/payments-platform/vectorDeployments?landscapeId=development",
  );
  await expect(vectors.json()).resolves.toMatchObject({
    data: [{ id: "vector-dev-us30-1", status: "DeploymentReady" }],
  });
});

test("reports a vector deployment filter naming an unknown landscape as not found", async () => {
  const unknown = await get(
    "/api/v1/projects/payments-platform/vectorDeployments?landscapeId=nowhere",
  );
  expect(unknown.status).toBe(404);

  const empty = await get("/api/v1/projects/payments-platform/vectorDeployments?landscapeId=");
  expect(empty.status).toBe(404);
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
  await expect(identity.json()).resolves.toMatchObject({
    email: "riley.operator@example.com",
  });

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
  await expect(denied.json()).resolves.toMatchObject({
    error: { message: "Login denied" },
  });
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
