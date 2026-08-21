import { type IncomingMessage, type Server, type ServerResponse, createServer } from "node:http";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import type { FormatsPlugin } from "ajv-formats";
import {
  type Context,
  OpenAPIBackend,
  type Request as OpenAPIRequest,
  SetMatchType,
} from "openapi-backend";
import { fixtures } from "./fixtures.js";
import type { operations } from "./generated/schema.js";

const OPENAPI_PATH = fileURLToPath(new URL("../../../api/openapi.yaml", import.meta.url));
const addFormats = createRequire(import.meta.url)("ajv-formats") as FormatsPlugin;
const SESSION_COOKIE = "kden-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";
const MOCK_SESSION = "mock-session";
const MOCK_CODE = "mock-code";
const MOCK_VERIFIER = "mock-verifier";

type OperationId = keyof operations;
type Scenario =
  | "default"
  | "forbidden"
  | "internal-error"
  | "no-projects"
  | "one-project"
  | "unauthenticated";

interface MockResponse {
  body?: unknown;
  headers?: Record<string, string>;
  status: number;
}

type MockHandler = (context: Context) => MockResponse;

interface MockServerOptions {
  operationHandlers?: Partial<Record<OperationId, MockHandler>>;
  scenario?: Scenario;
}

const jsonResponse = (status: number, body: unknown): MockResponse => ({ body, status });
const errorResponse = (status: number, message: string): MockResponse =>
  jsonResponse(status, { error: { code: String(status), message } });

const normalizeScenario = (value: string | undefined): Scenario => {
  switch (value) {
    case "forbidden":
    case "internal-error":
    case "no-projects":
    case "one-project":
    case "unauthenticated": {
      return value;
    }
    default: {
      return "default";
    }
  }
};

const scenarioFrom = (context: Context, fallback: Scenario): Scenario =>
  normalizeScenario(context.request.cookies[SCENARIO_COOKIE] ?? fallback);

const isAllowedReturnUrl = (returnUrl: URL): boolean => {
  if (returnUrl.protocol !== "http:" && returnUrl.protocol !== "https:") {
    return false;
  }
  if (returnUrl.hostname === "127.0.0.1" || returnUrl.hostname === "localhost") {
    return true;
  }
  const allowedOrigins = (process.env.KONFIDENCE_MOCK_ALLOWED_RETURN_ORIGINS ?? "")
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean);
  return allowedOrigins.includes(returnUrl.origin);
};

const decodeReturnUrl = (state: string): URL | undefined => {
  try {
    const returnUrl = new URL(Buffer.from(state, "base64url").toString("utf8"));
    if (isAllowedReturnUrl(returnUrl)) {
      return returnUrl;
    }
    return undefined;
  } catch {
    return undefined;
  }
};

const projectItems = <Item>(itemsByProject: Record<string, Item[]>, projectId: string) =>
  itemsByProject[projectId];

const createOperationHandlers = (defaultScenario: Scenario) => {
  const projectAccess = (context: Context): MockResponse | undefined => {
    const scenario = scenarioFrom(context, defaultScenario);
    if (scenario === "internal-error") {
      return errorResponse(500, "Mock API unavailable");
    }
    const projectId = String(context.request.params.projectId);
    if (
      scenario === "forbidden" ||
      !fixtures.projects.some((project) => project.id === projectId)
    ) {
      return errorResponse(403, "Access denied");
    }
    return undefined;
  };

  const handlers = {
    authCallbackV1: (context) => {
      const { code, state } = context.request.query;
      if (code !== MOCK_CODE) {
        return errorResponse(401, "Invalid authorization code");
      }
      const returnUrl = decodeReturnUrl(String(state));
      if (!returnUrl) {
        return errorResponse(400, "Invalid authentication state");
      }
      return {
        headers: {
          Location: returnUrl.href,
          "Set-Cookie": `${SESSION_COOKIE}=${MOCK_SESSION}; Path=/; HttpOnly; SameSite=Lax`,
        },
        status: 302,
      };
    },
    getIdentityV1: (context) => {
      const projectRoles =
        scenarioFrom(context, defaultScenario) === "forbidden"
          ? {}
          : Object.fromEntries(fixtures.projects.map((project) => [project.id, ["admin", "dev"]]));
      return jsonResponse(200, {
        email: "alex@example.com",
        familyName: "Example",
        givenName: "Alex",
        name: "Alex Example",
        projectRoles,
      });
    },
    listArtifactDeploymentsV1: (context) => {
      const denied = projectAccess(context);
      if (denied) {
        return denied;
      }
      const projectId = String(context.request.params.projectId);
      const { landscapeId, vectorDeploymentId } = context.request.query;
      const data = projectItems(fixtures.artifactDeploymentsByProject, projectId)!.filter(
        (deployment) =>
          (!landscapeId || deployment.landscapeId === landscapeId) &&
          (!vectorDeploymentId ||
            deployment.vectorDeploymentIds.includes(String(vectorDeploymentId))),
      );
      return jsonResponse(200, { data });
    },
    listLandscapesV1: (context) => {
      const denied = projectAccess(context);
      if (denied) {
        return denied;
      }
      const projectId = String(context.request.params.projectId);
      return jsonResponse(200, {
        data: projectItems(fixtures.landscapesByProject, projectId),
      });
    },
    listProjectsV1: (context) => {
      const scenario = scenarioFrom(context, defaultScenario);
      if (scenario === "internal-error") {
        return errorResponse(500, "Mock API unavailable");
      }
      let data = fixtures.projects;
      if (scenario === "no-projects") {
        data = [];
      } else if (scenario === "one-project") {
        data = fixtures.projects.slice(0, 1);
      }
      return jsonResponse(200, { data });
    },
    listStagesV1: (context) => {
      const denied = projectAccess(context);
      if (denied) {
        return denied;
      }
      const projectId = String(context.request.params.projectId);
      const { landscapeId } = context.request.query;
      const data = projectItems(fixtures.stagesByProject, projectId)!.filter(
        (stage) => !landscapeId || stage.landscapeId === landscapeId,
      );
      return jsonResponse(200, { data });
    },
    listVectorDeploymentsV1: (context) => {
      const denied = projectAccess(context);
      if (denied) {
        return denied;
      }
      const projectId = String(context.request.params.projectId);
      const { landscapeId } = context.request.query;
      const data = projectItems(fixtures.vectorDeploymentsByProject, projectId)!.filter(
        (deployment) => !landscapeId || deployment.landscapeId === landscapeId,
      );
      return jsonResponse(200, { data });
    },
    loginV1: (context) => {
      let returnUrl: URL;
      try {
        returnUrl = new URL(String(context.request.query.return_url));
      } catch {
        return errorResponse(400, "Invalid return URL");
      }
      if (!isAllowedReturnUrl(returnUrl)) {
        return errorResponse(400, "Return URL is not allowed");
      }
      const state = Buffer.from(returnUrl.href).toString("base64url");
      return {
        headers: {
          Location: `/api/v1/auth/callback?code=${MOCK_CODE}&state=${encodeURIComponent(state)}`,
        },
        status: 302,
      };
    },
    logoutV1: () => ({
      headers: {
        "Set-Cookie": `${SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`,
      },
      status: 200,
    }),
    postExchangeCodeV1: (context) => {
      const requestBody = context.request.requestBody as { code?: string; verifier?: string };
      if (requestBody.code !== MOCK_CODE || requestBody.verifier !== MOCK_VERIFIER) {
        return errorResponse(401, "Invalid exchange code or verifier");
      }
      return jsonResponse(200, { id: MOCK_SESSION });
    },
  } satisfies Record<OperationId, MockHandler>;

  return handlers;
};

const writeResponse = (response: ServerResponse, mockResponse: MockResponse): void => {
  const headers = { ...mockResponse.headers };
  if (mockResponse.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  response.writeHead(mockResponse.status, headers);
  response.end(mockResponse.body === undefined ? undefined : JSON.stringify(mockResponse.body));
};

const readBody = async (request: IncomingMessage): Promise<string | undefined> => {
  const chunks: Buffer[] = [];
  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return chunks.length === 0 ? undefined : Buffer.concat(chunks).toString("utf8");
};

const createMockApi = async (options: MockServerOptions = {}) => {
  const scenario = normalizeScenario(options.scenario ?? process.env.KONFIDENCE_MOCK_SCENARIO);
  const api = new OpenAPIBackend({
    apiRoot: "/api",
    customizeAjv: (ajv) => addFormats(ajv),
    definition: OPENAPI_PATH,
    strict: true,
    validate: true,
  });

  const operationHandlers: Record<OperationId, MockHandler> = {
    ...createOperationHandlers(scenario),
    ...options.operationHandlers,
  };
  api.register(operationHandlers);
  api.registerSecurityHandler("sessionCookie", (context: Context) => {
    const selectedScenario = scenarioFrom(context, scenario);
    return (
      selectedScenario !== "unauthenticated" &&
      context.request.cookies[SESSION_COOKIE] === MOCK_SESSION
    );
  });
  api.register({
    methodNotAllowed: (_context: Context) => errorResponse(405, "Method not allowed"),
    notFound: (_context: Context) => errorResponse(404, "Not found"),
    postResponseHandler: (
      context: Context,
      _request: IncomingMessage,
      response: ServerResponse,
    ) => {
      const mockResponse = context.response as MockResponse;
      if (!context.operation) {
        writeResponse(response, mockResponse);
        return;
      }
      const documentedResponses = context.operation.responses ?? {};
      if (
        !(String(mockResponse.status) in documentedResponses) &&
        !("default" in documentedResponses)
      ) {
        globalThis.console.error("Mock response uses an undocumented status", {
          operationId: context.operation.operationId,
          status: mockResponse.status,
        });
        writeResponse(response, {
          body: { error: { code: "500", message: "Mock response violates the OpenAPI contract" } },
          headers: { "X-Konfidence-Mock-Validation-Error": "true" },
          status: 500,
        });
        return;
      }
      const bodyValidation = context.api.validateResponse(
        mockResponse.body,
        context.operation,
        mockResponse.status,
      );
      const headerValidation = context.api.validateResponseHeaders(
        mockResponse.headers ?? {},
        context.operation,
        { setMatchType: SetMatchType.Superset, statusCode: mockResponse.status },
      );
      if (bodyValidation.errors || headerValidation.errors) {
        globalThis.console.error("Mock response violates the OpenAPI contract", {
          body: bodyValidation.errors,
          headers: headerValidation.errors,
          operationId: context.operation.operationId,
          status: mockResponse.status,
        });
        writeResponse(response, {
          body: { error: { code: "500", message: "Mock response violates the OpenAPI contract" } },
          headers: { "X-Konfidence-Mock-Validation-Error": "true" },
          status: 500,
        });
        return;
      }
      writeResponse(response, mockResponse);
    },
    unauthorizedHandler: (_context: Context) => errorResponse(401, "Authentication required"),
    validationFail: (context: Context) => {
      const status = context.operation?.responses?.["400"] ? 400 : 500;
      return errorResponse(status, "Invalid request");
    },
  });
  await api.init();
  return { api, operationHandlers };
};

const createMockServer = async (options: MockServerOptions = {}): Promise<Server> => {
  const { api } = await createMockApi(options);
  return createServer(async (request, response) => {
    if (request.url === "/health") {
      writeResponse(response, jsonResponse(200, { status: "ok" }));
      return;
    }
    try {
      const body = await readBody(request);
      const headers = Object.fromEntries(
        Object.entries(request.headers).filter(
          (entry): entry is [string, string | string[]] => entry[1] !== undefined,
        ),
      );
      const openAPIRequest: OpenAPIRequest = {
        body,
        headers,
        method: request.method ?? "GET",
        path: request.url ?? "/",
      };
      await api.handleRequest(openAPIRequest, request, response);
    } catch (error) {
      globalThis.console.error("Mock API request failed", error);
      if (!response.headersSent) {
        writeResponse(response, errorResponse(500, "Mock API request failed"));
      } else {
        response.end();
      }
    }
  });
};

export { createMockApi, createMockServer };
export type { MockServerOptions, Scenario };
