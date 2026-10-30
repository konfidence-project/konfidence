import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers";

const PROJECT_ID = "payments-platform";
const VECTOR_DEPLOYMENTS_PATH = `/projects/${PROJECT_ID}/vector-deployments`;
const ARTIFACT_DEPLOYMENTS_PATH = `/projects/${PROJECT_ID}/artifact-deployments`;
const EMPTY_VECTOR_DEPLOYMENTS_PATH = "/projects/identity-service/vector-deployments";
const EXPECTED_ROW_COUNT = 3;
const VECTOR_COMPONENT = "github.com/konfidence-project/example-app/vector";
const VECTOR_REPOSITORY = "https://bp.common.repositories.cloud.sap/konfidence/delivery-vector";
const VECTOR_VERSION = "2026.9.29-201426000Z";
const VECTOR_ID = "dev-us30-20260929-201426000Z";

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

const gotoVectorDeployments = async (
  page: Page,
  path: string = VECTOR_DEPLOYMENTS_PATH,
): Promise<void> => {
  await signIn(page, path, path);
};

const expectDetailFieldToFit = async (
  panel: ReturnType<Page["getByTestId"]>,
  label: string,
  value: string,
): Promise<void> => {
  const labelElement = panel.locator("dt", { hasText: label }).first();
  const valueElement = labelElement.locator("xpath=following-sibling::dd[1]");
  await expect(valueElement).toHaveText(value);

  // Measure in one browser call so the opening transition cannot move the
  // panel between the label and value measurements.
  await expect
    .poll(() =>
      labelElement.evaluate((element) => {
        const surface = element.closest('[data-testid="side-panel"]');
        const field = element.nextElementSibling;
        if (!surface || !field) {
          throw new Error("Could not measure the detail field");
        }
        const panelBox = surface.getBoundingClientRect();
        const labelBox = element.getBoundingClientRect();
        const valueBox = field.getBoundingClientRect();
        return {
          followsLabel: valueBox.left >= labelBox.right,
          insidePanel: valueBox.right <= panelBox.right,
        };
      }),
    )
    .toEqual({ followsLabel: true, insidePanel: true });
};

test.describe("vector deployments", () => {
  test("shows every deployment returned by the mock API", async ({ page }) => {
    await gotoVectorDeployments(page);

    await expect(page.getByTestId("page-heading")).toHaveText("Vector Deployments");
    await expect(page.getByTestId("vectordeployment-view-count")).toHaveText("3 of 3 deployments");

    const rows = page.getByTestId("vectordeployment-row");
    await expect(rows).toHaveCount(EXPECTED_ROW_COUNT);
    await expect(page.locator('[data-row-id="dev-us30-20260929-201426000Z"]')).toBeVisible();
    await expect(page.locator('[data-row-id="test-eu20-20260924-151030000Z"]')).toBeVisible();
    await expect(page.locator('[data-row-id="prod-eu30-20260918-094500000Z"]')).toBeVisible();
  });

  test("narrows the table by search text", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.getByTestId("vectordeployment-search").fill("2026.9.24-151030000Z");

    const rows = page.getByTestId("vectordeployment-row");
    await expect(rows).toHaveCount(1);
    await expect(page.locator('[data-row-id="test-eu20-20260924-151030000Z"]')).toBeVisible();
  });

  test("filters the table by status", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.getByTestId("vectordeployment-status-filter").selectOption("DeployingVector");

    const rows = page.getByTestId("vectordeployment-row");
    await expect(rows).toHaveCount(1);
    await expect(page.locator('[data-row-id="prod-eu30-20260918-094500000Z"]')).toBeVisible();
  });

  test("sorts by version when the column header is activated", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.getByRole("button", { name: /^Version/ }).click();

    await expect(page.getByTestId("vectordeployment-row").first()).toHaveAttribute(
      "data-row-id",
      "prod-eu30-20260918-094500000Z",
    );
  });

  test("opens the detail drawer and shows the vector context", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.locator(`[data-row-id="${VECTOR_ID}"]`).click();

    const panel = page.getByTestId("side-panel");
    await expect(panel).toBeVisible();
    await expect(page.getByTestId("vectordeployment-detail-id")).toHaveText(VECTOR_ID);
    await expect(panel).toContainText(VECTOR_COMPONENT);
    await expectDetailFieldToFit(panel, "Component", VECTOR_COMPONENT);
    await expectDetailFieldToFit(panel, "Version", VECTOR_VERSION);
    await expectDetailFieldToFit(panel, "Repository", VECTOR_REPOSITORY);
    await expectDetailFieldToFit(panel, "Deployment id", VECTOR_ID);
    await expect(panel).toContainText("Development");
    await expect(panel).toContainText("dev-us30");
    await expect(page).toHaveURL(`${VECTOR_DEPLOYMENTS_PATH}?deployment=${VECTOR_ID}`);
  });

  test("opens related artifact deployments from the artifact count", async ({ page }) => {
    await gotoVectorDeployments(page);

    const artifactCountButton = page.getByRole("button", {
      name: "View artifact deployments for dev-us30-20260929-201426000Z",
    });
    await expect(artifactCountButton).toHaveText("2");
    await artifactCountButton.click();

    await expect(page).toHaveURL(
      `${ARTIFACT_DEPLOYMENTS_PATH}?vectorDeploymentId=dev-us30-20260929-201426000Z`,
    );
    await expect(page.getByTestId("artifact-row")).toHaveCount(2);
  });

  test("supports keyboard-driven inspection of a row", async ({ page }) => {
    await gotoVectorDeployments(page);

    const firstRow = page.getByTestId("vectordeployment-row").first();
    await firstRow.focus();
    await page.keyboard.press("Enter");

    const panel = page.getByTestId("side-panel");
    await expect(panel).toBeVisible();
    await expect(page).toHaveURL(/[?&]deployment=/);

    await expect(page.getByTestId("side-panel-close")).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(panel).toBeHidden();
    await expect(page).not.toHaveURL(/[?&]deployment=/);
  });

  test("deep-links to a deployment via the query parameter", async ({ page }) => {
    await gotoVectorDeployments(
      page,
      `${VECTOR_DEPLOYMENTS_PATH}?deployment=test-eu20-20260924-151030000Z`,
    );

    await expect(page.getByTestId("side-panel")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-detail-id")).toHaveText(
      "test-eu20-20260924-151030000Z",
    );
  });

  test("keeps the panel closed when the URL points at a missing deployment id", async ({
    page,
  }) => {
    const path = `${VECTOR_DEPLOYMENTS_PATH}?deployment=does-not-exist`;

    await gotoVectorDeployments(page, path);

    await expect(page.getByTestId("side-panel")).toBeHidden();
    await expect(page).toHaveURL(path);
  });

  test("keeps deep-link state addressable through browser history", async ({ page }) => {
    await gotoVectorDeployments(page);

    const row = page.locator('[data-row-id="dev-us30-20260929-201426000Z"]');
    await row.click();
    await expect(page).toHaveURL(
      `${VECTOR_DEPLOYMENTS_PATH}?deployment=dev-us30-20260929-201426000Z`,
    );

    await page.goBack();
    await expect(page).toHaveURL(VECTOR_DEPLOYMENTS_PATH);
    await expect(page.getByTestId("side-panel")).toBeHidden();
    await expect(row).toBeFocused();

    await page.goForward();
    await expect(page).toHaveURL(
      `${VECTOR_DEPLOYMENTS_PATH}?deployment=dev-us30-20260929-201426000Z`,
    );
    await expect(page.getByTestId("side-panel")).toBeVisible();
    await expect(page.getByTestId("side-panel-close")).toBeFocused();
  });

  test("closes the drawer and clears the URL", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.locator('[data-row-id="dev-us30-20260929-201426000Z"]').click();
    await expect(page.getByTestId("side-panel")).toBeVisible();

    await page.getByTestId("side-panel-close").click();
    await expect(page.getByTestId("side-panel")).toBeHidden();
    await expect(page).toHaveURL(VECTOR_DEPLOYMENTS_PATH);
  });

  test("shows a no-results message when no rows match client filters", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.getByTestId("vectordeployment-search").fill("does-not-exist");

    await expect(page.getByTestId("vectordeployment-table-no-results")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(0);

    await page.getByTestId("vectordeployment-view-clear-filters").click();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("shows an empty state when the project has no deployments", async ({ page }) => {
    await gotoVectorDeployments(page, EMPTY_VECTOR_DEPLOYMENTS_PATH);

    await expect(page.getByTestId("vectordeployment-view-empty")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(0);
  });

  test("filters by landscape via the API and reflects the selection in the URL", async ({
    page,
  }) => {
    await gotoVectorDeployments(page);

    const requests: string[] = [];
    page.on("request", (request) => {
      if (request.url().includes("/vectorDeployments")) {
        requests.push(request.url());
      }
    });

    await page.getByTestId("vectordeployment-landscape-filter").selectOption("test");

    await expect(page).toHaveURL(`${VECTOR_DEPLOYMENTS_PATH}?landscapeId=test`);
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(1);
    await expect(page.locator('[data-row-id="test-eu20-20260924-151030000Z"]')).toBeVisible();
    expect(requests.some((url) => url.includes("landscapeId=test"))).toBe(true);
  });

  test("deep-links a landscape filter on first load", async ({ page }) => {
    await gotoVectorDeployments(page, `${VECTOR_DEPLOYMENTS_PATH}?landscapeId=development`);

    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(1);
    await expect(page.locator('[data-row-id="dev-us30-20260929-201426000Z"]')).toBeVisible();
    await expect(page.getByTestId("vectordeployment-landscape-filter")).toHaveValue("development");
  });

  test("clears filter query parameters via the clear-filters button", async ({ page }) => {
    await gotoVectorDeployments(page);

    await page.getByTestId("vectordeployment-landscape-filter").selectOption("development");
    await expect(page).toHaveURL(`${VECTOR_DEPLOYMENTS_PATH}?landscapeId=development`);

    await page.getByTestId("vectordeployment-view-clear-filters").click();
    await expect(page).toHaveURL(VECTOR_DEPLOYMENTS_PATH);
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("surfaces the filter-aware empty state when a landscape has no deployments", async ({
    page,
  }) => {
    await gotoVectorDeployments(page, `${VECTOR_DEPLOYMENTS_PATH}?landscapeId=sandbox`);

    await expect(page.getByTestId("vectordeployment-view-empty")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-view-empty-clear")).toBeVisible();

    await page.getByTestId("vectordeployment-view-empty-clear").click();
    await expect(page).toHaveURL(VECTOR_DEPLOYMENTS_PATH);
  });

  test("retries a rejected deployment request", async ({ page }) => {
    let failNext = true;
    await page.route("**/v1/projects/*/vectorDeployments*", async (route) => {
      if (failNext) {
        failNext = false;
        await route.abort("failed");
      } else {
        await route.continue();
      }
    });
    await gotoVectorDeployments(page);
    await expect(page.getByTestId("vectordeployment-view-error")).toBeVisible();
    await page.getByTestId("vectordeployment-view-retry").click();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("blocks the table when stages fail and recovers on retry", async ({ page }) => {
    let failNext = true;
    await page.route("**/v1/projects/*/stages*", async (route) => {
      if (failNext) {
        failNext = false;
        await route.fulfill({ body: "Unavailable", status: 500 });
      } else {
        await route.continue();
      }
    });
    await gotoVectorDeployments(page);
    await expect(page.getByTestId("vectordeployment-view-error")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(0);
    await page.getByTestId("vectordeployment-view-retry").click();
    await expect(page.getByTestId("vectordeployment-row")).toHaveCount(EXPECTED_ROW_COUNT);
  });

  test("shows an error state and retry action when the API fails", async ({ page }) => {
    await setScenario(page, "degraded");
    await gotoVectorDeployments(page);

    await expect(page.getByTestId("vectordeployment-view-error")).toBeVisible();
    await expect(page.getByTestId("vectordeployment-view-retry")).toBeVisible();
  });
});
