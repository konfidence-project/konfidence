import { page, userEvent } from "vitest/browser";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import GroupFixture from "./GroupFixture.svelte";
import SelectionFixture from "./SelectionFixture.svelte";
import RowOutsideGroup from "./RowOutsideGroup.svelte";
import "../../../styles/test.css";

describe("grouped rows public interface", () => {
  beforeEach(() => {
    document.documentElement.setAttribute("data-theme", "konfidence");
  });

  it("keeps behavioral readouts and controls outside screenshot regions", async () => {
    await render(GroupFixture);
    expect(
      page.getByTestId("fixture").element().contains(page.getByTestId("action-count").element()),
    ).toBe(false);
    await render(SelectionFixture);
    const selection = page.getByTestId("selection-fixture").element();
    expect(selection.contains(page.getByTestId("selected-value").element())).toBe(false);
    expect(
      selection.contains(page.getByRole("button", { name: "Reset externally" }).element()),
    ).toBe(false);
    expect(
      selection.contains(page.getByRole("button", { name: "Select externally" }).element()),
    ).toBe(false);
    expect(selection.contains(page.getByRole("button", { name: "Toggle groups" }).element())).toBe(
      false,
    );
  });

  it("shows a plain header by default and an independent action only in the interactive example", async () => {
    const rendered = await render(GroupFixture);
    await expect
      .element(page.getByRole("button", { name: "Header action" }))
      .not.toBeInTheDocument();
    await rendered.rerender({ withHeaderAction: true });
    await expect.element(page.getByRole("button", { name: "Header action" })).toBeVisible();
  });

  it("expands from free block header content outside the disclosure button", async () => {
    await render(GroupFixture);
    const heading = page.getByText("Example group", { exact: true });
    expect(heading.element().closest("button")).toBeNull();
    await heading.click();
    const trigger = page.getByRole("button", { name: "Example group" });
    await expect.element(trigger).toHaveAttribute("aria-expanded", "true");
    await heading.click();
    await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
  });

  it("aligns inset row content with the free header's chevron gutter", async () => {
    await page.viewport(760, 480);
    await render(GroupFixture, { initialExpanded: true, inset: true });
    const header = page
      .getByText("Example group", { exact: true })
      .element()
      .getBoundingClientRect();
    const row = page
      .getByText("Historical item", { exact: true })
      .element()
      .getBoundingClientRect();
    expect(Math.abs(header.left - row.left)).toBeLessThanOrEqual(1);
  });

  for (const withHeaderAction of [false, true]) {
    it(`vertically centers a single-line header ${withHeaderAction ? "with" : "without"} an action`, async () => {
      await page.viewport(760, 480);
      await render(GroupFixture, { withHeaderAction });
      const trigger = page.getByRole("button", { name: "Example group" }).element();
      const header = trigger.parentElement!.getBoundingClientRect();
      const title = page
        .getByText("Example group", { exact: true })
        .element()
        .getBoundingClientRect();
      const icon = trigger.querySelector<HTMLElement>("ui5-icon")!.getBoundingClientRect();
      const center = header.top + header.height / 2;
      expect(Math.abs(title.top + title.height / 2 - center)).toBeLessThanOrEqual(1);
      expect(Math.abs(icon.top + icon.height / 2 - center)).toBeLessThanOrEqual(1);
    });
  }

  for (const width of [360, 760]) {
    it(`vertically centers file types beside the complete row content at ${width}px`, async () => {
      await page.viewport(width, 720);
      await render(SelectionFixture);
      await page.getByRole("button", { name: "Selectable files" }).click();
      for (const [name, type] of [
        ["First file", "JSON"],
        ["Second file", "TypeScript"],
      ]) {
        const row = page.getByRole("button", { name }).element().getBoundingClientRect();
        const content = page.getByText(type, { exact: true }).element().getBoundingClientRect();
        expect(
          Math.abs(content.top + content.height / 2 - row.top - row.height / 2),
        ).toBeLessThanOrEqual(1);
      }
    });
  }

  it("toggles once per chevron click, Enter or Space and hides collapsed rows from Tab", async () => {
    const rendered = await render(GroupFixture, { withHeaderAction: true });
    const trigger = page.getByRole("button", { name: "Example group" });
    const row = page.getByRole("button", { includeHidden: true, name: "Select history" });
    await expect.element(row).not.toBeVisible();
    await expect
      .element(trigger.element().querySelector<HTMLElement>("ui5-icon"))
      .toHaveAttribute("name", "slim-arrow-right");
    await trigger.click();
    await expect.element(trigger).toHaveAttribute("aria-expanded", "true");
    await expect
      .element(trigger.element().querySelector<HTMLElement>("ui5-icon"))
      .toHaveAttribute("name", "slim-arrow-down");
    await rendered.rerender({ title: "Updated content" });
    await expect.element(row).toBeVisible();
    (trigger.element() as HTMLElement).focus();
    await userEvent.keyboard(" ");
    await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
    await userEvent.tab();
    await expect.element(page.getByRole("button", { name: "Header action" })).toHaveFocus();
    await userEvent.tab();
    await expect.element(row).not.toHaveFocus();
    (trigger.element() as HTMLElement).focus();
    await userEvent.keyboard("{Enter}");
    await expect.element(trigger).toHaveAttribute("aria-expanded", "true");
  });

  it("keeps independent header actions outside the trigger and leaves selection and expansion unchanged", async () => {
    await render(GroupFixture, { withHeaderAction: true });
    const action = page.getByRole("button", { name: "Header action" });
    expect(action.element().parentElement!.closest("button")).toBeNull();
    await action.click();
    await userEvent.keyboard("{Enter}");
    await expect.element(page.getByTestId("action-count")).toHaveTextContent("2");
    await expect
      .element(page.getByRole("button", { name: "Example group" }))
      .toHaveAttribute("aria-expanded", "false");
    await page.getByRole("button", { name: "Example group" }).click();
    await expect
      .element(page.getByRole("button", { name: "Select history" }))
      .toHaveAttribute("aria-pressed", "false");
  });

  it("makes empty groups static, programmatically focusable and not tab stops", async () => {
    await render(GroupFixture, { extra: false });
    await expect
      .element(page.getByRole("button", { name: "Example group" }))
      .not.toBeInTheDocument();
    const header = document.getElementById("example-header")!;
    expect(header.tabIndex).toBe(-1);
    header.focus();
    await expect.element(header).toHaveFocus();
    await page.getByText("Example group", { exact: true }).click();
    await expect.element(page.getByText("Historical item")).not.toBeInTheDocument();
  });

  it("expands and selects locally in independent groups with distinct disclosure relationships", async () => {
    await render(GroupFixture);
    await render(GroupFixture);
    const triggers = page.getByRole("button", { name: "Example group" });
    await triggers.nth(0).click();
    const first = page.getByRole("button", { name: "Select history" });
    await first.click();
    await expect.element(first).toHaveAttribute("aria-pressed", "true");
    await expect.element(triggers.nth(1)).toHaveAttribute("aria-expanded", "false");
    await triggers.nth(1).click();
    const second = page.getByRole("button", { name: "Select history" }).nth(1);
    await expect.element(second).toHaveAttribute("aria-pressed", "false");
    await second.click();
    await expect
      .element(page.getByRole("button", { name: "Select history" }).nth(0))
      .toHaveAttribute("aria-pressed", "true");
    const ids = [0, 1].map((index) => triggers.nth(index).element().getAttribute("aria-controls")!);
    expect(ids[0]).not.toBe(ids[1]);
    for (const id of ids) {
      expect(document.getElementById(id)).not.toBeNull();
    }
  });

  it("shares parent selection through ordinary and function bindings and preserves it on repeat and collapse", async () => {
    await render(SelectionFixture);
    const trigger = page.getByRole("button", { name: "Selectable files" });
    await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
    await trigger.click();
    const first = page.getByRole("button", { name: "First file" });
    const second = page.getByRole("button", { includeHidden: true, name: "Second file" });
    await expect.element(first).toHaveAttribute("aria-pressed", "true");
    await second.click();
    await expect.element(first).toHaveAttribute("aria-pressed", "false");
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    await second.click();
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    const other = page.getByRole("button", { exact: true, name: "Other file" });
    await other.click();
    await expect.element(second).toHaveAttribute("aria-pressed", "false");
    await expect.element(other).toHaveAttribute("aria-pressed", "true");
    await second.click();
    await trigger.click();
    await expect.element(second).not.toBeVisible();
    await trigger.click();
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
  });

  it("updates external selection and reset without losing parent state on unmount or remount", async () => {
    await render(SelectionFixture);
    await page.getByRole("button", { name: "Select externally" }).click();
    await expect
      .element(page.getByRole("button", { name: "Selectable files" }))
      .toHaveAttribute("aria-expanded", "false");
    await page.getByRole("button", { name: "Toggle groups" }).click();
    await expect.element(page.getByTestId("selected-value")).toHaveTextContent("second");
    await page.getByRole("button", { name: "Toggle groups" }).click();
    await page.getByRole("button", { name: "Selectable files" }).click();
    await expect
      .element(page.getByRole("button", { name: "Second file" }))
      .toHaveAttribute("aria-pressed", "true");
    await page.getByRole("button", { name: "Reset externally" }).click();
    await expect.element(page.getByTestId("selected-value")).toHaveTextContent("none");
    await page.getByRole("button", { name: "Selectable files" }).click();
    await expect
      .element(page.getByRole("button", { name: "Second file" }))
      .toHaveAttribute("aria-pressed", "false");
  });

  it("keeps row actions independent, with their own disabled state", async () => {
    await render(GroupFixture, { initialExpanded: true });
    const row = page.getByRole("button", { name: "Select history" });
    await page.getByRole("button", { exact: true, name: "Action" }).click();
    await userEvent.keyboard("{Enter}");
    await expect.element(page.getByTestId("action-count")).toHaveTextContent("2");
    await expect.element(row).toHaveAttribute("aria-pressed", "false");
    await expect.element(page.getByRole("button", { name: "Disabled" })).toBeDisabled();
    await expect
      .element(page.getByRole("button", { name: "Example group" }))
      .toHaveAttribute("aria-expanded", "true");
  });

  it("rejects a row outside a group with a descriptive programming error", async () => {
    await expect(render(RowOutsideGroup)).rejects.toThrow(
      "SelectableRow must be rendered inside an ExpandableGroup.",
    );
  });

  it("selects through content and keeps a full-height marker, stable selected hover and natural height", async () => {
    await render(SelectionFixture);
    page.getByTestId("selection-fixture").element().style.setProperty("--motion-fast", "0ms");
    await page.getByRole("button", { name: "Selectable files" }).click();
    await page.getByText("TypeScript", { exact: true }).click();
    const second = page.getByRole("button", { name: "Second file" });
    await expect.element(second).toHaveAttribute("aria-pressed", "true");
    const row = second.element().parentElement!;
    const style = getComputedStyle(row, "::before");
    expect(style.width).toBe("3px");
    expect(style.top).toBe("-1px");
    expect(style.bottom).toBe("0px");
    expect(row.getBoundingClientRect().height).toBeGreaterThanOrEqual(64);
    const background = getComputedStyle(row).backgroundColor;
    await second.hover();
    expect(getComputedStyle(row).backgroundColor).toBe(background);
    expect(
      getComputedStyle(page.getByRole("group", { exact: true, name: "Selectable files" }).element())
        .borderTopLeftRadius,
    ).toBe("14px");
  });

  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  for (const mode of ["light", "dark"]) {
    for (const width of [360, 760]) {
      it(`renders ${mode} selectable files at ${width}px`, async () => {
        await page.viewport(width, 720);
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(SelectionFixture);
        await page.getByRole("button", { name: "Selectable files" }).click();
        await expect.element(page.getByTestId("selection-fixture")).toMatchScreenshot();
      });
      it(`renders ${mode} at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture, {
          title: "group-with-a-very-long-identifier-that-wraps-without-hiding-actions-or-selection",
          withHeaderAction: true,
        });
        // Capture settled selection paint rather than a frame of its color transition.
        page.getByTestId("fixture").element().style.setProperty("--motion-fast", "0ms");
        await page.getByRole("button", { name: "Example group" }).click();
        await page.getByRole("button", { name: "Select history" }).click();
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
      });
      it(`renders ${mode} header-only at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture);
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
        await page.getByRole("button", { name: "Example group" }).click();
        await expect.element(page.getByTestId("fixture")).toMatchScreenshot();
      });
      it(`renders ${mode} inset row hover at ${width}px`, async () => {
        await page.viewport(width, 480);
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(GroupFixture, { initialExpanded: true, inset: true });
        const fixture = page.getByTestId("fixture");
        fixture.element().style.setProperty("--motion-fast", "0ms");
        await page.getByRole("button", { name: "Select history" }).hover();
        expect(fixture.element().getAnimations({ subtree: true })).toHaveLength(0);
        await expect.element(fixture).toMatchScreenshot();
      });
    }
  }
});
