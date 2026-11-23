import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import TogglePill from "../TogglePill.svelte";
import "../../../styles/test.css";

describe("TogglePill", () => {
  it("updates its read-only label when the value changes and retains consumer classes", async () => {
    const screen = await render(TogglePill, { class: "consumer-class", on: true });
    await expect.element(page.getByText("On", { exact: true })).toHaveClass("consumer-class");
    await expect.element(page.getByRole("switch")).not.toBeInTheDocument();
    await screen.rerender({ class: "consumer-class", on: false });
    await expect.element(page.getByText("Off", { exact: true })).toHaveClass("consumer-class");
    await expect.element(page.getByText("On", { exact: true })).not.toBeInTheDocument();
    await expect.element(page.getByRole("switch")).not.toBeInTheDocument();
  });

  it("uses consumer healthy tokens for the On pill", async () => {
    const screen = await render(TogglePill, { on: true });
    screen.container.style.setProperty("--status-healthy-fg", "rgb(20, 40, 60)");
    screen.container.style.setProperty("--status-healthy-bg", "rgb(220, 230, 240)");
    const pill = page.getByText("On", { exact: true }).element();
    expect(getComputedStyle(pill).color).toBe("rgb(20, 40, 60)");
    expect(getComputedStyle(pill).backgroundColor).toBe("rgb(220, 230, 240)");
  });
});
