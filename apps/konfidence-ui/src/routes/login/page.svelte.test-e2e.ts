import { expect, test } from "@playwright/test";
import { signIn } from "../../../e2e/helpers";

// The Konfidence primary CTA renders with the design-system's amber-gradient
// fill and glow shadow (see @konfidence/design-system Button). Guarding both
// prevents drift back to Skeleton's plain preset fill or an unstyled button
// when the design-system CSS is regenerated.
const KONFIDENCE_PRIMARY_BG_GRADIENT_START = "rgb(255, 203, 107)"; // #FFCB6B
const KONFIDENCE_PRIMARY_BG_GRADIENT_END = "rgb(245, 158, 11)"; // #F59E0B

test("propagates returnTo through the sign-in flow", async ({ page }) => {
  await page.goto("/some/deep/path");

  await expect(page).toHaveURL("/login?returnTo=%2Fsome%2Fdeep%2Fpath");

  const signInButton = page.getByTestId("sign-in");
  const backgroundImage = await signInButton.evaluate(
    (element) => globalThis.getComputedStyle(element).backgroundImage,
  );
  expect(backgroundImage).toContain(KONFIDENCE_PRIMARY_BG_GRADIENT_START);
  expect(backgroundImage).toContain(KONFIDENCE_PRIMARY_BG_GRADIENT_END);

  await signInButton.click();

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
