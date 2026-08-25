import { type IncomingMessage, type Server, createServer } from "node:http";
import { fileURLToPath } from "node:url";
import { OpenAPIBackend, type Request as OpenAPIRequest } from "openapi-backend";
import { errorResponse, handlers, securityHandlers, writeResponse } from "./handlers.js";

const OPENAPI_PATH = fileURLToPath(new URL("../../../api/openapi.yaml", import.meta.url));

const readBody = async (request: IncomingMessage): Promise<string | undefined> => {
  const chunks: Buffer[] = [];
  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return chunks.length === 0 ? undefined : Buffer.concat(chunks).toString("utf8");
};

const createMockServer = async (): Promise<Server> => {
  const api = new OpenAPIBackend({
    ajvOpts: { validateFormats: false },
    apiRoot: "/api",
    definition: OPENAPI_PATH,
    handlers,
    securityHandlers,
    strict: true,
    validate: true,
  });

  await api.init();

  return createServer(async (request, response) => {
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

export { createMockServer };
