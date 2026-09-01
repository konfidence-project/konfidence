import { vi } from "vitest";
import { createApiClient } from "$lib/konfidence-api/client";
import { SessionStore } from "$lib/auth/session.svelte";

/**
 * Canonical identity payload matching `components.schemas.Identity` in the
 * OpenAPI spec. Used across auth tests to represent a successfully
 * authenticated user.
 */
const identityBody = {
  email: "alex.admin@example.com",
  familyName: "Admin",
  givenName: "Alex",
  name: "Alex Admin",
  projectRoles: { "payments-platform": ["admin", "dev"] },
};

const jsonResponse = (status: number, body: unknown): Response =>
  new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status,
  });

const mockFetchOnce = (response: Response): void => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => response),
  );
};

const mockFetchReject = (error: Error): void => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw error;
    }),
  );
};

/**
 * Creates a fresh {@link SessionStore} wired to its own {@link createApiClient}.
 * The unauthorized middleware is bound to the returned store, matching the
 * production wiring in `+layout.svelte` without touching the module-level
 * singleton in `client-instance.ts`.
 */
const createTestSession = (): SessionStore => {
  let handler: (() => void) | undefined;
  const client = createApiClient({
    onUnauthorized: () => handler?.(),
  });
  const store = new SessionStore(client);
  handler = () => store.handleUnauthorized();
  return store;
};

export { createTestSession, identityBody, jsonResponse, mockFetchOnce, mockFetchReject };
