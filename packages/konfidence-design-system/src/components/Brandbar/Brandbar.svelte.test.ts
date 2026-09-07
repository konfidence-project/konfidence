import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import Brandbar from "./Brandbar.svelte";

describe("<Brandbar>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  for (const mode of ["light", "dark"]) {
    it(`renders the ${mode} brand bar`, async () => {
      await page.viewport(320, 240);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      const { container } = render(Brandbar);
      const bar = container.querySelector<HTMLElement>("[aria-hidden]");
      await expect.element(bar).toMatchScreenshot();
    });
  }
});
