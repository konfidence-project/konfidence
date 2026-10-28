import { page, userEvent } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import SidePanelFixture from "./SidePanelFixture.svelte";
import "../../../styles/test.css";

describe("<SidePanel>", () => {
  it("stays closed until asked to open", async () => {
    await render(SidePanelFixture);
    const panel = page.getByTestId("side-panel");
    await expect.element(panel).not.toBeVisible();
  });

  it("opens on demand and exposes an accessible title", async () => {
    await render(SidePanelFixture, { open: true, title: "Artifact detail" });
    const panel = page.getByRole("dialog", { name: "Artifact detail" });
    await expect.element(panel).toBeVisible();
    await expect.element(page.getByTestId("panel-body")).toBeVisible();
  });

  it("can open non-modally and keeps the surrounding page interactive", async () => {
    await page.viewport(1200, 600);
    const onClose = vi.fn();
    await render(SidePanelFixture, { modal: false, onClose, open: true });
    await expect.element(page.getByTestId("side-panel")).toHaveAttribute("aria-modal", "false");

    await page.getByTestId("outside-btn").click();
    await expect.element(page.getByTestId("side-panel")).toBeVisible();

    await page
      .getByTestId("outside-btn")
      .element()
      .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "Escape" }));
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("focuses the close button when opened", async () => {
    await render(SidePanelFixture, { modal: false, open: true });
    await vi.waitFor(() => {
      expect(document.activeElement).toBe(page.getByTestId("side-panel-close").element());
    });
  });

  it("converts an open panel between non-modal and modal presentation", async () => {
    await render(SidePanelFixture, { modal: false, open: true });
    const dialog = page.getByTestId("side-panel").element() as HTMLElement;
    expect(dialog.getAttribute("aria-modal")).toBe("false");

    await page.getByTestId("toggle-modal").click();
    await expect.element(dialog).toHaveAttribute("aria-modal", "true");
    await expect.element(page.getByTestId("side-panel")).toBeVisible();
  });

  it("closes only the topmost non-modal panel with Escape", async () => {
    await page.viewport(1200, 600);
    await render(SidePanelFixture, { modal: false, open: true, secondOpen: true });
    await expect.element(page.getByRole("dialog", { name: "Details" })).toBeVisible();
    await expect.element(page.getByRole("dialog", { name: "Second details" })).toBeVisible();

    await vi.waitFor(() =>
      expect(document.activeElement).toBe(
        page
          .getByRole("dialog", { name: "Second details" })
          .getByTestId("side-panel-close")
          .element(),
      ),
    );
    await userEvent.keyboard("{Escape}");

    await expect
      .element(page.getByRole("dialog", { name: "Second details" }))
      .not.toBeInTheDocument();
    await expect.element(page.getByRole("dialog", { name: "Details" })).toBeVisible();
  });

  it("closes when the close button is clicked", async () => {
    await render(SidePanelFixture, { open: true });
    await page.getByTestId("side-panel-close").click();
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
  });

  it("renders the footer snippet when provided", async () => {
    await render(SidePanelFixture, { open: true });
    await expect.element(page.getByTestId("panel-footer")).toBeInTheDocument();
  });

  it("invokes onClose when the user closes the panel", async () => {
    const onClose = vi.fn();
    await render(SidePanelFixture, { onClose, open: true });
    await page.getByTestId("side-panel-close").click();
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
    expect(onClose).toHaveBeenCalled();
  });

  it("reports when the closing transition completes", async () => {
    const onCloseComplete = vi.fn();
    await render(SidePanelFixture, {
      modal: false,
      onCloseComplete,
      open: true,
      title: "Closing panel",
    });
    const dialog = page.getByRole("dialog", { name: "Closing panel" });
    await dialog.getByTestId("side-panel-close").click();
    const element = dialog.element() as HTMLElement;
    await expect.element(element).toHaveAttribute("data-state", "closed");
    element.dispatchEvent(
      new TransitionEvent("transitionend", { bubbles: true, propertyName: "translate" }),
    );
    await vi.waitFor(() => expect(onCloseComplete).toHaveBeenCalledOnce());
  });

  describe("full-page link", () => {
    it("hides the link when no full-page URL is given", async () => {
      await render(SidePanelFixture, { open: true });
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      await expect.element(page.getByTestId("side-panel-full-page")).not.toBeInTheDocument();
    });

    it("renders a normal link to the full page", async () => {
      await render(SidePanelFixture, { fullPageHref: "/details/one", open: true });
      const link = page.getByRole("link", { name: "Open full page" });
      await expect.element(link).toBeVisible();
      await expect.element(link).toHaveAttribute("href", "/details/one");
    });
  });

  describe("header offset geometry", () => {
    it("offsets below the app shell header when the header-height variable is present", async () => {
      await page.viewport(560, 480);
      await render(SidePanelFixture, { headerHeight: "60px", open: true });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      const rect = dialog.getBoundingClientRect();
      expect(rect.top).toBeCloseTo(60, 0);
      expect(rect.height).toBeCloseTo(480 - 60, 0);
    });

    it("falls back to full viewport height when the header-height variable is absent", async () => {
      await page.viewport(560, 480);
      await render(SidePanelFixture, { open: true });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      const rect = dialog.getBoundingClientRect();
      expect(rect.top).toBeCloseTo(0, 0);
      expect(rect.height).toBeCloseTo(480, 0);
    });

    it("insets the backdrop to clear the header when the header-height variable is present", async () => {
      await page.viewport(560, 480);
      await render(SidePanelFixture, { headerHeight: "60px", open: true });
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      const backdrop = getComputedStyle(
        page.getByTestId("side-panel-backdrop").element(),
      ).backgroundImage;
      expect(backdrop).toContain("60px");
      expect(backdrop).toMatch(/rgba\(0, 0, 0, 0\)\s+60px,\s*rgba\(45, 30, 8, 0\.42\)\s+60px/);
    });
  });

  describe("width", () => {
    it("defaults to a wide panel when no width is given", async () => {
      await page.viewport(1200, 600);
      await render(SidePanelFixture, { open: true });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      expect(dialog.getBoundingClientRect().width).toBeCloseTo(544, 0);
    });

    it("honours an explicit width prop", async () => {
      await page.viewport(1200, 600);
      await render(SidePanelFixture, { open: true, width: 700 });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      expect(dialog.getBoundingClientRect().width).toBeCloseTo(700, 0);
    });

    it("clamps the width to the max bound", async () => {
      await page.viewport(1400, 600);
      await render(SidePanelFixture, { open: true, width: 5000 });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      expect(dialog.getBoundingClientRect().width).toBeCloseTo(960, 0);
    });

    it("resizes wider on ArrowLeft and reports the new width", async () => {
      await page.viewport(1200, 600);
      const onWidthChange = vi.fn();
      await render(SidePanelFixture, { onWidthChange, open: true, width: 500 });
      const resizer = page.getByTestId("side-panel-resizer");
      await expect.element(resizer).toBeInTheDocument();
      (resizer.element() as HTMLElement).focus();
      await resizer
        .element()
        .dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "ArrowLeft" }));
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      expect(dialog.getBoundingClientRect().width).toBeCloseTo(516, 0);
      expect(onWidthChange).toHaveBeenCalledWith(516);
    });

    it("exposes resize bounds on the separator for assistive tech", async () => {
      await page.viewport(1200, 600);
      await render(SidePanelFixture, { open: true, width: 600 });
      const resizer = page.getByTestId("side-panel-resizer").element() as HTMLElement;
      expect(resizer.getAttribute("role")).toBe("separator");
      expect(resizer.getAttribute("aria-valuenow")).toBe("600");
      expect(resizer.getAttribute("aria-valuemin")).toBe("360");
      expect(resizer.getAttribute("aria-valuemax")).toBe("960");
    });

    it("fills the viewport width on mobile and hides the resizer", async () => {
      await page.viewport(390, 700);
      await render(SidePanelFixture, { open: true, width: 600 });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      expect(dialog.getBoundingClientRect().width).toBeCloseTo(390, 0);
      const resizer = page.getByTestId("side-panel-resizer").element() as HTMLElement;
      expect(getComputedStyle(resizer).display).toBe("none");
    });
  });

  describe("slide transition", () => {
    it("transitions translate so the panel slides in and out", async () => {
      await page.viewport(1200, 600);
      await render(SidePanelFixture, { open: true });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      await expect.element(page.getByTestId("side-panel")).toBeVisible();
      const style = getComputedStyle(dialog);
      expect(style.transitionProperty).toContain("translate");
      expect(style.transitionProperty).toContain("opacity");
      expect(style.transitionDuration).not.toBe("0s");
    });

    it("parks the closed panel off-screen to the right (slide-out end state)", async () => {
      await page.viewport(1200, 600);
      await render(SidePanelFixture, { open: false });
      const dialog = page.getByTestId("side-panel").element() as HTMLElement;
      const style = getComputedStyle(dialog);
      expect(style.display).toBe("none");
    });
  });

  describe("mode screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.classList.remove("dark");
    });

    for (const mode of ["light", "dark"]) {
      it(`renders the open drawer in ${mode} mode`, async () => {
        await page.viewport(960, 480);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        await render(SidePanelFixture, {
          fullPageHref: "/details/payments-api",
          open: true,
          title: "payments-api@3.4.1",
        });
        const screenshotPanel = page.getByRole("dialog", { name: "payments-api@3.4.1" });
        await vi.waitFor(() =>
          expect(document.activeElement).toBe(
            screenshotPanel.getByTestId("side-panel-close").element(),
          ),
        );
        const screenshotResizer = screenshotPanel
          .getByTestId("side-panel-resizer")
          .element() as HTMLElement;
        screenshotResizer.focus();
        screenshotResizer.style.background = "var(--border-strong, rgba(120, 90, 40, 0.7))";
        await expect.element(screenshotPanel).toMatchScreenshot();
      });
    }
  });
});
