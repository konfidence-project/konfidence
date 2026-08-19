import { expect, test } from "@playwright/test";

const HTTP_OK = 200;

test("renders the dashboard", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Konfidence Dashboard");
});

test("serves the SPA document for a deep link", async ({ request }) => {
  const response = await request.get("/projects/example/landscape");

  expect(response.status()).toBe(HTTP_OK);
  expect(await response.text()).toContain('<div style="display: contents">');
});
