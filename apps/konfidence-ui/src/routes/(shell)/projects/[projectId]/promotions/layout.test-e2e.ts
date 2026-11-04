import { expect, test } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH } from "./helpers.ts";

for (const mode of ["light", "dark"] as const) {
  for (const width of [390, 1440]) {
    test(`grouped promotion layout ${mode} at ${width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ height: 1000, width });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      const gates = [
        { state: "Satisfied", type: "test.unit.example.com" },
        { state: "Satisfied", type: "test.e2e.example.com" },
        { state: "Satisfied", type: "architecture.example.com" },
        { state: "Satisfied", type: "security.example.com" },
        { state: "Pending", type: "explicit-approval.konfidence.cloud" },
      ];
      const progress = (satisfied: number) =>
        structuredClone(gates).map((gate, index) => {
          gate.state = index < satisfied ? "Satisfied" : "Pending";
          return gate;
        });
      await page.route(API, (route) =>
        route.fulfill({
          json: {
            data: [
              {
                ...configs[1],
                promotions: [
                  { ...promotion("latest", 3, { vector: "v2.14.0" }), gates },
                  {
                    ...promotion("previous", 2, { vector: "v2.13.5" }),
                    gates: progress(2),
                  },
                  {
                    ...promotion("older", 1, { vector: "v2.13.4" }),
                    gates: progress(1),
                  },
                ],
              },
              configs[0],
            ],
          },
        }),
      );
      await signIn(page, PATH, PATH);
      const group = page.getByRole("group", { name: "Promotion configuration a-flow" });
      await expect(group).toBeVisible();
      await expect(page.getByRole("button", { name: "View promotion latest" })).toBeHidden();
      await group.screenshot({ path: testInfo.outputPath("collapsed.png") });
      await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
      await page.mouse.move(0, 0);
      await expect(group.getByRole("button", { name: /^View promotion/ })).toHaveCount(3);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true,
      );
      await group.screenshot({ path: testInfo.outputPath("expanded.png") });
    });
  }
}
