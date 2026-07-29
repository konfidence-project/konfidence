import { type BrowserContext, expect, test } from "@playwright/test";

const useScenario = (context: BrowserContext, scenario: string) =>
  context.addCookies([
    {
      name: "konfidence_mock_scenario",
      url: "http://127.0.0.1:4173",
      value: scenario,
    },
  ]);

test("authenticates through the mock API and renders a project landscape", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByLabel("Open account menu for Alex Example")).toBeVisible();
  await expect(page.getByLabel("Stage dev-us30")).toBeVisible();
  await expect(page.getByLabel("Stage prod-eu30")).toBeVisible();
  await expect(
    page.getByLabel("Stage dev-us30").getByText("Deploying", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Stage dev-us30").getByText("Tasks", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Stage prod-eu30").getByText("Live", { exact: true })).toBeVisible();
});

test("shows a chooser when multiple projects are available", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(page.locator("ui5-li").filter({ hasText: "Payments Platform" })).toBeVisible();
  await expect(page.locator("ui5-li").filter({ hasText: "Identity Service" })).toBeVisible();
  await expect(page.locator("ui5-li").filter({ hasText: "Analytics Pipeline" })).toBeVisible();
  await expect(page.locator("ui5-li").filter({ hasText: "Legacy Migration" })).toBeVisible();
});

test("shows an empty state when no projects are available", async ({ context, page }) => {
  await useScenario(context, "no-projects");
  await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(
    page.locator('ui5-illustrated-message[title-text="No projects available"]'),
  ).toBeVisible();
});

test("redirects directly to the only project", async ({ context, page }) => {
  await useScenario(context, "one-project");
  await page.goto("/");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByLabel("Stage prod-eu30")).toBeVisible();
});

test("returns not found for a project outside the project list", async ({ page }) => {
  await page.goto("/projects/unknown/landscape");

  await expect(page.getByText("Error 404", { exact: true })).toBeVisible();
});

test("renders access denied for a user without project permissions", async ({ context, page }) => {
  await useScenario(context, "forbidden");
  await page.goto("/projects");

  await expect(page.getByText("Access denied", { exact: true })).toBeVisible();
  await expect(page.getByText("Error 403", { exact: true })).toBeVisible();
});

test("renders an upstream failure without hiding the error status", async ({ context, page }) => {
  await useScenario(context, "api-error");
  await page.goto("/projects");

  await expect(page.getByText("Application unavailable", { exact: true })).toBeVisible();
  await expect(page.getByText("Error 503", { exact: true })).toBeVisible();
});

test("renders vector and artifact deployments from the OpenAPI contract", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");
  await expect(page.getByText("Vector Deployments", { exact: true })).toBeVisible();
  await expect(page.getByText("vector-dev-us30-1", { exact: true })).toBeVisible();
  await expect(page.getByText("dev-us30", { exact: true }).first()).toBeVisible();

  await page.goto("/projects/payments-platform/artifact-deployments");
  await expect(page.getByText("Artifact Deployments", { exact: true })).toBeVisible();
  await expect(page.getByText("artifact-dev-us30-1-1", { exact: true })).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: "artifact-dev-us30-1-1" })).toContainText(
    "ArtifactDeployed",
  );
});

test("switches projects and resets to the landscape", async ({ page }) => {
  await page.goto("/projects/payments-platform/artifact-deployments");

  await page.getByLabel("Project").click();
  await page.getByRole("option", { name: "Identity Service" }).click();
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(page.getByLabel("Stage dev-us30")).toBeVisible();

  await page.getByLabel("Open account menu for Alex Example").click();
  await page.locator('ui5-user-menu-item[data-id="settings"]').click();
  await Promise.all([
    expect(page).toHaveURL("/projects/identity-service/landscape?settings=profile"),
    expect(page.locator("ui5-user-settings-dialog")).toHaveAttribute("open", ""),
    expect(page.getByLabel("Project")).toHaveText("Identity Service"),
    expect(
      page.locator('ui5-side-navigation-item[href="/projects/identity-service/landscape"]'),
    ).toBeVisible(),
  ]);
});

test("clears the API session when signing out", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");
  const accountMenu = page.getByLabel("Open account menu for Alex Example");
  await expect(accountMenu).toBeVisible();
  await page.route("**/api/login?**", (route) => route.abort());

  await accountMenu.click();
  await page.getByRole("button", { name: "Sign Out" }).click();

  await expect
    .poll(async () => {
      const cookies = await page.context().cookies();
      return cookies.some((cookie) => cookie.name === "kden_session");
    })
    .toBe(false);
});
