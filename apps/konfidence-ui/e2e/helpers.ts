import type { Page } from "@playwright/test";

const SIGN_IN_TEST_ID = "sign-in";

const signIn = async (page: Page, returnTo?: string): Promise<void> => {
  const target = returnTo ?? "/";
  await page.goto(target);
  await page.getByTestId(SIGN_IN_TEST_ID).click();
  await page.waitForURL((url) => url.pathname + url.search === target);
};

export { signIn };
