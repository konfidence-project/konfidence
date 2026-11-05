import { page, userEvent } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import TagFixture from "./TagFixture.svelte";
import "../../../styles/test.css";

describe("Tag", () => {
  it("renders metadata as non-interactive text with no icon or empty icon slot", async () => {
    await render(TagFixture, { label: "US-30", mono: true });
    const el = page.getByText("US-30").element();
    expect(el.closest(".tag")?.tagName).toBe("SPAN");
    expect(el.closest(".tag")?.querySelector("ui5-icon")).toBeNull();
    expect(getComputedStyle(el.closest(".tag")!).fontFamily).toContain("Mono");
  });
  for (const size of ["sm", "md"] as const) {
    it(`keeps ${size} height with an optional icon`, async () => {
      await render(TagFixture, { icon: "accept", size });
      const el = page.getByText("Promotion").element().closest(".tag")!;
      expect(el.getBoundingClientRect().height).toBe(size === "sm" ? 20 : 24);
      expect(el.querySelector("ui5-icon")?.getAttribute("aria-hidden")).toBe("true");
    });
  }
  it("uses a native button with Enter and Space activation", async () => {
    const onclick = vi.fn();
    await render(TagFixture, { interactive: true, onclick });
    const button = page.getByRole("button", { name: "Promotion" });
    (button.element() as HTMLButtonElement).focus();
    await userEvent.keyboard("{Enter}");
    await userEvent.keyboard(" ");
    expect(onclick).toHaveBeenCalledTimes(2);
  });
  it("disables native buttons", async () => {
    const onclick = vi.fn();
    await render(TagFixture, { disabled: true, interactive: true, onclick });
    const button = page.getByRole("button", { name: "Promotion" });
    await expect.element(button).toBeDisabled();
    (button.element() as HTMLButtonElement).click();
    expect(onclick).not.toHaveBeenCalled();
  });
  it("disabled links remove navigation and suppress click callbacks", async () => {
    const onclick = vi.fn();
    await render(TagFixture, { disabled: true, href: "#promotions", onclick });
    const anchor = page.getByText("Promotion").element().closest("a")!;
    expect(anchor.hasAttribute("href")).toBe(false);
    expect(anchor.getAttribute("aria-disabled")).toBe("true");
    anchor.click();
    expect(onclick).not.toHaveBeenCalled();
  });
});
