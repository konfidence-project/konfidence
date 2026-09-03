import { expect, test } from "@playwright/test";
import { signIn } from "../../../e2e/helpers";

test("logs the user out and redirects to /login", async ({ page }) => {
  await signIn(page);

  await page.getByTestId("user-menu-trigger").click();
  await page.getByTestId("sign-out").click();

  await expect(page).toHaveURL("/login");
  const cookies = await page.context().cookies();
  expect(cookies.some((cookie) => cookie.name === "kden-session")).toBe(false);
});

test("redirects an unauthenticated visit to /logout back to /login", async ({ page }) => {
  await page.goto("/logout");

  await expect(page).toHaveURL("/login");
});

test("redirects to /login when the session cookie is cleared externally", async ({ page }) => {
  await signIn(page);

  await page.context().clearCookies();
  await page.reload();

  await expect(page).toHaveURL(/\/login\?returnTo=%2F/);
});
