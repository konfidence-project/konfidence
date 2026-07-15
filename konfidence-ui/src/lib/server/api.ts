import { env } from "$env/dynamic/private";

const DEFAULT_API_URL = "http://localhost:8090";

const getApiUrl = (): string => (env.KONFIDENCE_API_URL ?? DEFAULT_API_URL).replace(/\/$/, "");

export default getApiUrl;
