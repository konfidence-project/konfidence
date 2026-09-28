import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../styles/test.css";
import ColorModeSelectFixture from "./ColorModeSelectFixture.svelte";

describe("<ColorModeSelect>", () => {
  it("exposes a single checked radio option and updates controlled selection", async () => {
    render(ColorModeSelectFixture);
    await page.getByRole("button", { name: "Change color mode" }).click();
    await expect
      .element(page.getByRole("menuitemradio", { name: "System" }))
      .toHaveAttribute("aria-checked", "true");
    await expect
      .element(page.getByRole("menuitemradio", { name: "Light" }))
      .toHaveAttribute("aria-checked", "false");
    await page.getByRole("menuitemradio", { name: "Dark" }).click();
    await expect.element(page.getByTestId("selected-mode")).toHaveTextContent("dark");
    await page.getByRole("button", { name: "Change color mode" }).click();
    await expect
      .element(page.getByRole("menuitemradio", { name: "Dark" }))
      .toHaveAttribute("aria-checked", "true");
    await expect
      .element(page.getByRole("menuitemradio", { name: "System" }))
      .toHaveAttribute("aria-checked", "false");
  });
});
