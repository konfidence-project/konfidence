import { expect, test } from "@playwright/test";

test("all implemented component examples render independently in the complete gallery", async ({
  page,
}) => {
  test.setTimeout(120_000);
  await page.goto("/");
  await expect(
    page.getByRole("heading", { level: 1, name: "Konfidence design system" }),
  ).toBeVisible();
  const samples = page
    .locator('[data-catalog-status="implemented"]')
    .getByRole("figure")
    .filter({ has: page.getByRole("tablist") });
  expect(await samples.count()).toBeGreaterThan(30);
  // Each visit activates the actual lazy sandbox, not just its placeholder.
  /* eslint-disable no-await-in-loop */
  for (const sample of await samples.all()) {
    await sample.scrollIntoViewIfNeeded();
    await expect(sample.getByText("Preview ready", { exact: true })).toBeVisible();
    await expect(sample.getByRole("alert")).toHaveCount(0);
  }
  /* eslint-enable no-await-in-loop */
});
