import type { Page } from "@playwright/test";

const SIGN_IN_TEST_ID = "sign-in";

/**
 * Signs in and waits for the caller's explicit destination.
 */
const signIn = async (
  page: Page,
  expectedDestination: string | RegExp = "/projects",
  returnTo = "/",
): Promise<void> => {
  await page.goto(returnTo);
  await page.getByTestId(SIGN_IN_TEST_ID).click();
  await page.waitForURL(expectedDestination);
};

export { signIn };
