import { page, userEvent } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import PanelShellFixture from "./PanelShellFixture.svelte";
import "../../../styles/test.css";

const waitForCloseFocus = async (): Promise<void> => {
  await vi.waitFor(() =>
    expect(document.activeElement).toBe(page.getByTestId("panel-close").element()),
  );
};

describe("PanelShell with Skeleton Dialog", () => {
  it("restores focus after a modal dismissal and releases scroll locking", async () => {
    await page.viewport(1200, 800);
    const onClose = vi.fn();
    await render(PanelShellFixture, { modal: true, onClose });
    await page.getByTestId("open").click();
    await waitForCloseFocus();
    await vi.waitFor(() => expect(document.body.style.overflow).toBe("hidden"));
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByTestId("panel")).not.toBeVisible();
    await vi.waitFor(() => expect(document.activeElement).toBe(page.getByTestId("open").element()));
    expect(document.body.style.overflow).not.toBe("hidden");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("restarts modality effects without replacing content, losing form state, or closing", async () => {
    await page.viewport(1200, 800);
    const onClose = vi.fn();
    const onCloseComplete = vi.fn();
    await render(PanelShellFixture, { onClose, onCloseComplete, open: true });
    await waitForCloseFocus();
    const panel = page.getByTestId("panel").element();
    await page.getByRole("textbox", { name: "Draft" }).fill("unfinished draft");
    await page.getByTestId("modality").click();
    await expect.element(panel).toHaveAttribute("aria-modal", "true");
    await vi.waitFor(() => expect(document.body.style.overflow).toBe("hidden"));
    await vi.waitFor(() =>
      expect(document.activeElement).toBe(page.getByTestId("modality").element()),
    );
    await page.getByTestId("modality").click();
    await expect.element(panel).toHaveAttribute("aria-modal", "false");
    await vi.waitFor(() => expect(document.body.style.overflow).not.toBe("hidden"));
    await page.getByTestId("outside").click();
    await expect.element(panel).toBeVisible();
    expect(page.getByTestId("panel").element()).toBe(panel);
    await expect
      .element(page.getByRole("textbox", { name: "Draft" }))
      .toHaveValue("unfinished draft");
    expect(onClose).not.toHaveBeenCalled();
    expect(onCloseComplete).not.toHaveBeenCalled();
  });

  it("moves focus on focusKey changes without reopening the surface", async () => {
    await render(PanelShellFixture, { open: true });
    await waitForCloseFocus();
    const panel = page.getByTestId("panel").element();
    await page.getByRole("textbox", { name: "Draft" }).fill("next selection");
    await page.getByTestId("refocus").click();
    await waitForCloseFocus();
    expect(page.getByTestId("panel").element()).toBe(panel);
    await expect.element(panel).toHaveAttribute("data-state", "open");
  });

  it("reports programmatic close completion once, without reporting a user dismissal", async () => {
    const onClose = vi.fn();
    const onCloseComplete = vi.fn();
    await render(PanelShellFixture, { onClose, onCloseComplete, open: true });
    await waitForCloseFocus();
    await page.getByTestId("set-open").click();
    await expect.element(page.getByTestId("panel")).toHaveAttribute("data-state", "closed");
    await vi.waitFor(() => expect(onCloseComplete).toHaveBeenCalledOnce());
    expect(onClose).not.toHaveBeenCalled();
  });

  it("does not complete a stale close after reopening", async () => {
    const onCloseComplete = vi.fn();
    await render(PanelShellFixture, { onCloseComplete, open: true });
    await waitForCloseFocus();
    await page.getByTestId("set-open").click();
    await page.getByTestId("open").click();
    const panel = page.getByTestId("panel").element();
    panel.dispatchEvent(
      new TransitionEvent("transitionend", { bubbles: true, propertyName: "translate" }),
    );
    await expect.element(panel).toHaveAttribute("data-state", "open");
    expect(onCloseComplete).not.toHaveBeenCalled();
  });

  it("keeps the inherited header offset after portalling", async () => {
    await page.viewport(1200, 800);
    await render(PanelShellFixture, { open: true });
    await waitForCloseFocus();
    const panel = page.getByTestId("panel").element();
    expect(panel.getBoundingClientRect().top).toBe(64);
    expect(panel.getBoundingClientRect().height).toBe(736);
  });

  it("traps keyboard focus within modal content", async () => {
    await render(PanelShellFixture, { modal: true, open: true });
    await waitForCloseFocus();
    await userEvent.keyboard("{Shift>}{Tab}{/Shift}");
    await vi.waitFor(() =>
      expect(document.activeElement).toBe(page.getByTestId("refocus").element()),
    );
    await userEvent.keyboard("{Tab}");
    await waitForCloseFocus();
  });

  it("cancels close completion when the owner unmounts", async () => {
    const onCloseComplete = vi.fn();
    await render(PanelShellFixture, { onCloseComplete, open: true });
    await waitForCloseFocus();
    await page.getByTestId("set-open").click();
    await page.getByTestId("unmount").click();
    await expect.element(page.getByTestId("panel")).not.toBeInTheDocument();
    // Wait beyond the close fallback deadline to detect leaked callbacks.
    await new Promise((resolve) => setTimeout(resolve, 400));
    expect(onCloseComplete).not.toHaveBeenCalled();
  });

  it("does not dismiss an independent newer panel when an older panel closes", async () => {
    await page.viewport(1200, 800);
    await render(PanelShellFixture, { open: true, secondOpen: true });
    await vi.waitFor(() =>
      expect(document.activeElement).toBe(page.getByTestId("second-panel-close").element()),
    );
    await page.getByTestId("set-open").click();
    await expect.element(page.getByTestId("panel")).not.toBeVisible();
    await expect.element(page.getByTestId("second-panel")).toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByTestId("second-panel")).not.toBeInTheDocument();
  });
});
