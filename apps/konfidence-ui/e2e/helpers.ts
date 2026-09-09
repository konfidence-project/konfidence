import type { Page } from "@playwright/test";

const SIGN_IN_TEST_ID = "sign-in";

const signIn = async (page: Page, returnTo = "/"): Promise<void> => {
  await page.goto(returnTo);
  await page.getByTestId(SIGN_IN_TEST_ID).click();
  await page.waitForURL((url) => url.pathname + url.search === returnTo);
};

export { signIn };
