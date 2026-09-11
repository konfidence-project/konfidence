import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import EmptyState from "./EmptyState.svelte";
import EmptyStateFixture from "./__tests__/EmptyStateFixture.svelte";
import "../../../../../apps/konfidence-ui/src/app.css";

const TONES = ["empty", "info", "error"] as const;

describe("<EmptyState>", () => {
  it("renders as a status region by default", async () => {
    render(EmptyState, { title: "No results" });
    const status = page.getByRole("status");
    await expect.element(status).toBeInTheDocument();
    await expect.element(status).toHaveTextContent("No results");
  });

  it("renders as an alert when tone is error", async () => {
    render(EmptyState, { title: "Something broke", tone: "error" });
    await expect.element(page.getByRole("alert")).toHaveTextContent("Something broke");
  });

  it("surfaces the description when supplied", async () => {
    render(EmptyState, {
      description: "Try clearing filters.",
      title: "No results",
    });
    await expect.element(page.getByText("Try clearing filters.")).toBeInTheDocument();
  });

  describe("tone screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      for (const tone of TONES) {
        it(`renders the ${mode} ${tone} tone`, async () => {
          await page.viewport(360, 260);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(EmptyStateFixture, {
            description:
              tone === "error"
                ? "Artifact deployments are currently unavailable."
                : "This project does not have any artifact deployments yet.",
            title:
              tone === "error" ? "Failed to load artifact deployments" : "No artifact deployments",
            tone,
          });
          const role = tone === "error" ? "alert" : "status";
          await expect.element(page.getByRole(role)).toMatchScreenshot();
        });
      }
    }
  });
});
