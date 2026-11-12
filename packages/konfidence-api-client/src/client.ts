import createOpenApiClient, {
  type Client,
  type ClientOptions,
  type Middleware,
} from "openapi-fetch";
import { withConflictRetries } from "./retry.js";
import type { paths } from "./schema.js";

type KonfidenceApiClient = Client<paths>;
type KonfidenceApiClientOptions = ClientOptions;

const createKonfidenceApiClient = (options: KonfidenceApiClientOptions = {}): KonfidenceApiClient =>
  createOpenApiClient<paths>({
    ...options,
    fetch: withConflictRetries(options.fetch ?? globalThis.fetch),
  });

export { createKonfidenceApiClient };
export type { KonfidenceApiClient, KonfidenceApiClientOptions, Middleware };
