import { page } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import TableFixture from "./TableFixture.svelte";
import "../../../styles/test.css";

describe("<Table>", () => {
  it("renders header and body content", async () => {
    await render(TableFixture);
    await expect.element(page.getByText("Deployment")).toBeInTheDocument();
    await expect.element(page.getByText("artifact-1")).toBeInTheDocument();
  });

  it("supports a screen-reader caption", async () => {
    await render(TableFixture, { caption: "Artifact deployments" });
    await expect
      .element(page.getByRole("table", { name: "Artifact deployments" }))
      .toBeInTheDocument();
  });

  it("row fires onselect on click", async () => {
    const onselect = vi.fn();
    await render(TableFixture, { onselect });
    await page.getByTestId("row-1").click();
    expect(onselect).toHaveBeenCalledTimes(1);
  });

  it("row fires onselect on Enter", async () => {
    const onselect = vi.fn();
    await render(TableFixture, { onselect });
    const row = page.getByTestId("row-1").element() as HTMLElement;
    row.focus();
    row.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "Enter" }));
    expect(onselect).toHaveBeenCalledTimes(1);
  });

  describe("mode screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.classList.remove("dark");
    });

    for (const mode of ["light", "dark"]) {
      it(`renders the populated table in ${mode} mode`, async () => {
        await page.viewport(600, 240);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(TableFixture);
        await expect.element(page.getByTestId("table-frame")).toMatchScreenshot();
      });
    }
  });
});
