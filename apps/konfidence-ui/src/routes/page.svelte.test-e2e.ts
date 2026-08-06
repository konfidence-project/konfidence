import { expect, test } from "@playwright/test";

test("renders the dashboard", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Konfidence Dashboard");
});
