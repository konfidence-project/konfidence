import { page, userEvent } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../styles/test.css";
import TreeGroupFixture from "./TreeGroupFixture.svelte";

describe("<TreeGroup>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  it("selects one row per group and retains selection on repeated clicks and collapse", async () => {
    const onValueChange = vi.fn();
    const onOpenChange = vi.fn();
    await render(TreeGroupFixture, { onOpenChange, onValueChange });
    const header = page.getByRole("button", { name: "Changed Files 3 files" });
    const index = page.getByRole("button", { name: "src/index.ts modified TypeScript" });
    const pkg = page.getByRole("button", { name: "package.json" });

    await expect.element(pkg).toHaveAttribute("aria-pressed", "true");
    await index.click();
    await expect.element(index).toHaveAttribute("aria-pressed", "true");
    await expect.element(pkg).toHaveAttribute("aria-pressed", "false");
    await expect.element(page.getByTestId("selection")).toHaveTextContent("index");
    await index.click();
    expect(onValueChange).toHaveBeenCalledExactlyOnceWith("index");

    await page.getByRole("button", { name: "Other file" }).click();
    await expect.element(index).toHaveAttribute("aria-pressed", "true");
    await header.click();
    await expect.element(header).toHaveAttribute("aria-expanded", "false");
    await expect.element(page.getByTestId("open")).toHaveTextContent("false");
    expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(false);
    await header.click();
    await expect.element(index).toHaveAttribute("aria-pressed", "true");
  });

  it("supports keyboard selection and removes collapsed rows from the tab order", async () => {
    await render(TreeGroupFixture);
    const header = page.getByRole("button", { name: "Changed Files 3 files" });
    await header.click();
    await userEvent.keyboard("{Tab}");
    await expect.element(page.getByRole("button", { name: "Other group" })).toHaveFocus();
    await userEvent.keyboard("{Shift>}{Tab}{/Shift}");
    await userEvent.keyboard("{Enter}");
    await expect.element(header).toHaveAttribute("aria-expanded", "true");
    await userEvent.keyboard("{Tab} ");
    await expect.element(page.getByTestId("selection")).toHaveTextContent("index");
    await expect.element(page.getByRole("button", { name: "Unavailable file" })).toBeDisabled();
  });

  it("responds to external state changes without firing user callbacks", async () => {
    const onValueChange = vi.fn();
    const onOpenChange = vi.fn();
    await render(TreeGroupFixture, { onOpenChange, onValueChange });
    await page.getByRole("button", { name: "Reset externally" }).click();
    await expect.element(page.getByTestId("selection")).toHaveTextContent("none");
    await expect
      .element(page.getByRole("button", { name: "Changed Files 3 files" }))
      .toHaveAttribute("aria-expanded", "false");
    expect(onValueChange).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("connects each disclosure to a unique list", async () => {
    await render(TreeGroupFixture);
    const headers = [...document.querySelectorAll<HTMLButtonElement>("button[aria-controls]")];
    const ids = headers.map((header) => header.getAttribute("aria-controls"));
    expect(new Set(ids).size).toBe(headers.length);
    for (const id of ids) {
      expect(document.getElementById(id ?? "")?.tagName).toBe("UL");
    }
  });

  describe("variant screenshots", () => {
    for (const mode of ["light", "dark"] as const) {
      for (const variant of ["expanded", "collapsed", "narrow"] as const) {
        it(`renders ${mode} ${variant}`, async () => {
          await page.viewport(variant === "narrow" ? 280 : 420, 480);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.classList.toggle("dark", mode === "dark");
          await render(TreeGroupFixture);
          if (variant === "collapsed") {
            await page.getByRole("button", { name: "Changed Files 3 files" }).click();
          }
          await expect
            .element(page.getByRole("group", { exact: true, name: "Changed Files" }))
            .toMatchScreenshot();
        });
      }
    }
  });
});
