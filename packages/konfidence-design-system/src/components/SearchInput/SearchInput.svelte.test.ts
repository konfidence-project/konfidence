import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import SearchInput from "./SearchInput.svelte";
import SearchInputFixture from "./__tests__/SearchInputFixture.svelte";
import "../../../../../apps/konfidence-ui/src/app.css";

describe("<SearchInput>", () => {
  it("renders a searchbox with the default placeholder", async () => {
    render(SearchInput, { "aria-label": "Search deployments" });
    const box = page.getByRole("searchbox", { name: "Search deployments" });
    await expect.element(box).toBeInTheDocument();
    await expect.element(box).toHaveAttribute("placeholder", "Search\u2026");
  });

  it("shows the clear button when a value is present", async () => {
    render(SearchInput, { "aria-label": "s", value: "hello" });
    await expect.element(page.getByTestId("search-clear")).toBeInTheDocument();
  });

  it("hides the clear button when the value is empty", async () => {
    render(SearchInput, { "aria-label": "s" });
    await expect.element(page.getByTestId("search-clear")).not.toBeInTheDocument();
  });

  describe("state screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      for (const value of ["", "payments-api"]) {
        const state = value === "" ? "empty" : "with-value";
        it(`renders the ${mode} ${state} state`, async () => {
          await page.viewport(400, 160);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(SearchInputFixture, { value });
          await expect
            .element(page.getByRole("searchbox", { name: "Search artifact deployments" }))
            .toMatchScreenshot();
        });
      }
    }
  });
});
