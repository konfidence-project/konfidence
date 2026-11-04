import { expect, test } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH } from "./helpers.ts";

test("wraps long identifiers and references in narrow dark-mode overview and details", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
  const id = `promotion-${"long-identifier-".repeat(10)}`;
  const long = promotion(id, 1, {
    source: { kind: "VectorTemplate", name: "template-with-a-long-source-reference" },
    vector: `registry.example.com/${"release-vector-".repeat(15)}`,
  });
  await page.route(API, (route) =>
    route.fulfill({
      json: {
        data: [
          {
            ...configs[0],
            id: "configuration-with-a-long-production-release-reference",
            promotions: [long],
            source: long.source,
          },
        ],
      },
    }),
  );
  await signIn(page, PATH, PATH);
  const group = page.getByRole("button", {
    name: "Promotion configuration configuration-with-a-long-production-release-reference",
  });
  await expect(group).toHaveAttribute("aria-expanded", "false");
  await page.screenshot({
    fullPage: true,
    path: testInfo.outputPath("promotions-collapsed-mobile-dark.png"),
  });
  await group.click();
  const row = page.getByRole("button", { exact: true, name: `View promotion ${id}` });
  await expect(row).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({
    fullPage: true,
    path: testInfo.outputPath("promotions-overview-mobile-dark.png"),
  });
  await row.click();
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  await expect(panel).toContainText(long.vector);
  expect(await panel.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
  await page.screenshot({
    fullPage: true,
    path: testInfo.outputPath("promotions-details-mobile-dark.png"),
  });
});
