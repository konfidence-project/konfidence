import { page } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import PanelResizerFixture from "./PanelResizerFixture.svelte";
import "../../../styles/test.css";

afterEach(() => vi.restoreAllMocks());

describe("PanelResizer left edge", () => {
  it("supports directional keys and Home/End without handling the other axis", async () => {
    await page.viewport(1200, 800);
    const onResize = vi.fn();
    await render(PanelResizerFixture, { onResize });
    const handle = page.getByTestId("resizer").element();
    for (const [key, expected] of [
      ["ArrowLeft", 416],
      ["ArrowRight", 400],
      ["Home", 800],
      ["ArrowLeft", 800],
      ["End", 220],
      ["ArrowRight", 220],
    ] as const) {
      handle.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key }));
      // oxlint-disable-next-line eslint/no-await-in-loop -- each adjustment depends on the preceding size.
      await expect.element(handle).toHaveAttribute("aria-valuenow", String(expected));
      expect(onResize).toHaveBeenLastCalledWith(expected);
    }
    const ignored = new KeyboardEvent("keydown", {
      bubbles: true,
      cancelable: true,
      key: "ArrowUp",
    });
    handle.dispatchEvent(ignored);
    expect(ignored.defaultPrevented).toBe(false);
    expect(onResize).toHaveBeenCalledTimes(6);
  });

  it("clamps and rounds externally supplied sizes without committing them", async () => {
    const onResize = vi.fn();
    await render(PanelResizerFixture, { initialSize: 900.4, onResize });
    await expect.element(page.getByTestId("resizer")).toHaveAttribute("aria-valuenow", "800");
    expect(onResize).not.toHaveBeenCalled();
  });

  for (const endEvent of ["pointerup", "pointercancel", "lostpointercapture"]) {
    it(`updates size during dragging and commits once on ${endEvent}`, async () => {
      await page.viewport(1200, 800);
      const onResize = vi.fn();
      await render(PanelResizerFixture, { onResize });
      const handle = page.getByTestId("resizer").element() as HTMLElement;
      const capture = vi.spyOn(handle, "setPointerCapture").mockImplementation(() => undefined);
      vi.spyOn(handle, "hasPointerCapture").mockReturnValue(true);
      const release = vi.spyOn(handle, "releasePointerCapture").mockImplementation(() => undefined);
      const point = { clientX: 300 };
      const moved = { clientX: 200 };
      handle.dispatchEvent(
        new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 1, ...point }),
      );
      handle.dispatchEvent(
        new PointerEvent("pointermove", { bubbles: true, pointerId: 2, ...moved }),
      );
      await expect.element(page.getByTestId("size")).toHaveTextContent("400");
      handle.dispatchEvent(
        new PointerEvent("pointermove", { bubbles: true, pointerId: 1, ...moved }),
      );
      await expect.element(page.getByTestId("size")).toHaveTextContent("500");
      expect(onResize).not.toHaveBeenCalled();
      handle.dispatchEvent(new PointerEvent(endEvent, { bubbles: true, pointerId: 1 }));
      handle.dispatchEvent(new PointerEvent("lostpointercapture", { bubbles: true, pointerId: 1 }));
      expect(capture).toHaveBeenCalledWith(1);
      expect(release).toHaveBeenCalledWith(1);
      expect(onResize).toHaveBeenCalledExactlyOnceWith(500);
    });
  }

  it("ignores non-primary button drags", async () => {
    const onResize = vi.fn();
    await render(PanelResizerFixture, { onResize });
    const handle = page.getByTestId("resizer").element() as HTMLElement;
    const capture = vi.spyOn(handle, "setPointerCapture");
    handle.dispatchEvent(
      new PointerEvent("pointerdown", { bubbles: true, button: 2, pointerId: 1 }),
    );
    handle.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, pointerId: 1 }));
    expect(capture).not.toHaveBeenCalled();
    expect(onResize).not.toHaveBeenCalled();
  });
});
