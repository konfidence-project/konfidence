import { page, userEvent } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import GroupFixture from "./GroupFixture.svelte";
import SelectionFixture from "./SelectionFixture.svelte";
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

  it("selects one value per group, suppresses repeated changes and retains selection on collapse", async () => {
    const onValueChange = vi.fn();
    const onselect = vi.fn();
    await render(SelectionFixture, { onValueChange, onselect });
    const first = page.getByRole("button", { name: "First file" });
    const second = page.getByRole("button", { includeHidden: true, name: "Second file" });
    await expect.element(first).toHaveAttribute("aria-pressed", "true");
    await second.click();
    await expect.element(first).toHaveAttribute("aria-pressed", "false");
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    await expect.element(page.getByTestId("selected-value")).toHaveTextContent("second");
    await second.click();
    expect(onValueChange).toHaveBeenCalledExactlyOnceWith("second");
    expect(onselect).toHaveBeenCalledOnce();
    await page.getByRole("button", { exact: true, name: "Other file" }).click();
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    const trigger = page.getByRole("button", { name: "Selectable files" });
    await trigger.click();
    await expect.element(second).not.toBeVisible();
    await trigger.click();
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
  });

  it("keeps external reset silent and supports standalone application-controlled selection", async () => {
    const onValueChange = vi.fn();
    const onOpenChange = vi.fn();
    await render(SelectionFixture, { onOpenChange, onValueChange });
    const standalone = page.getByRole("button", { name: "Standalone file" });
    await standalone.click();
    await expect.element(standalone).toHaveAttribute("aria-pressed", "true");
    await standalone.click();
    await expect.element(standalone).toHaveAttribute("aria-pressed", "false");
    await page.getByRole("button", { name: "Reset externally" }).click();
    await expect.element(page.getByTestId("selected-value")).toHaveTextContent("none");
    await expect.element(page.getByTestId("expanded-value")).toHaveTextContent("false");
    await expect
      .element(page.getByRole("button", { name: "Selectable files" }))
      .toHaveAttribute("aria-expanded", "false");
    expect(onValueChange).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("disables row selection while keeping independent actions separate", async () => {
    const onValueChange = vi.fn();
    const onselect = vi.fn();
    const onaction = vi.fn();
    await render(SelectionFixture, { onValueChange, onaction, onselect });
    const disabled = page.getByRole("button", { name: "Disabled file" });
    await expect.element(disabled).toBeDisabled();
    (disabled.element() as HTMLButtonElement).click();
    expect(onselect).not.toHaveBeenCalled();
    expect(onValueChange).not.toHaveBeenCalled();
    await page.getByRole("button", { name: "Inspect file" }).click();
    expect(onaction).toHaveBeenCalledOnce();
    expect(onselect).not.toHaveBeenCalled();
    expect(onValueChange).not.toHaveBeenCalled();
    await expect
      .element(page.getByRole("button", { name: "Selectable files" }))
      .toHaveAttribute("aria-expanded", "true");
    const second = page.getByRole("button", { name: "Second file" });
    (second.element() as HTMLButtonElement).focus();
    await userEvent.keyboard(" ");
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    await userEvent.tab();
    await expect.element(disabled).not.toHaveFocus();
  });

  it("selects via trailing content and paints a full-height marker without changing row height", async () => {
    await render(SelectionFixture);
    const second = page.getByRole("button", { name: "Second file" });
    await page.getByText("TypeScript", { exact: true }).click();
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    const row = second.element().parentElement!;
    const style = getComputedStyle(row, "::before");
    expect(style.width).toBe("3px");
    expect(style.top).toBe("-1px");
    expect(style.bottom).toBe("0px");
    expect(row.getBoundingClientRect().height).toBeGreaterThanOrEqual(64);
    const group = page.getByRole("group", { exact: true, name: "Selectable files" }).element();
    expect(getComputedStyle(group).borderTopLeftRadius).toBe("14px");
  });

  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  for (const mode of ["light", "dark"]) {
    for (const width of [360, 760]) {
      it(`renders ${mode} selectable files at ${width}px`, async () => {
        await page.viewport(width, 720);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(SelectionFixture);
        await expect.element(page.getByTestId("selection-fixture")).toMatchScreenshot();
      });
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
