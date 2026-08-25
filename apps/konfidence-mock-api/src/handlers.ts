import type { IncomingMessage, ServerResponse } from "node:http";
import type { Context } from "openapi-backend";
import { scenarios, type ProjectFixture, type ScenarioFixture } from "./fixtures.js";
import type { operations } from "./generated/schema.js";

const SESSION_COOKIE = "kden-session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";
const MOCK_SESSION = "mock-session";
const MOCK_CODE = "mock-code";

interface MockResponse {
  body?: unknown;
  headers?: Record<string, string>;
  status: number;
}

type MockHandler = (context: Context) => MockResponse;

const jsonResponse = (status: number, body: unknown): MockResponse => ({ body, status });
const errorResponse = (status: number, message: string): MockResponse =>
  jsonResponse(status, { error: { code: String(status), message } });

const writeResponse = (response: ServerResponse, mockResponse: MockResponse): void => {
  const headers = { ...mockResponse.headers };
  if (mockResponse.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  response.writeHead(mockResponse.status, headers);
  response.end(mockResponse.body === undefined ? undefined : JSON.stringify(mockResponse.body));
};

const parseUrl = (value: string): URL | undefined => {
  try {
    return new URL(value);
  } catch {
    return undefined;
  }
};

const scenarioFrom = (context: Context): ScenarioFixture => {
  switch (context.request.cookies[SCENARIO_COOKIE]) {
    case "degraded": {
      return scenarios.degraded;
    }
    case "developer": {
      return scenarios.developer;
    }
    default: {
      return scenarios.admin;
    }
  }
};

const withProject = (
  context: Context,
  handle: (project: ProjectFixture) => MockResponse,
): MockResponse => {
  const scenario = scenarioFrom(context);
  if (scenario.resourcesUnavailable) {
    return errorResponse(500, "Mock API unavailable");
  }
  const projectId = String(context.request.params.projectId);
  const project = scenario.projects.find((item) => item.project.id === projectId);
  return project ? handle(project) : errorResponse(403, "Access denied");
};

const operationHandlers = {
  authCallbackV1: (context) => {
    const { error, error_description: errorDescription } = context.request.query;
    if (error) {
      return errorResponse(401, String(errorDescription ?? error));
    }
    const returnUrl = parseUrl(
      Buffer.from(String(context.request.query.state), "base64url").toString("utf8"),
    );
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
    const scenario = scenarioFrom(context);
    const projectRoles = Object.fromEntries(
      scenario.projects.map(({ project, roles }) => [project.id, roles]),
    );
    return jsonResponse(200, { ...scenario.user, projectRoles });
  },
  listArtifactDeploymentsV1: (context) =>
    withProject(context, (project) => {
      const { landscapeId, vectorDeploymentId } = context.request.query;
      const data = project.artifactDeployments.filter(
        (deployment) =>
          (!landscapeId || deployment.landscapeId === landscapeId) &&
          (!vectorDeploymentId ||
            deployment.vectorDeploymentIds.includes(String(vectorDeploymentId))),
      );
      return jsonResponse(200, { data });
    }),
  listLandscapesV1: (context) =>
    withProject(context, (project) => jsonResponse(200, { data: project.landscapes })),
  listProjectsV1: (context) => {
    const projects = scenarioFrom(context).projects.map(({ project }) => project);
    return jsonResponse(200, { data: projects });
  },
  listStagesV1: (context) =>
    withProject(context, (project) => {
      const { landscapeId } = context.request.query;
      const data = project.stages.filter(
        (stage) => !landscapeId || stage.landscapeId === landscapeId,
      );
      return jsonResponse(200, { data });
    }),
  listVectorDeploymentsV1: (context) =>
    withProject(context, (project) => {
      const { landscapeId } = context.request.query;
      const data = project.vectorDeployments.filter(
        (deployment) => !landscapeId || deployment.landscapeId === landscapeId,
      );
      return jsonResponse(200, { data });
    }),
  loginV1: (context) => {
    const returnUrl = parseUrl(String(context.request.query.return_url));
    if (!returnUrl) {
      return errorResponse(400, "Invalid return URL");
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
  postExchangeCodeV1: () => ({
    headers: {
      "Set-Cookie": `${SESSION_COOKIE}=${MOCK_SESSION}; Path=/; HttpOnly; SameSite=Lax`,
    },
    status: 200,
  }),
} satisfies Record<keyof operations, MockHandler>;

const isAuthenticated = (context: Context): boolean =>
  context.request.cookies[SESSION_COOKIE] === MOCK_SESSION;

const handlers = {
  ...operationHandlers,
  methodNotAllowed: () => errorResponse(405, "Method not allowed"),
  notFound: () => errorResponse(404, "Not found"),
  postResponseHandler: (context: Context, _request: IncomingMessage, response: ServerResponse) =>
    writeResponse(response, context.response as MockResponse),
  unauthorizedHandler: () => errorResponse(401, "Authentication required"),
  validationFail: () => errorResponse(400, "Invalid request"),
};

const securityHandlers = { sessionCookie: isAuthenticated };

export { errorResponse, handlers, securityHandlers, writeResponse };
