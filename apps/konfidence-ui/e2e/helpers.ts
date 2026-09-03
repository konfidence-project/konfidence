import type { Page } from "@playwright/test";

const SIGN_IN_TEST_ID = "sign-in";
const LANDSCAPE_PATH = /\/projects\/[^/]+\/landscape/;

/**
 * Signs in and waits for the dashboard shell to settle. When `returnTo` is
 * omitted the app lands on `/` which redirects to the first project's
 * landscape; the helper waits for that final URL so subsequent assertions
 * are not racing the redirect.
 */
const signIn = async (page: Page, returnTo?: string): Promise<void> => {
  await page.goto(returnTo ?? "/");
  await page.getByTestId(SIGN_IN_TEST_ID).click();
  if (returnTo === undefined) {
    await page.waitForURL(LANDSCAPE_PATH);
    return;
  }
  await page.waitForURL((url) => url.pathname + url.search === returnTo);
};

export { signIn };
