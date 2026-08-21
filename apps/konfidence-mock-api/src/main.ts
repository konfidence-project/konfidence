import { createMockServer } from "./server.js";

const DEFAULT_PORT = 8091;
const port = Number(process.env.KONFIDENCE_MOCK_API_PORT ?? DEFAULT_PORT);
const server = await createMockServer();

server.listen(port, "127.0.0.1", () => {
  globalThis.console.info(`Konfidence mock API listening on http://127.0.0.1:${port}`);
});

const shutdown = (): void => {
  server.close(() => process.exit(0));
};

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
