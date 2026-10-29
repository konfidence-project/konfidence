import { page, userEvent } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import BottomSheetFixture from "./BottomSheetFixture.svelte";
import "../../../styles/test.css";

describe("<BottomSheet>", () => {
  it("stays closed until asked to open", async () => {
    await render(BottomSheetFixture);
    const panel = page.getByTestId("bottom-sheet");
    await expect.element(panel).not.toBeVisible();
  });

  it("opens on demand and exposes an accessible title", async () => {
    await render(BottomSheetFixture, { open: true, title: "dev → prod" });
    const panel = page.getByRole("dialog", { name: "dev → prod" });
    await expect.element(panel).toBeVisible();
    await expect.element(page.getByTestId("panel-body")).toBeVisible();
  });

  it("can open non-modally and closes with Escape from the surrounding page", async () => {
    await page.viewport(1200, 800);
    const onClose = vi.fn();
    await render(BottomSheetFixture, { modal: false, onClose, open: true });
    await expect.element(page.getByTestId("bottom-sheet")).toHaveAttribute("aria-modal", "false");
    await vi.waitFor(() =>
      expect(document.activeElement).toBe(page.getByTestId("bottom-sheet-close").element()),
    );

    const outside = page.getByTestId("outside-btn").element() as HTMLButtonElement;
    outside.focus();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByTestId("bottom-sheet")).not.toBeVisible();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("closes when the close button is clicked and fires onClose", async () => {
    const onClose = vi.fn();
    await render(BottomSheetFixture, { onClose, open: true });
    await page.getByTestId("bottom-sheet-close").click();
    await expect.element(page.getByTestId("bottom-sheet")).not.toBeVisible();
    expect(onClose).toHaveBeenCalled();
  });

  it("closes on a backdrop click", async () => {
    const onClose = vi.fn();
    await render(BottomSheetFixture, { onClose, open: true });
    await page.viewport(1200, 800);
    await expect.element(page.getByTestId("bottom-sheet")).toBeVisible();
    // oxlint-disable-next-line eslint/id-length -- Playwright names the coordinate fields x/y.
    await page.getByTestId("bottom-sheet-backdrop").click({ position: { x: 600, y: 50 } });
    await expect.element(page.getByTestId("bottom-sheet")).not.toBeVisible();
    expect(onClose).toHaveBeenCalled();
  });

  it("renders the footer snippet when provided", async () => {
    await render(BottomSheetFixture, { open: true });
    await expect.element(page.getByTestId("panel-footer")).toBeInTheDocument();
  });

  it("scrolls the body when content overflows", async () => {
    await page.viewport(1200, 600);
    await render(BottomSheetFixture, { open: true, overflow: true });
    await expect.element(page.getByTestId("bottom-sheet")).toBeVisible();
    const body = page.getByTestId("panel-body").element().parentElement!
      .parentElement as HTMLElement;
    expect(getComputedStyle(body).overflow).toContain("auto");
    expect(body.scrollHeight).toBeGreaterThan(body.clientHeight);
  });

  it("slides up from the bottom and parks off-screen when closed", async () => {
    await page.viewport(1200, 600);
    await render(BottomSheetFixture, { open: true });
    const dialog = page.getByTestId("bottom-sheet").element() as HTMLElement;
    await expect.element(page.getByTestId("bottom-sheet")).toBeVisible();
    const style = getComputedStyle(dialog);
    expect(style.transitionProperty).toContain("translate");
    expect(style.transitionDuration).not.toBe("0s");
  });

  describe("vertical resize", () => {
    it("exposes resize bounds on a horizontal separator for assistive tech", async () => {
      await page.viewport(1200, 800);
      await render(BottomSheetFixture, { height: 400, open: true });
      const resizer = page.getByTestId("bottom-sheet-resizer").element() as HTMLElement;
      expect(resizer.getAttribute("role")).toBe("separator");
      expect(resizer.getAttribute("aria-orientation")).toBe("horizontal");
      expect(resizer.getAttribute("aria-valuenow")).toBe("400");
      expect(Number(resizer.getAttribute("aria-valuemin"))).toBeGreaterThan(0);
      expect(Number(resizer.getAttribute("aria-valuemax"))).toBeGreaterThan(
        Number(resizer.getAttribute("aria-valuemin")),
      );
    });

    it("grows taller on ArrowUp and reports the new height", async () => {
      await page.viewport(1200, 800);
      const onHeightChange = vi.fn();
      await render(BottomSheetFixture, { height: 400, onHeightChange, open: true });
      const resizer = page.getByTestId("bottom-sheet-resizer");
      await expect.element(resizer).toBeInTheDocument();
      (resizer.element() as HTMLElement).focus();
      await resizer
        .element()
        .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "ArrowUp" }));
      const resizerEl = resizer.element() as HTMLElement;
      expect(resizerEl.getAttribute("aria-valuenow")).toBe("416");
      expect(onHeightChange).toHaveBeenCalledWith(416);
    });

    it("shrinks on ArrowDown and snaps to the clamp with Home/End", async () => {
      await page.viewport(1200, 800);
      await render(BottomSheetFixture, { height: 400, open: true });
      const resizer = page.getByTestId("bottom-sheet-resizer");
      const resizerEl = resizer.element() as HTMLElement;
      resizerEl.focus();
      await resizer
        .element()
        .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "ArrowDown" }));
      expect(resizerEl.getAttribute("aria-valuenow")).toBe("384");
      await resizer
        .element()
        .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "End" }));
      expect(resizerEl.getAttribute("aria-valuenow")).toBe(resizerEl.getAttribute("aria-valuemin"));
      await resizer
        .element()
        .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "Home" }));
      expect(resizerEl.getAttribute("aria-valuenow")).toBe(resizerEl.getAttribute("aria-valuemax"));
    });

    it("hides the drag resizer on small viewports", async () => {
      await page.viewport(390, 700);
      await render(BottomSheetFixture, { height: 400, open: true });
      await expect.element(page.getByTestId("bottom-sheet")).toBeVisible();
      const resizer = page.getByTestId("bottom-sheet-resizer").element() as HTMLElement;
      expect(getComputedStyle(resizer).display).toBe("none");
    });
  });

  describe("mode screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.classList.remove("dark");
    });

    for (const mode of ["light", "dark"]) {
      it(`renders the open sheet in ${mode} mode`, async () => {
        await page.viewport(960, 540);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(BottomSheetFixture, {
          open: true,
          title: "dev → prod",
        });
        const screenshotSheet = page.getByRole("dialog", { name: "dev → prod" });
        await vi.waitFor(() =>
          expect(document.activeElement).toBe(
            screenshotSheet.getByTestId("bottom-sheet-close").element(),
          ),
        );
        (screenshotSheet.getByTestId("bottom-sheet-resizer").element() as HTMLElement).focus();
        await expect.element(screenshotSheet).toMatchScreenshot();
      });
    }
  });
});
