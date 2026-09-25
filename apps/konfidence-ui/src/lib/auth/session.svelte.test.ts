import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HTTP_INTERNAL_SERVER_ERROR, HTTP_OK, HTTP_UNAUTHORIZED } from "$lib/http-status";
import {
  createTestSession,
  identityBody,
  jsonResponse,
  mockFetchOnce,
  mockFetchReject,
} from "$lib/auth/session.test-helpers";

const gotoMock = vi.fn(async (_target: string): Promise<void> => undefined);

vi.mock("$app/navigation", () => ({
  goto: (target: string) => gotoMock(target),
}));

beforeEach(() => {
  gotoMock.mockClear();
});

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("session store", () => {
  it("starts in the idle status with no user", () => {
    const session = createTestSession();

    expect(session.status).toBe("idle");
    expect(session.user).toBeUndefined();
  });

  it("transitions to authenticated on a successful identity response", async () => {
    mockFetchOnce(jsonResponse(HTTP_OK, identityBody));
    const session = createTestSession();

    await session.refresh();

    expect(session.status).toBe("authenticated");
    expect(session.user?.name).toBe("Alex Admin");
    expect(session.user?.roles).toEqual(["admin", "dev"]);
  });

  it("transitions to unauthenticated on a 401", async () => {
    mockFetchOnce(jsonResponse(HTTP_UNAUTHORIZED, { error: { code: "401", message: "no" } }));
    const session = createTestSession();

    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.user).toBeUndefined();
    expect(session.error).toBeUndefined();
  });

  it("records an error on unexpected non-OK responses", async () => {
    mockFetchOnce(
      jsonResponse(HTTP_INTERNAL_SERVER_ERROR, { error: { code: "500", message: "oops" } }),
    );
    const session = createTestSession();

    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.error).toContain("500");
  });

  it("records an error when the request fails", async () => {
    mockFetchReject(new Error("network down"));
    const session = createTestSession();
    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.error).toBe("network down");
  });

  it("uses a relative return path when the API is same-origin", () => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", "/api");
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl("/projects/foo"), globalThis.location.origin);

    expect(loginUrl.origin).toBe(globalThis.location.origin);
    expect(loginUrl.pathname).toBe("/api/v1/login");
    expect(loginUrl.searchParams.get("return_url")).toBe("/projects/foo");
  });

  it("uses the root path when no returnTo path is provided", () => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", "/api");
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl(), globalThis.location.origin);

    expect(loginUrl.searchParams.get("return_url")).toBe("/");
  });

  it("uses an absolute UI return URL when the API is cross-origin", () => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", "https://api.example.com/api");
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl("/projects/foo"));

    expect(loginUrl.origin).toBe("https://api.example.com");
    expect(loginUrl.pathname).toBe("/api/v1/login");
    expect(loginUrl.searchParams.get("return_url")).toBe(
      new URL("/projects/foo", globalThis.location.origin).href,
    );
  });

  it("uses a relative return path when an absolute API URL is same-origin", () => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", `${globalThis.location.origin}/api`);
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl("/projects/foo"));

    expect(loginUrl.origin).toBe(globalThis.location.origin);
    expect(loginUrl.pathname).toBe("/api/v1/login");
    expect(loginUrl.searchParams.get("return_url")).toBe("/projects/foo");
  });

  it.each([
    "https://attacker.example.com/projects",
    "//attacker.example.com/projects",
    String.raw`/\attacker.example.com`,
    "projects/foo",
  ])("falls back to the root path for an unsafe returnTo value: %s", (returnTo) => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", "/api");
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl(returnTo), globalThis.location.origin);

    expect(loginUrl.searchParams.get("return_url")).toBe("/");
  });

  it("uses the absolute UI root for an unsafe target with a cross-origin API", () => {
    vi.stubEnv("VITE_KONFIDENCE_API_BASE_URL", "https://api.example.com/api");
    const session = createTestSession();
    const loginUrl = new URL(session.buildLoginUrl("https://attacker.example.com/projects"));

    expect(loginUrl.searchParams.get("return_url")).toBe(
      new URL("/", globalThis.location.origin).href,
    );
  });

  it("navigates to /login after signing out", async () => {
    mockFetchOnce(new Response("", { status: HTTP_OK }));
    const session = createTestSession();

    await session.signOut();

    expect(gotoMock).toHaveBeenCalledWith("/login");
    expect(session.user).toBeUndefined();
    expect(session.status).toBe("unauthenticated");
  });
});
