import { expect, test } from "@playwright/test";
import { signIn } from "../../../e2e/helpers";

test("propagates returnTo through the sign-in flow", async ({ page }) => {
  await page.goto("/some/deep/path");

  await expect(page).toHaveURL("/login?returnTo=%2Fsome%2Fdeep%2Fpath");

  await page.getByTestId("sign-in").click();

  await expect(page).toHaveURL("/some/deep/path");
});

test("renders the callback error description", async ({ page }) => {
  await page.goto("/login?error=access_denied&error_description=Login%20denied");

  await expect(page.getByRole("alert")).toHaveText("Login denied");
});

test("falls back to the error code when no description is provided", async ({ page }) => {
  await page.goto("/login?error=access_denied");

  await expect(page.getByRole("alert")).toHaveText("access_denied");
});

test("redirects an already authenticated visitor away from /login", async ({ page }) => {
  await signIn(page);

  await page.goto("/login");

  await expect(page).toHaveURL("/");
});
