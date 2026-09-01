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

  it("builds the login URL from a returnTo path", () => {
    const session = createTestSession();

    const url = session.buildLoginUrl("/projects/foo");

    expect(url).toContain("/api/v1/login?return_url=");
    expect(url).toContain(
      encodeURIComponent(new URL("/projects/foo", globalThis.location.origin).href),
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
