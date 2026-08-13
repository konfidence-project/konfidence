import createOpenApiClient from "openapi-fetch";
import type { paths } from "$lib/konfidence-api/schema";

const api = createOpenApiClient<paths>({ baseUrl: "/api" });

export default api;
