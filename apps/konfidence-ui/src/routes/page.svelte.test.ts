import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import DashboardPage from "./+page.svelte";

describe("dashboard page", () => {
  it("renders the dashboard heading", async () => {
    render(DashboardPage);

    await expect
      .element(page.getByRole("heading", { level: 1 }))
      .toHaveTextContent("Konfidence Dashboard");
  });
});
