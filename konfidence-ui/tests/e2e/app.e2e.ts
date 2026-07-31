import { type BrowserContext, type Locator, type Page, expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const BASE_URL = "http://127.0.0.1:4174";
const themes = ["konfidence", "konfidence-dark", "sap_horizon"] as const;

const useScenario = (context: BrowserContext, scenario: string) =>
  context.addCookies([
    {
      name: "konfidence_mock_scenario",
      url: BASE_URL,
      value: scenario,
    },
  ]);

const expectSelectedProject = async (page: Page, projectId: string) => {
  await expect(page.getByRole("combobox", { name: "Project" })).toHaveValue(projectId);
  await expect(page.locator(`a[href="/projects/${projectId}/landscape"]`)).toHaveAttribute(
    "aria-current",
    "page",
  );
};

const openMenuWithKeyboard = async ({
  itemName,
  page,
  scope,
  triggerName,
}: {
  itemName: string;
  page: Page;
  scope: Page | Locator;
  triggerName: string;
}) => {
  const trigger = scope.getByRole("button", { name: triggerName });
  await trigger.focus();
  await page.keyboard.press("ArrowDown");
  const item = page.getByRole("menuitem", { name: itemName });
  await expect(item).toHaveAttribute("data-highlighted", "");
  return { item, trigger };
};

const getRouteHtml = async (page: Page, pathname: string) => {
  const response = await page.request.get(pathname);
  expect(response.ok()).toBe(true);
  return response.text();
};

test("authenticates through the mock API and renders a project landscape", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByLabel("Open account menu for Alex Example")).toBeVisible();
  const developmentStage = page.locator('[aria-label="Stage dev-us30"]:visible');
  const productionStage = page.locator('[aria-label="Stage prod-eu30"]:visible');
  await expect(developmentStage).toBeVisible();
  await expect(productionStage).toBeVisible();
  await expect(developmentStage.getByText("Deploying", { exact: true })).toBeVisible();
  await expect(developmentStage.getByText("Tasks", { exact: true })).toBeVisible();
  await expect(productionStage.getByText("Live", { exact: true })).toBeVisible();
});

test("shows a chooser when multiple projects are available", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(page.getByRole("link", { name: /Payments Platform/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Identity Service/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Analytics Pipeline/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Legacy Migration/ })).toBeVisible();
});

test("keeps project navigation synchronized through card and history navigation", async ({
  page,
}) => {
  await page.goto("/projects");
  await page.getByRole("link", { name: /Identity Service/ }).click();
  await expectSelectedProject(page, "identity-service");

  await page.goBack();
  await expect(page).toHaveURL("/projects");
  await expect(page.getByRole("combobox", { name: "Project" })).toHaveValue("");
  await expect(page.getByRole("navigation", { name: "Delivery" })).toHaveCount(0);

  await page.goForward();
  await expectSelectedProject(page, "identity-service");
});

test("keeps responsive navigation named and unfocusable while closed", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");
  await page.getByRole("button", { name: "Open navigation" }).click();
  await expect(page.getByRole("link", { name: "Landscape" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Vector Deployments" })).toBeVisible();

  await page.setViewportSize({ height: 844, width: 390 });
  await page.getByRole("banner").getByRole("button", { name: "Close navigation" }).click();
  await expect(page.locator("#primary-navigation")).toHaveCSS("visibility", "hidden");
  await expect(page.getByRole("link", { name: "Landscape" })).toHaveCount(0);
  await page.getByRole("button", { name: "Open navigation" }).click();
  await expect(page.getByRole("link", { name: "Landscape" })).toBeVisible();
});

test("supports account menu keyboard navigation, dismissal, and focus restoration", async ({
  page,
}) => {
  await page.goto("/projects/payments-platform/landscape");
  const { trigger } = await openMenuWithKeyboard({
    itemName: "Settings",
    page,
    scope: page,
    triggerName: "Open account menu for Alex Example",
  });
  const menu = page.getByRole("menu", { name: "Account menu" });
  const signOut = page.getByRole("menuitem", { name: "Sign Out" });
  await expect(menu).toBeVisible();

  await page.keyboard.press("ArrowDown");
  await expect(signOut).toHaveAttribute("data-highlighted", "");
  await page.keyboard.press("Escape");
  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();
});

test("invokes a stage menu action from the keyboard", async ({ context, page }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: BASE_URL });
  await page.goto("/projects/payments-platform/landscape");
  const stage = page.locator('[aria-label="Stage dev-us30"]:visible');
  const { item: copy, trigger } = await openMenuWithKeyboard({
    itemName: "Copy stage name",
    page,
    scope: stage,
    triggerName: "More actions for dev-us30",
  });
  await page.keyboard.press("Enter");

  await expect(copy).toBeHidden();
  await expect(trigger).toBeFocused();
  await expect
    .poll(() => page.evaluate(() => globalThis.navigator.clipboard.readText()))
    .toBe("dev-us30");
  await expect(stage.getByText("dev-us30 copied to clipboard")).toHaveText(
    "dev-us30 copied to clipboard",
  );
});

test("shows an empty state when no projects are available", async ({ context, page }) => {
  await useScenario(context, "no-projects");
  await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(page.getByRole("heading", { name: "No projects available" })).toBeVisible();
});

test("redirects directly to the only project", async ({ context, page }) => {
  await useScenario(context, "one-project");
  await page.goto("/");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.locator('[aria-label="Stage prod-eu30"]:visible')).toBeVisible();
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

test("renders, sorts, and selects vector deployments from the OpenAPI contract", async ({
  page,
}) => {
  await page.goto("/projects/payments-platform/vector-deployments");
  await expect(page.getByRole("heading", { name: "Vector Deployments" })).toBeVisible();
  await expect(page.getByText("vector-dev-us30-1", { exact: true })).toBeVisible();
  await expect(page.getByText("dev-us30", { exact: true }).first()).toBeVisible();

  await page.getByRole("button", { name: "Version" }).click();
  await expect(page.getByRole("columnheader", { name: /Version/ })).toHaveAttribute(
    "aria-sort",
    "ascending",
  );
  await page.getByRole("button", { name: "vector-dev-us30-1" }).click();
  await expect(page.getByRole("heading", { name: "vector-dev-us30-1" })).toBeVisible();
  await expect(page.getByText("Deployment history coming soon")).toBeVisible();
});

test("keeps vector details within an intermediate desktop viewport", async ({ page }) => {
  await page.setViewportSize({ height: 800, width: 900 });
  await page.goto("/projects/payments-platform/vector-deployments");
  await page.getByRole("button", { name: "vector-dev-us30-1" }).click();

  await expect(page.getByRole("button", { name: "Back to vector deployments" })).toBeVisible();
  const hasPageOverflow = await page.evaluate(
    () =>
      Math.max(
        globalThis.document.body.scrollWidth,
        globalThis.document.documentElement.scrollWidth,
      ) > globalThis.innerWidth,
  );
  expect(hasPageOverflow).toBe(false);
});

test("switches projects and resets to the landscape", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");

  await page.getByRole("combobox", { name: "Project" }).selectOption({ label: "Identity Service" });
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(page.locator('[aria-label="Stage dev-us30"]:visible')).toBeVisible();

  await page.getByLabel("Open account menu for Alex Example").click();
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await Promise.all([
    expect(page).toHaveURL("/projects/identity-service/landscape?settings=profile"),
    expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible(),
    expect(page.locator("#project-select")).toHaveValue("identity-service"),
    expect(page.locator('a[href="/projects/identity-service/landscape"]')).toHaveAttribute(
      "aria-current",
      "page",
    ),
  ]);
});

test("keeps landscape CSR route-scoped while projects and vector routes remain SSR", async ({
  page,
}) => {
  await page.goto("/projects");
  const projectsHtml = await getRouteHtml(page, "/projects");
  const vectorHtml = await getRouteHtml(page, "/projects/payments-platform/vector-deployments");
  const landscapeHtml = await getRouteHtml(page, "/projects/payments-platform/landscape");

  expect(projectsHtml).toContain('data-theme="konfidence"');
  expect(projectsHtml).toContain("Select a project to inspect its delivery landscape.");
  expect(vectorHtml).toContain("Vector Deployments");
  expect(vectorHtml).toContain("vector-dev-us30-1");
  expect(landscapeHtml).not.toContain("Delivery landscape");
  expect(landscapeHtml).not.toContain('aria-label="Stage dev-us30"');
});

test("persists theme selection", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape?settings=appearance");
  await page.getByRole("radio", { name: /Konfidence Dark/ }).check();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence-dark");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence-dark");

  await page.getByRole("button", { name: "Close settings" }).click();
  await expect(page.locator('[aria-label="Stage prod-eu30"]:visible')).toBeVisible();
});

test("switches language and persists it for client and server rendering", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments?settings=appearance");
  await page.getByRole("combobox", { name: "Language" }).selectOption("de");

  await Promise.all([
    expect(page.locator("html")).toHaveAttribute("lang", "de"),
    expect(page.getByRole("dialog", { name: "Einstellungen" })).toBeVisible(),
  ]);
  await page.getByRole("button", { name: "Einstellungen schließen" }).click();
  await expect(page.getByRole("heading", { name: "Vektordeployments" })).toBeVisible();

  await page.reload();
  await Promise.all([
    expect(page.locator("html")).toHaveAttribute("lang", "de"),
    expect(page.getByRole("heading", { name: "Vektordeployments" })).toBeVisible(),
  ]);

  const vectorHtml = await getRouteHtml(page, "/projects/payments-platform/vector-deployments");
  expect(vectorHtml).toContain('<html lang="de"');
  expect(vectorHtml).toContain("Vektordeployments");
});

for (const theme of themes) {
  test(`landscape view has no axe violations in ${theme}`, async ({ context, page }) => {
    await context.addCookies([{ name: "konfidence_theme", url: BASE_URL, value: theme }]);
    await page.goto("/projects/payments-platform/landscape");
    await expect(page.locator('[aria-label="Stage prod-eu30"]:visible')).toBeVisible();
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations).toEqual([]);
  });
}

test("vector deployments table has no axe violations", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");
  await expect(page.getByRole("table", { name: "Vector deployments" })).toBeVisible();

  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
});

test("has no axe violations with Skeleton overlays open", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");
  await page.getByRole("button", { name: "Open account menu for Alex Example" }).click();
  await expect(page.getByRole("menu", { name: "Account menu" })).toBeVisible();
  const menuResults = await new AxeBuilder({ page }).analyze();
  expect(menuResults.violations).toEqual([]);

  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
  const dialogResults = await new AxeBuilder({ page }).analyze();
  expect(dialogResults.violations).toEqual([]);
});

test("clears the API session when signing out", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");
  const accountMenu = page.getByLabel("Open account menu for Alex Example");
  await expect(accountMenu).toBeVisible();
  await page.route("**/api/login?**", (route) => route.abort());

  await accountMenu.click();
  await page.getByRole("menuitem", { name: "Sign Out" }).click();

  await expect
    .poll(async () => {
      const cookies = await page.context().cookies();
      return cookies.some((cookie) => cookie.name === "kden_session");
    })
    .toBe(false);
});
