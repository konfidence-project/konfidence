import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { resetSessionForTest, session } from "$lib/auth/session.svelte";

const gotoMock = vi.fn(async (_target: string): Promise<void> => undefined);

vi.mock("$app/navigation", () => ({
  goto: (target: string) => gotoMock(target),
}));

const HTTP_OK = 200;
const HTTP_UNAUTHORIZED = 401;
const HTTP_INTERNAL_SERVER_ERROR = 500;

const mockFetchOnce = (response: Response): void => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => response),
  );
};

const jsonResponse = (status: number, body: unknown): Response =>
  new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status,
  });

const identityBody = {
  email: "alex.admin@example.com",
  familyName: "Admin",
  givenName: "Alex",
  name: "Alex Admin",
  projectRoles: { "payments-platform": ["admin", "dev"] },
};

beforeEach(() => {
  resetSessionForTest();
  gotoMock.mockClear();
});

afterEach(() => {
  vi.unstubAllGlobals();
  resetSessionForTest();
});

describe("session store", () => {
  it("starts in the idle status with no user", () => {
    expect(session.status).toBe("idle");
    expect(session.user).toBeUndefined();
  });

  it("transitions to authenticated on a successful identity response", async () => {
    mockFetchOnce(jsonResponse(HTTP_OK, identityBody));

    await session.refresh();

    expect(session.status).toBe("authenticated");
    expect(session.user?.name).toBe("Alex Admin");
    expect(session.user?.roles).toEqual(["admin", "dev"]);
  });

  it("transitions to unauthenticated on a 401", async () => {
    mockFetchOnce(jsonResponse(HTTP_UNAUTHORIZED, { error: { code: "401", message: "no" } }));

    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.user).toBeUndefined();
    expect(session.error).toBeUndefined();
  });

  it("records an error on unexpected non-OK responses", async () => {
    mockFetchOnce(
      jsonResponse(HTTP_INTERNAL_SERVER_ERROR, { error: { code: "500", message: "oops" } }),
    );

    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.error).toContain("500");
  });

  it("records an error when the request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new Error("network down");
      }),
    );

    await session.refresh();

    expect(session.status).toBe("unauthenticated");
    expect(session.error).toBe("network down");
  });

  it("builds the login URL from a returnTo path", () => {
    const url = session.buildLoginUrl("/projects/foo");

    expect(url).toContain("/api/v1/login?return_url=");
    expect(url).toContain(
      encodeURIComponent(new URL("/projects/foo", globalThis.location.origin).href),
    );
  });

  it("navigates to /login after signing out", async () => {
    mockFetchOnce(new Response("", { status: HTTP_OK }));

    await session.signOut();

    expect(gotoMock).toHaveBeenCalledWith("/login");
    expect(session.user).toBeUndefined();
    expect(session.status).toBe("unauthenticated");
  });
});
