import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import Icon from "./Icon.svelte";
import IconTestContainer from "./__tests__/IconTestContainer.svelte";
import { ICON_NAMES } from "./icons.js";

describe("<Icon>", () => {
  it("renders an svg with the given icon path", async () => {
    render(Icon, { name: "check" });
    const svg = document.querySelector<SVGElement>('svg[data-icon="check"]');
    expect(svg).not.toBeNull();
    expect(svg?.getAttribute("viewBox")).toBe("0 0 16 16");
    expect(svg?.querySelector("path")).not.toBeNull();
  });

  it("hides the svg from assistive technology by default", async () => {
    render(Icon, { name: "check" });
    const svg = document.querySelector<SVGElement>('svg[data-icon="check"]');
    expect(svg?.getAttribute("aria-hidden")).toBe("true");
  });

  it("exposes the svg when ariaHidden is false", async () => {
    render(Icon, { ariaHidden: false, name: "check" });
    const svg = document.querySelector<SVGElement>('svg[data-icon="check"]');
    expect(svg?.getAttribute("aria-hidden")).toBeNull();
  });

  it("applies a custom size", async () => {
    render(Icon, { name: "check", size: 24 });
    const svg = document.querySelector<SVGElement>('svg[data-icon="check"]');
    expect(svg?.getAttribute("width")).toBe("24");
    expect(svg?.getAttribute("height")).toBe("24");
  });

  it("exposes every declared icon name", () => {
    const EXPECTED = 48;
    expect(ICON_NAMES.length).toBe(EXPECTED);
  });

  describe("grid screenshot", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"] as const) {
      it(`renders the ${mode} icon grid`, async () => {
        await page.viewport(320, 320);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(IconTestContainer);
        await expect.element(page.getByTestId("icon-grid")).toMatchScreenshot();
      });
    }
  });
});
