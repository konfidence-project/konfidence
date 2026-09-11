import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers";

/**
 * End-to-end coverage for the artifact-deployments dashboard destination.
 * Exercises the acceptance criteria against the mock API scenarios.
 */
const PROJECT_ID = "payments-platform";
const ARTIFACT_DEPLOYMENTS_PATH = `/projects/${PROJECT_ID}/artifact-deployments`;
const EXPECTED_ROW_COUNT = 3;

const setScenario = async (page: Page, scenario: string): Promise<void> => {
  await page.context().addCookies([
    {
      domain: "127.0.0.1",
      name: "konfidence_mock_scenario",
      path: "/",
      value: scenario,
    },
  ]);
};

/**
 * Signs in and lands directly on the requested artifact-deployments URL.
 * Uses `signIn`'s `returnTo` argument so we skip the intermediate
 * project-selection page — the mock login flow follows the return URL
 * back to the deep-linked target.
 */
const gotoArtifactDeployments = async (
  page: Page,
  path: string = ARTIFACT_DEPLOYMENTS_PATH,
): Promise<void> => {
  await signIn(page, path, path);
};

test.describe("artifact deployments", () => {
  test("shows every deployment returned by the mock API", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await expect(page.getByTestId("page-heading")).toHaveText("Artifact Deployments");
    await expect(page.getByTestId("artifact-view-count")).toHaveText("3 of 3 deployments");

    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(EXPECTED_ROW_COUNT);
    await expect(page.locator('[data-row-id="artifact-dev-us30-1"]')).toBeVisible();
    await expect(page.locator('[data-row-id="artifact-dev-us30-2"]')).toBeVisible();
    await expect(page.locator('[data-row-id="artifact-test-eu20-1"]')).toBeVisible();
  });

  test("narrows the table by search text", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.getByTestId("artifact-search").fill("payments-ui");

    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(1);
    await expect(page.locator('[data-row-id="artifact-dev-us30-2"]')).toBeVisible();
  });

  test("filters the table by status", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.getByTestId("artifact-status-filter").selectOption("ArtifactFetched");

    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(1);
    await expect(page.locator('[data-row-id="artifact-dev-us30-2"]')).toBeVisible();
  });

  test("sorts by version when the column header is activated", async ({ page }) => {
    await gotoArtifactDeployments(page);

    // Only the payments-api rows share a component name; sorting by
    // version puts the older one first when ascending.
    await page.getByTestId("artifact-search").fill("payments-api");
    await page.getByRole("button", { name: /^Version/ }).click();

    const firstRow = page.getByTestId("artifact-row").first();
    await expect(firstRow).toHaveAttribute("data-row-id", "artifact-test-eu20-1");
  });

  test("opens the detail drawer and shows the resolved relationships", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.locator('[data-row-id="artifact-dev-us30-1"]').click();

    const panel = page.getByTestId("side-panel");
    await expect(panel).toBeVisible();
    await expect(page.getByTestId("artifact-detail-id")).toHaveText("artifact-dev-us30-1");
    await expect(page.getByTestId("artifact-detail-stages")).toContainText("dev-us30");
    await expect(page.getByTestId("artifact-detail-vectors")).toContainText("vector-dev-us30-1");
    await expect(page.getByTestId("artifact-detail-vectors")).toContainText("delivery-vector");
    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?deployment=artifact-dev-us30-1`);
  });

  test("supports keyboard-driven inspection of a row", async ({ page }) => {
    await gotoArtifactDeployments(page);

    const firstRow = page.getByTestId("artifact-row").first();
    await firstRow.focus();
    await page.keyboard.press("Enter");

    const panel = page.getByTestId("side-panel");
    await expect(panel).toBeVisible();
    await expect(page).toHaveURL(/[?&]deployment=/);

    await page.keyboard.press("Escape");
    await expect(panel).toBeHidden();
    await expect(page).not.toHaveURL(/[?&]deployment=/);
  });

  test("deep-links to a deployment via the query parameter", async ({ page }) => {
    await gotoArtifactDeployments(
      page,
      `${ARTIFACT_DEPLOYMENTS_PATH}?deployment=artifact-test-eu20-1`,
    );

    const panel = page.getByTestId("side-panel");
    await expect(panel).toBeVisible();
    await expect(page.getByTestId("artifact-detail-id")).toHaveText("artifact-test-eu20-1");
  });

  test("clears the deployment param when the URL points at a missing id", async ({ page }) => {
    /*
     * `signIn`'s `waitForURL` matches the exact `returnTo` first; the
     * view's own `$effect` then strips the stale `?deployment` value,
     * leaving the bare path. We wait for that final URL rather than
     * asserting mid-flight.
     */
    await signIn(
      page,
      ARTIFACT_DEPLOYMENTS_PATH,
      `${ARTIFACT_DEPLOYMENTS_PATH}?deployment=does-not-exist`,
    );

    await expect(page.getByTestId("side-panel")).toBeHidden();
    await expect(page).toHaveURL(ARTIFACT_DEPLOYMENTS_PATH);
  });

  test("keeps deep-link state addressable through the back button", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.locator('[data-row-id="artifact-dev-us30-1"]').click();
    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?deployment=artifact-dev-us30-1`);

    await page.goBack();
    await expect(page).toHaveURL(ARTIFACT_DEPLOYMENTS_PATH);
    await expect(page.getByTestId("side-panel")).toBeHidden();

    await page.goForward();
    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?deployment=artifact-dev-us30-1`);
    await expect(page.getByTestId("side-panel")).toBeVisible();
  });

  test("closes the drawer when the close button is clicked and clears the URL", async ({
    page,
  }) => {
    await gotoArtifactDeployments(page);

    await page.locator('[data-row-id="artifact-dev-us30-2"]').click();
    await expect(page.getByTestId("side-panel")).toBeVisible();

    await page.getByTestId("side-panel-close").click();
    await expect(page.getByTestId("side-panel")).toBeHidden();
    await expect(page).toHaveURL(ARTIFACT_DEPLOYMENTS_PATH);
  });

  test("shows a no-results message when no rows match the filters", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.getByTestId("artifact-search").fill("does-not-exist");

    await expect(page.getByTestId("artifact-table-no-results")).toBeVisible();
    await expect(page.getByTestId("artifact-row")).toHaveCount(0);

    await page.getByTestId("artifact-view-clear-filters").click();
    await expect(page.getByTestId("artifact-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("shows an empty state when the project has no deployments", async ({ page }) => {
    await setScenario(page, "developer");
    await gotoArtifactDeployments(page);

    await expect(page.getByTestId("artifact-view-empty")).toBeVisible();
    await expect(page.getByTestId("artifact-row")).toHaveCount(0);
  });

  test("filters by landscape via the API and reflects the selection in the URL", async ({
    page,
  }) => {
    await gotoArtifactDeployments(page);

    const requests: string[] = [];
    page.on("request", (request) => {
      if (request.url().includes("/artifactDeployments")) {
        requests.push(request.url());
      }
    });

    await page.getByTestId("artifact-landscape-filter").selectOption("test");

    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?landscapeId=test`);
    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(1);
    await expect(page.locator('[data-row-id="artifact-test-eu20-1"]')).toBeVisible();

    // The `landscapeId=test` query must reach the API (server-side
    // narrowing), not just filter locally.
    expect(requests.some((url) => url.includes("landscapeId=test"))).toBe(true);
  });

  test("deep-links a landscape filter on first load", async ({ page }) => {
    await gotoArtifactDeployments(page, `${ARTIFACT_DEPLOYMENTS_PATH}?landscapeId=development`);

    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(2);
    await expect(page.locator('[data-row-id="artifact-test-eu20-1"]')).toHaveCount(0);
    await expect(page.getByTestId("artifact-landscape-filter")).toHaveValue("development");
  });

  test("filters by vector deployment via the API and clears when landscape changes", async ({
    page,
  }) => {
    await gotoArtifactDeployments(page);

    await page.getByTestId("artifact-vector-filter").selectOption("vector-dev-us30-1");
    await expect(page).toHaveURL(
      `${ARTIFACT_DEPLOYMENTS_PATH}?vectorDeploymentId=vector-dev-us30-1`,
    );

    const rows = page.getByTestId("artifact-row");
    await expect(rows).toHaveCount(2);
    await expect(page.locator('[data-row-id="artifact-dev-us30-1"]')).toBeVisible();
    await expect(page.locator('[data-row-id="artifact-dev-us30-2"]')).toBeVisible();

    // Picking a landscape drops the incompatible vector filter.
    await page.getByTestId("artifact-landscape-filter").selectOption("test");
    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?landscapeId=test`);
    await expect(page.getByTestId("artifact-vector-filter")).toHaveValue("");
  });

  test("clears the filter query params via the clear-filters button", async ({ page }) => {
    await gotoArtifactDeployments(page);

    await page.getByTestId("artifact-landscape-filter").selectOption("development");
    await expect(page).toHaveURL(`${ARTIFACT_DEPLOYMENTS_PATH}?landscapeId=development`);

    await page.getByTestId("artifact-view-clear-filters").click();
    await expect(page).toHaveURL(ARTIFACT_DEPLOYMENTS_PATH);
    await expect(page.getByTestId("artifact-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("surfaces the filter-aware empty state when a filter yields no rows", async ({ page }) => {
    await gotoArtifactDeployments(
      page,
      `${ARTIFACT_DEPLOYMENTS_PATH}?vectorDeploymentId=vector-does-not-exist`,
    );

    await expect(page.getByTestId("artifact-view-empty")).toBeVisible();
    await expect(page.getByTestId("artifact-view-empty-clear")).toBeVisible();

    await page.getByTestId("artifact-view-empty-clear").click();
    await expect(page).toHaveURL(ARTIFACT_DEPLOYMENTS_PATH);
  });

  test("shows an error state and retries when the API fails", async ({ page }) => {
    await setScenario(page, "degraded");
    await gotoArtifactDeployments(page);

    await expect(page.getByTestId("artifact-view-error")).toBeVisible();
    await expect(page.getByTestId("artifact-view-retry")).toBeVisible();
  });
});
