import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import BadgeFixture from "./BadgeFixture.svelte";
import "../../../styles/test.css";

describe("Badge", () => {
  afterEach(() => {
    document.documentElement.classList.remove("dark");
    delete document.documentElement.dataset.theme;
  });

  for (const theme of ["light", "dark"]) {
    for (const [color, state] of [
      ["red", "error"],
      ["orange", "degraded"],
      ["amber", "warning"],
      ["green", "healthy"],
      ["blue", "deploying"],
      ["violet", "promoting"],
    ] as const) {
      it(`preserves the original filled ${color} palette in ${theme}`, async () => {
        document.documentElement.dataset.theme = "konfidence";
        document.documentElement.classList.toggle("dark", theme === "dark");
        await render(BadgeFixture, { color, showDot: false, status: color });
        const badge = page.getByText("State").element();
        const reference = document.createElement("span");
        reference.style.color = `var(--status-${state}-fg)`;
        reference.style.backgroundColor = `var(--status-${state}-bg)`;
        document.body.append(reference);
        try {
          expect(getComputedStyle(badge).color).toBe(getComputedStyle(reference).color);
          expect(getComputedStyle(badge).backgroundColor).toBe(
            getComputedStyle(reference).backgroundColor,
          );
          expect(getComputedStyle(badge).borderTopColor).toBe("rgba(0, 0, 0, 0)");
          expect(badge.querySelector(".dot")).toBeNull();
        } finally {
          reference.remove();
        }
      });
    }
  }

  for (const variant of ["filled", "subtle"] as const) {
    it(`supports an animated dot in the medium ${variant} badge`, async () => {
      await render(BadgeFixture, { animated: true, size: "md", status: "healthy", variant });
      const badge = page.getByText("State").element();
      const dot = badge.querySelector(".dot")!;
      expect(getComputedStyle(badge).fontSize).toBe("13px");
      expect(badge.getBoundingClientRect().height).toBeCloseTo(31.5, 1);
      expect(getComputedStyle(dot).animationName).not.toBe("none");
      expect(dot.getBoundingClientRect().width).toBe(8);
    });
  }

  it("keeps In Progress static unless animation is requested", async () => {
    await render(BadgeFixture, { status: "active" });
    const dot = page.getByText("State").element().querySelector(".dot")!;
    expect(getComputedStyle(dot).animationName).toBe("none");
  });
  it("can animate a non-active state", async () => {
    await render(BadgeFixture, { animated: true, status: "waiting" });
    const dot = page.getByText("State").element().querySelector(".dot")!;
    expect(getComputedStyle(dot).animationName).not.toBe("none");
    expect(dot.getAttribute("aria-hidden")).toBe("true");
  });
  it("omits the dot even when animation is requested", async () => {
    await render(BadgeFixture, { animated: true, showDot: false });
    expect(page.getByText("State").element().querySelector(".dot")).toBeNull();
  });
  it("offers a subtle colored border without a dot", async () => {
    await render(BadgeFixture, { showDot: false, status: "healthy", variant: "subtle" });
    const el = page.getByText("State").element();
    expect(getComputedStyle(el).borderTopColor).not.toBe("rgba(0, 0, 0, 0)");
    expect(el.querySelector(".dot")).toBeNull();
  });
  for (const status of [
    "waiting",
    "ready",
    "blocked",
    "active",
    "succeeded",
    "failed",
    "rejected",
    "superseded",
  ]) {
    it(`renders a visible tone for ${status}`, async () => {
      await render(BadgeFixture, { status });
      const el = page.getByText("State").element();
      expect(getComputedStyle(el).backgroundColor).not.toBe("rgba(0, 0, 0, 0)");
    });
  }
});
