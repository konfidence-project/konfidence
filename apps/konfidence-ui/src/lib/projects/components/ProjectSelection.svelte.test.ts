import { page } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import ProjectsTestProvider from "$lib/projects/components/ProjectsTestProvider.svelte";

vi.mock("$app/state", () => ({
  page: { url: new URL("http://127.0.0.1/projects?embedded=1") },
}));

describe("project selection", () => {
  it("renders accessible project links and preserves embedded mode", async () => {
    render(ProjectsTestProvider, {
      projects: [
        { id: "payments", name: "Payments" },
        { id: "identity", name: "Identity" },
      ],
    });

    await expect
      .element(page.getByRole("link", { name: "Payments" }))
      .toHaveAttribute("href", "/projects/payments/landscape?embedded=1");
    await expect.element(page.getByRole("link", { name: "Identity" })).toBeVisible();
  });
});
