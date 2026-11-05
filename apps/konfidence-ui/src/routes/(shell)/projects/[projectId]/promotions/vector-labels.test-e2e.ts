import { expect, test } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH } from "./helpers.ts";

test("shows repository and version rather than the full registry path in the header and row", async ({
  page,
}) => {
  const vector =
    "https://registry.example.com:5000/konfidence//delivery-vector:2026.9.29-201426000Z";
  const label = "delivery-vector:2026.9.29-201426000Z";
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [promotion("long-vector", 1, { vector })] }] },
    }),
  );
  await signIn(page, PATH, PATH);
  const header = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await expect(header.getByText(label, { exact: true })).toBeVisible();
  await expect(header.getByText(label, { exact: true })).toHaveAttribute("title", vector);
  await header.click();
  const row = page.getByRole("button", { name: "View promotion long-vector" });
  await expect(row.getByText(label, { exact: true })).toBeVisible();
  await expect(row.getByText(label, { exact: true })).toHaveAttribute("title", vector);
});

test("keeps unusually long labels to one line with the complete value retained", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ height: 1000, width: 390 });
  const vector = `vector-${"very-long-name-".repeat(30)}`;
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [promotion("long-label", 1, { vector })] }] },
    }),
  );
  await signIn(page, PATH, PATH);
  const header = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await header.click();
  const label = page
    .getByRole("button", { name: "View promotion long-label" })
    .getByTitle(vector, { exact: true });
  await expect(label).toBeVisible();
  const dimensions = await label.evaluate((element) => {
    const style = getComputedStyle(element);
    return {
      clipped: element.scrollWidth > element.clientWidth,
      height: element.getBoundingClientRect().height,
      lineHeight: Number.parseFloat(style.lineHeight),
    };
  });
  expect(dimensions.height).toBeLessThanOrEqual(dimensions.lineHeight + 1);
  expect(dimensions.clipped).toBe(true);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page
    .getByRole("group", { name: "Promotion configuration a-flow" })
    .screenshot({ path: testInfo.outputPath("long-label.png") });
});
