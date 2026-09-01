import cookie from "@fastify/cookie";
import swagger from "@fastify/swagger";
import swaggerUI from "@fastify/swagger-ui";
import fastify, { type FastifyError, type FastifyInstance } from "fastify";
import openapiGlue from "fastify-openapi-glue";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { operationHandlers, securityHandlers } from "./handlers.js";

const OPENAPI_PATH = fileURLToPath(new URL("../../../api/openapi.yaml", import.meta.url));

// Swagger UI cannot start a session on its own, so point it at the mock login redirect.
const DOCS_BANNER = `document.addEventListener("DOMContentLoaded", () => {
  document.body.insertAdjacentHTML(
    "afterbegin",
    '<p style="font-family:sans-serif;padding:1rem"><a href="/docs/login">Start a mock session</a> before running authenticated requests.</p>',
  );
});
`;

const createMockServer = async (): Promise<FastifyInstance> => {
  const server = fastify({
    ajv: { customOptions: { coerceTypes: false, validateFormats: false } },
  });

  server.setErrorHandler((error: FastifyError, _request, reply) => {
    const status = error.statusCode ?? 500;
    const message = error.name === "Unauthorized" ? "Authentication required" : error.message;
    return reply.code(status).send({ error: { code: String(status), message } });
  });

  server.setNotFoundHandler((_request, reply) =>
    reply.code(404).send({ error: { code: "404", message: "Not found" } }),
  );

  await server.register(cookie);
  await server.register(swagger, {
    mode: "static",
    specification: { baseDir: dirname(OPENAPI_PATH), path: OPENAPI_PATH },
  });

  server.get("/docs/login", (request, reply) => {
    const returnUrl = `http://${request.headers.host ?? "127.0.0.1"}/docs/`;
    return reply.redirect(`/api/v1/login?return_url=${encodeURIComponent(returnUrl)}`);
  });

  await server.register(swaggerUI, {
    routePrefix: "/docs",
    theme: {
      js: [{ content: DOCS_BANNER, filename: "mock-api.js" }],
      title: "Konfidence Mock API",
    },
    uiConfig: { withCredentials: true },
  });

  await server.register(openapiGlue, {
    prefix: "/api",
    securityHandlers,
    serviceHandlers: operationHandlers,
    specification: OPENAPI_PATH,
  });
  await server.ready();

  return server;
};

export { createMockServer };
