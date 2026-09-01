import { page } from "vitest/browser";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import DashboardPage from "./+page.svelte";
import SessionTestProvider from "$lib/auth/SessionTestProvider.svelte";
import { createTestSession } from "$lib/auth/session.test-helpers";

vi.mock("$app/navigation", () => ({
  goto: vi.fn(async () => undefined),
}));

beforeEach(() => {
  vi.unstubAllGlobals();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("dashboard page", () => {
  it("renders the dashboard heading", async () => {
    render(SessionTestProvider, {
      component: DashboardPage,
      session: createTestSession(),
    });

    await expect
      .element(page.getByRole("heading", { level: 1 }))
      .toHaveTextContent("Konfidence Dashboard");
  });

  it("renders a sign-out link", async () => {
    render(SessionTestProvider, {
      component: DashboardPage,
      session: createTestSession(),
    });

    await expect.element(page.getByTestId("sign-out")).toHaveAttribute("href", "/logout");
  });
});
