import { expect, test } from "@playwright/test";
import { signIn } from "../../e2e/helpers";

const HTTP_OK = 200;

test("redirects unauthenticated visitors to the login page", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL(/\/login\?returnTo=%2F$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Sign in to Konfidence");
});

test("redirects an authenticated visitor to the first project's landscape", async ({ page }) => {
  await signIn(page);

  await expect(page).toHaveURL(/\/projects\/[^/]+\/landscape$/);
  await expect(page.getByTestId("page-heading")).toHaveText("Landscape");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence");
});

test("serves the SPA document for a deep link", async ({ request }) => {
  const response = await request.get("/projects/example/landscape");

  expect(response.status()).toBe(HTTP_OK);
  expect(await response.text()).toContain('<div style="display: contents">');
});
