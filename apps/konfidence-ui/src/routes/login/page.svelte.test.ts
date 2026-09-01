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
  it("renders a sign-in link that targets the default return URL", async () => {
    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    const link = page.getByTestId("sign-in");
    await expect.element(link).toBeVisible();
    const href = await link.element().getAttribute("href");
    expect(href).toContain("/api/v1/login?return_url=");
    expect(href).toContain(encodeURIComponent(new URL("/", globalThis.location.origin).href));
  });

  it("propagates a returnTo query parameter into the sign-in URL", async () => {
    setLoginUrl("?returnTo=/projects/foo/landscape");

    render(SessionTestProvider, {
      component: LoginPage,
      session: createTestSession(),
    });

    const link = page.getByTestId("sign-in");
    const href = await link.element().getAttribute("href");
    expect(href).toContain(
      encodeURIComponent(new URL("/projects/foo/landscape", globalThis.location.origin).href),
    );
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
