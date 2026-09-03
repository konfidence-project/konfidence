import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import AvatarGroupHarness from "./AvatarGroupHarness.svelte";

describe("<AvatarGroup>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders child avatars and the more slot", async () => {
    render(AvatarGroupHarness);
    const group = document.querySelector<HTMLElement>(".avatar-group");
    expect(group?.querySelectorAll(".avatar").length).toBe(3);
    expect(group?.querySelector(".avatar-group__more")?.textContent?.trim()).toBe("+5");
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders in ${mode} mode`, async () => {
      await page.viewport(220, 96);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(AvatarGroupHarness);
      const group = document.querySelector<HTMLElement>(".avatar-group");
      if (group) {
        await expect.element(group).toMatchScreenshot();
      }
    });
  }
});
