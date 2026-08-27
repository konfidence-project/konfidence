import { readFile } from "node:fs/promises";
import type { IncomingMessage, ServerResponse } from "node:http";
import { dirname, join } from "node:path";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const SWAGGER_UI_PATH = dirname(require.resolve("swagger-ui-dist/package.json"));
const PERMANENT_REDIRECT = 308;

const DOCUMENTATION_HTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Konfidence Mock API</title>
    <link rel="stylesheet" href="/docs/swagger-ui.css">
    <style>
      .mock-docs-header { align-items: center; background: #17223b; color: #fff; display: flex; font-family: sans-serif; gap: 1rem; justify-content: space-between; padding: .75rem 1.5rem; }
      .mock-docs-header div { align-items: baseline; display: flex; flex-wrap: wrap; gap: .75rem; }
      .mock-docs-header a { background: #fff; border-radius: .25rem; color: #17223b; font-weight: 600; padding: .5rem .75rem; text-decoration: none; white-space: nowrap; }
      @media (max-width: 40rem) { .mock-docs-header { align-items: flex-start; flex-direction: column; } }
    </style>
  </head>
  <body>
    <header class="mock-docs-header">
      <div><strong>Konfidence Mock API</strong><span>Use a mock session to run authenticated requests.</span></div>
      <a href="/docs/login">Start mock session</a>
    </header>
    <div id="swagger-ui"></div>
    <script src="/docs/swagger-ui-bundle.js"></script>
    <script>
      SwaggerUIBundle({
        deepLinking: true,
        displayRequestDuration: true,
        dom_id: "#swagger-ui",
        url: "/docs/openapi.yaml",
        withCredentials: true,
      });
    </script>
  </body>
</html>
`;

const serveSwagger = async (
  request: IncomingMessage,
  response: ServerResponse,
  openapiPath: string,
): Promise<boolean> => {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return false;
  }

  const send = (contentType: string, body: Buffer | string): void => {
    response.writeHead(200, { "Content-Type": contentType });
    response.end(request.method === "HEAD" ? undefined : body);
  };
  const { pathname } = new URL(request.url ?? "/", "http://localhost");
  if (pathname === "/docs") {
    response.writeHead(PERMANENT_REDIRECT, { Location: "/docs/" });
    response.end();
    return true;
  }
  if (pathname === "/docs/login") {
    const returnUrl = `http://${request.headers.host ?? "127.0.0.1"}/docs/`;
    response.writeHead(302, {
      Location: `/api/v1/login?return_url=${encodeURIComponent(returnUrl)}`,
    });
    response.end();
    return true;
  }
  if (pathname === "/docs/") {
    send("text/html; charset=utf-8", DOCUMENTATION_HTML);
    return true;
  }
  if (pathname === "/docs/openapi.yaml") {
    send("application/yaml", await readFile(openapiPath));
    return true;
  }
  if (pathname === "/docs/swagger-ui.css") {
    send("text/css; charset=utf-8", await readFile(join(SWAGGER_UI_PATH, "swagger-ui.css")));
    return true;
  }
  if (pathname === "/docs/swagger-ui-bundle.js") {
    send(
      "text/javascript; charset=utf-8",
      await readFile(join(SWAGGER_UI_PATH, "swagger-ui-bundle.js")),
    );
    return true;
  }
  return false;
};

export { serveSwagger };
