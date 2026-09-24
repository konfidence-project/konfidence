import { page } from "vitest/browser";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import LoginPage from "./+page.svelte";
import SessionTestProvider from "$lib/auth/SessionTestProvider.svelte";
import { createTestSession } from "$lib/auth/session.test-helpers";

const pageState = {
  url: new URL("http://127.0.0.1/login"),
};

vi.mock("$app/state", () => ({
  page: {
    get url(): URL {
      return pageState.url;
    },
  },
}));

vi.mock("$app/navigation", () => ({
  goto: vi.fn(async () => undefined),
}));

const setLoginUrl = (search: string): void => {
  pageState.url = new URL(`http://127.0.0.1/login${search}`);
};

beforeEach(() => {
  setLoginUrl("");
  vi.unstubAllGlobals();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("login page", () => {
  it("renders a sign-in link that targets the default return path", async () => {
    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    const link = page.getByTestId("sign-in");
    await expect.element(link).toBeVisible();

    const href = await link.element().getAttribute("href");
    expect(href).not.toBeNull();

    const loginUrl = new URL(href!, globalThis.location.origin);
    expect(loginUrl.pathname).toBe("/api/v1/login");
    expect(loginUrl.searchParams.get("return_url")).toBe("/");
  });

  it("propagates a returnTo path into the sign-in URL", async () => {
    setLoginUrl("?returnTo=/projects/foo/landscape");

    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    const link = page.getByTestId("sign-in");
    const href = await link.element().getAttribute("href");
    expect(href).not.toBeNull();

    const loginUrl = new URL(href!, globalThis.location.origin);
    expect(loginUrl.searchParams.get("return_url")).toBe("/projects/foo/landscape");
  });

  it("falls back to the root path for an external returnTo target", async () => {
    setLoginUrl("?returnTo=https%3A%2F%2Fattacker.example.com%2Fprojects");

    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    const link = page.getByTestId("sign-in");
    const href = await link.element().getAttribute("href");
    expect(href).not.toBeNull();

    const loginUrl = new URL(href!, globalThis.location.origin);
    expect(loginUrl.searchParams.get("return_url")).toBe("/");
  });

  it("renders the error description from the callback query", async () => {
    setLoginUrl("?error=access_denied&error_description=Login%20denied");

    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    await expect.element(page.getByRole("alert")).toHaveTextContent("Login denied");
  });

  it("falls back to the error code when no description is provided", async () => {
    setLoginUrl("?error=access_denied");

    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    await expect.element(page.getByRole("alert")).toHaveTextContent("access_denied");
  });
});
