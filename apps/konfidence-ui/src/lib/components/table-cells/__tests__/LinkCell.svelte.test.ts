import { page } from "vitest/browser";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import LinkCellFixture from "./LinkCellFixture.svelte";

const { goto } = vi.hoisted(() => ({ goto: vi.fn() }));

vi.mock("$app/navigation", () => ({ goto }));
vi.mock("$app/state", () => ({ page: { url: new URL("http://localhost/") } }));

describe("<LinkCell>", () => {
  beforeEach(() => {
    goto.mockClear();
  });

  it("uses app navigation without selecting its container for an internal URL", async () => {
    const onselect = vi.fn();
    render(LinkCellFixture, { onselect });

    const link = page.getByRole("button", { name: "View related deployments" });
    await expect.element(link).toHaveTextContent("3");
    await link.click();

    expect(goto).toHaveBeenCalledOnce();
    expect(goto).toHaveBeenCalledWith("/projects");
    expect(onselect).not.toHaveBeenCalled();
  });

  it("uses app navigation for an absolute same-origin URL", async () => {
    const onselect = vi.fn();
    const url = "http://localhost/projects";
    render(LinkCellFixture, { onselect, url });

    await page.getByRole("button", { name: "View related deployments" }).click();

    expect(goto).toHaveBeenCalledWith(url);
    expect(onselect).not.toHaveBeenCalled();
  });

  it("renders a native link without selecting its container for an external URL", async () => {
    const onselect = vi.fn();
    render(LinkCellFixture, { onselect, url: "https://example.com/deployments" });

    const link = page.getByRole("link", { name: "View related deployments" });
    await expect.element(link).toHaveAttribute("href", "https://example.com/deployments");

    document.querySelector("a")?.addEventListener("click", (event) => event.preventDefault(), {
      once: true,
    });
    await link.click();

    expect(goto).not.toHaveBeenCalled();
    expect(onselect).not.toHaveBeenCalled();
  });
});
