import { createMockServer } from "./server.js";

const DEFAULT_PORT = 8091;
const port = Number(process.env.KONFIDENCE_MOCK_API_PORT ?? DEFAULT_PORT);
const server = await createMockServer();

await server.listen({ host: "127.0.0.1", port });
console.info(`Konfidence mock API listening on http://127.0.0.1:${port}`);

process.once("SIGINT", () => void server.close());
process.once("SIGTERM", () => void server.close());
