import { page } from "vitest/browser";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import DashboardPage from "./+page.svelte";
import { resetSessionForTest } from "$lib/auth/session.svelte";

vi.mock("$app/navigation", () => ({
  goto: vi.fn(async () => undefined),
}));

beforeEach(() => {
  resetSessionForTest();
});

afterEach(() => {
  resetSessionForTest();
});

describe("dashboard page", () => {
  it("renders the dashboard heading", async () => {
    render(DashboardPage);

    await expect
      .element(page.getByRole("heading", { level: 1 }))
      .toHaveTextContent("Konfidence Dashboard");
  });

  it("renders a sign-out link", async () => {
    render(DashboardPage);

    await expect.element(page.getByTestId("sign-out")).toHaveAttribute("href", "/logout");
  });
});
