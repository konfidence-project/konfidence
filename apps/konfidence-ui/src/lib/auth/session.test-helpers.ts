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

const jsonResponse = (status: number, body: unknown): Response => Response.json(body, { status });

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
  // Indirection through an object lets the middleware close over the store
  // that is created immediately after the client, avoiding a TDZ reference.
  const holder: { store?: SessionStore } = {};
  const client = createApiClient({
    onUnauthorized: () => holder.store?.handleUnauthorized(),
  });
  holder.store = new SessionStore(client);
  return holder.store;
};

export { createTestSession, identityBody, jsonResponse, mockFetchOnce, mockFetchReject };
