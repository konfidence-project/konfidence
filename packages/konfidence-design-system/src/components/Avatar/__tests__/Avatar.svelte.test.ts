import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../styles/test.css";
import AvatarFixture from "./AvatarFixture.svelte";

describe("<Avatar>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  it("renders the initials", async () => {
    render(AvatarFixture, { initials: "RB" });
    const el = document.querySelector<HTMLElement>('[aria-label="Alex Admin"]');
    expect(el?.textContent?.trim()).toBe("RB");
  });

  it("adds the orbit class when orbit=true", async () => {
    render(AvatarFixture, { orbit: true });
    const el = document.querySelector<HTMLElement>(".avatar");
    expect(el?.classList.contains("avatar--orbit")).toBe(true);
  });

  for (const mode of ["light", "dark"] as const) {
    for (const orbit of [false, true]) {
      it(`renders ${mode} ${orbit ? "orbit" : "default"}`, async () => {
        await page.viewport(160, 160);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.classList.toggle("dark", mode === "dark");
        render(AvatarFixture, { initials: "AK", orbit });
        const avatar = document.querySelector<HTMLElement>('[aria-label="Alex Admin"]');
        if (avatar) {
          await expect.element(avatar).toMatchScreenshot();
        }
      });
    }
  }
});
