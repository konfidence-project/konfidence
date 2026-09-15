import type { Page } from "@playwright/test";

const SIGN_IN_TEST_ID = "sign-in";
const MOCK_ORIGIN = "http://127.0.0.1:8091";
const SCENARIO_COOKIE = "konfidence_mock_scenario";

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

/**
 * Selects a mock-server scenario by writing the scenario cookie for the mock
 * origin. Call before {@link signIn} so the initial identity call already
 * observes the desired scenario.
 */
const useScenario = async (page: Page, scenario: string): Promise<void> => {
  await page.context().addCookies([
    {
      name: SCENARIO_COOKIE,
      url: MOCK_ORIGIN,
      value: scenario,
    },
  ]);
};

export { signIn, useScenario };
