import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import SkipLinkTestContainer from "./__tests__/SkipLinkTestContainer.svelte";

describe("<SkipLink>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders an anchor pointing at the target id", async () => {
    render(SkipLinkTestContainer, {
      props: { label: "Skip to main content", target: "app-shell-main" },
    });
    const link = page.getByRole("link", { name: "Skip to main content" });
    await expect.element(link).toHaveAttribute("href", "#app-shell-main");
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders focused in ${mode} mode`, async () => {
      await page.viewport(320, 120);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(SkipLinkTestContainer, {
        props: { label: "Skip to main content", target: "app-shell-main" },
      });
      const anchor = document.querySelector<HTMLAnchorElement>("a.skip-link");
      anchor?.focus();
      await expect
        .element(page.getByRole("link", { name: "Skip to main content" }))
        .toMatchScreenshot();
    });
  }
});
