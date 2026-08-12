import { type ServerResponse, createServer } from "node:http";
import {
  artifactDeploymentsByProject,
  landscapesByProject,
  projects,
  stagesByProject,
  vectorDeploymentsByProject,
} from "./fixtures";

const DEFAULT_PORT = 8091;
const SESSION_COOKIE = "kden_session";
const SCENARIO_COOKIE = "konfidence_mock_scenario";
const HTTP_OK = 200;
const HTTP_FOUND = 302;
const HTTP_UNAUTHORIZED = 401;
const HTTP_FORBIDDEN = 403;
const HTTP_NOT_FOUND = 404;
const HTTP_SERVICE_UNAVAILABLE = 503;

const json = (response: ServerResponse, status: number, body: unknown): void => {
  response.writeHead(status, { "content-type": "application/json" });
  response.end(JSON.stringify(body));
};

const cookiesFrom = (header: string | undefined): Record<string, string> =>
  Object.fromEntries(
    (header ?? "")
      .split(";")
      .map((part) => part.trim())
      .filter(Boolean)
      .map((part) => {
        const separator = part.indexOf("=");
        return [part.slice(0, separator), decodeURIComponent(part.slice(separator + 1))];
      }),
  );

const errorResponse = (response: ServerResponse, status: number, message: string): void =>
  json(response, status, { error: { code: String(status), message } });

const safeReturnTo = (value: string | null): string => {
  if (!value?.startsWith("/") || value.startsWith("//")) {
    return "/";
  }
  return value;
};

const server = createServer((request, response) => {
  const url = new URL(request.url ?? "/", "http://mock-api.local");
  const cookies = cookiesFrom(request.headers.cookie);
  const scenario = cookies[SCENARIO_COOKIE] ?? process.env.KONFIDENCE_MOCK_SCENARIO ?? "default";
  const authenticated = cookies[SESSION_COOKIE] === "mock-session";

  if (url.pathname === "/health") {
    json(response, HTTP_OK, { status: "ok" });
    return;
  }
  if (url.pathname === "/v1/login" && request.method === "GET") {
    response.writeHead(HTTP_FOUND, {
      location: safeReturnTo(url.searchParams.get("return_url")),
      "set-cookie": `${SESSION_COOKIE}=mock-session; Path=/; HttpOnly; SameSite=Lax`,
    });
    response.end();
    return;
  }
  if (url.pathname === "/v1/auth/callback" && request.method === "GET") {
    response.writeHead(HTTP_FOUND, { location: "/" });
    response.end();
    return;
  }
  if (url.pathname === "/v1/logout" && request.method === "POST") {
    if (!authenticated) {
      errorResponse(response, HTTP_UNAUTHORIZED, "Authentication required");
      return;
    }
    response.writeHead(HTTP_OK, {
      "content-type": "application/json",
      "set-cookie": `${SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`,
    });
    response.end(JSON.stringify({}));
    return;
  }
  if (url.pathname === "/v1/identity" && request.method === "GET") {
    if (!authenticated || scenario === "unauthenticated") {
      errorResponse(response, HTTP_UNAUTHORIZED, "Authentication required");
      return;
    }
    let roles = ["ADMIN", "DEV"];
    if (scenario === "forbidden") {
      roles = ["VIEWER"];
    }
    json(response, HTTP_OK, {
      email: "alex@example.com",
      familyName: "Example",
      givenName: "Alex",
      name: "Alex Example",
      roles,
    });
    return;
  }

  if (!authenticated) {
    errorResponse(response, HTTP_UNAUTHORIZED, "Authentication required");
    return;
  }
  if (scenario === "forbidden") {
    errorResponse(response, HTTP_FORBIDDEN, "Access denied");
    return;
  }
  if (scenario === "api-error" && url.pathname.startsWith("/v1/projects")) {
    errorResponse(response, HTTP_SERVICE_UNAVAILABLE, "Mock API unavailable");
    return;
  }
  if (scenario === "invalid-response" && url.pathname === "/v1/projects") {
    json(response, HTTP_OK, { unexpected: true });
    return;
  }
  if (url.pathname === "/v1/projects" && request.method === "GET") {
    let data = projects;
    if (scenario === "no-projects") {
      data = [];
    } else if (scenario === "one-project") {
      data = [projects[0]];
    }
    json(response, HTTP_OK, { data });
    return;
  }

  const match = url.pathname.match(
    /^\/v1\/projects\/(?<projectId>[^/]+)\/(?<resource>landscapes|stages|vectorDeployments|artifactDeployments)$/,
  );
  if (!match?.groups) {
    errorResponse(response, HTTP_NOT_FOUND, "Not found");
    return;
  }

  const { projectId, resource } = match.groups;
  if (!projects.some((project) => project.id === projectId)) {
    errorResponse(response, HTTP_NOT_FOUND, "Project not found");
    return;
  }

  const dataByResource = {
    artifactDeployments: artifactDeploymentsByProject[projectId] ?? [],
    landscapes: landscapesByProject[projectId] ?? [],
    stages: stagesByProject[projectId] ?? [],
    vectorDeployments: vectorDeploymentsByProject[projectId] ?? [],
  };
  json(response, HTTP_OK, { data: dataByResource[resource as keyof typeof dataByResource] });
});

const port = Number(process.env.KONFIDENCE_MOCK_API_PORT ?? DEFAULT_PORT);
server.listen(port, "127.0.0.1", () => {
  globalThis.console.info(`Konfidence mock API listening on http://127.0.0.1:${port}`);
});

const shutdown = (): void => {
  server.close(() => process.exit(0));
};
process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
