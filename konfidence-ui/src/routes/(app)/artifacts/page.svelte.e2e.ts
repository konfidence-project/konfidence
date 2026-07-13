import { type Page, expect, test } from "@playwright/test";

const testSearch = async (page: Page): Promise<void> => {
  await page.getByRole("searchbox", { name: "Search artifacts" }).fill("artifact-10000");
  await expect(page.getByText(/1 matches/)).toBeVisible();
  await expect(page.getByRole("cell", { name: "artifact-10000" })).toBeVisible();
  await page.getByRole("searchbox", { name: "Search artifacts" }).fill("");
};

const testColumnControls = async (page: Page): Promise<void> => {
  await page.getByLabel("Filter status").selectOption("Ready");
  await expect(page.getByText(/6,666 matches/)).toBeVisible();
  await page.getByRole("button", { name: /Version/ }).click();
  await expect(page.getByRole("button", { name: /Version/ })).toHaveAttribute(
    "data-sort-order",
    "asc",
  );
};

test("renders artifacts dashboard", async ({ page }) => {
  await page.goto("/artifacts");
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByRole("heading", { name: "Artifacts" })).toBeVisible();
  await expect(page.getByText(/10,000 matches/)).toBeVisible();

  await testSearch(page);
  await testColumnControls(page);
});
