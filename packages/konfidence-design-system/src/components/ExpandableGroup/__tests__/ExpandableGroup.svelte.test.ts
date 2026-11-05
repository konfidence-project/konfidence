import { page, userEvent } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import GroupFixture from "./GroupFixture.svelte";
import "../../../styles/test.css";

describe("grouped rows public interface", () => {
  it("places expanded rows directly below the header separator", async () => {
    await render(GroupFixture, { withSummary: false });
    const header = page.getByRole("button", { name: "Example group" });
    await header.click();
    const row = page.getByRole("button", { name: "Select history" });
    await expect.element(row).toBeVisible();
    const headerBox = header.element().getBoundingClientRect();
    const rowBox = row.element().parentElement!.getBoundingClientRect();
    expect(rowBox.top - headerBox.bottom).toBeLessThanOrEqual(1);
  });
  it("uses an outlined right/down chevron for expansion", async () => {
    await render(GroupFixture);
    const trigger = page.getByRole("button", { name: "Example group" });
    await expect
      .element(trigger.element().querySelector<HTMLElement>("ui5-icon"))
      .toHaveAttribute("name", "slim-arrow-right");
    await trigger.click();
    await expect
      .element(trigger.element().querySelector<HTMLElement>("ui5-icon"))
      .toHaveAttribute("name", "slim-arrow-down");
  });
  it("supports header-only collapse without an always-visible row", async () => {
    await render(GroupFixture, { withSummary: false });
    const trigger = page.getByRole("button", { name: "Example group" });
    await expect.element(page.getByText("Always visible summary")).not.toBeInTheDocument();
    await expect.element(page.getByText("Historical item")).not.toBeVisible();
    await trigger.click();
    await expect.element(page.getByRole("button", { name: "Select history" })).toBeVisible();
    await trigger.click();
    await expect.element(page.getByText("Historical item")).not.toBeVisible();
  });
  it("keeps the summary visible and controls expansion with native keyboard interaction", async () => {
    const rendered = await render(GroupFixture);
    const trigger = page.getByRole("button", { name: "Example group" });
    await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
    await expect.element(page.getByRole("button", { name: "Select summary" })).toBeVisible();
    await expect.element(page.getByText("Historical item")).not.toBeVisible();
    await trigger.click();
    await expect.element(trigger).toHaveAttribute("aria-expanded", "true");
    await rendered.rerender({ title: "Updated content" });
    await expect.element(page.getByRole("button", { name: "Select history" })).toBeVisible();
    (trigger.element() as HTMLButtonElement).focus();
    await userEvent.keyboard(" ");
    await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
    await userEvent.keyboard("{Enter}");
    await expect.element(trigger).toHaveAttribute("aria-expanded", "true");
  });

  it("omits the trigger without additional content", async () => {
    await render(GroupFixture, { extra: false });
    await expect
      .element(page.getByRole("button", { name: "Example group" }))
      .not.toBeInTheDocument();
    await expect.element(page.getByRole("button", { name: "Select summary" })).toBeVisible();
  });

  it("separates actions from selection and expansion", async () => {
    const onselect = vi.fn();
    const onaction = vi.fn();
    await render(GroupFixture, { onaction, onselect });
    const row = page.getByRole("button", { name: "Select summary" });
    await page.getByRole("button", { exact: true, name: "Action" }).click();
    await userEvent.keyboard("{Enter}");
    expect(onaction).toHaveBeenCalledTimes(2);
    expect(onselect).not.toHaveBeenCalled();
    await expect
      .element(page.getByRole("button", { exact: true, name: "Disabled" }))
      .toBeDisabled();
    await expect
      .element(page.getByRole("button", { name: "Example group" }))
      .toHaveAttribute("aria-expanded", "false");
    (row.element() as HTMLButtonElement).focus();
    await userEvent.keyboard(" ");
    expect(onselect).toHaveBeenCalledOnce();
    await expect.element(row).toHaveAttribute("aria-pressed", "true");
    await expect.element(row).toHaveFocus();
    await userEvent.tab();
    await expect.element(page.getByRole("button", { exact: true, name: "Action" })).toHaveFocus();
    await userEvent.tab();
    await expect
      .element(page.getByRole("button", { includeHidden: true, name: "Select history" }))
      .not.toHaveFocus();
  });

  it("allows independent expansion of multiple groups", async () => {
    await render(GroupFixture);
    await render(GroupFixture);
    const firstTrigger = page.getByRole("button", { name: "Example group" }).nth(0);
    const secondTrigger = page.getByRole("button", { name: "Example group" }).nth(1);
    await firstTrigger.click();
    await expect.element(secondTrigger).toHaveAttribute("aria-expanded", "false");
    await secondTrigger.click();
    await expect.element(firstTrigger).toHaveAttribute("aria-expanded", "true");
    await expect.element(secondTrigger).toHaveAttribute("aria-expanded", "true");
  });

  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  for (const mode of ["light", "dark"]) {
    for (const width of [360, 760]) {
      it(`renders ${mode} at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture, {
          title: "group-with-a-very-long-identifier-that-wraps-without-hiding-actions-or-selection",
        });
        await page.getByRole("button", { name: "Example group" }).click();
        await page.getByRole("button", { name: "Select summary" }).click();
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
      });
      it(`renders ${mode} header-only at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture, { withSummary: false });
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
        await page.getByRole("button", { name: "Example group" }).click();
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
      });
      it(`renders ${mode} inset row hover at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture, { inset: true });
        const fixture = page.getByTestId("fixture");
        // Capture the final hover state, not a transition or its temporary compositing layer.
        fixture.element().style.setProperty("--motion-fast", "0ms");
        await page.getByRole("button", { name: "Select summary" }).hover();
        expect(fixture.element().getAnimations({ subtree: true })).toHaveLength(0);
        await expect.element(fixture).toMatchScreenshot();
      });
    }
  }
});
