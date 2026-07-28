import AxeBuilder from "@axe-core/playwright";
import { type BrowserContext, type Locator, type Page, expect, test } from "@playwright/test";

const PROJECT_SELECTOR_INSTANCE_COUNT = 2;

const useScenario = (context: BrowserContext, scenario: string) =>
  context.addCookies([
    {
      name: "konfidence_mock_scenario",
      url: "http://127.0.0.1:4173",
      value: scenario,
    },
  ]);

const collectBrowserIssues = (page: Page): string[] => {
  const issues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      issues.push(message.text());
    }
  });
  page.on("pageerror", (error) => issues.push(error.message));
  return issues;
};

const expectNoAxeViolations = async (page: Page, include?: string): Promise<void> => {
  const builder = new AxeBuilder({ page });
  if (include) {
    builder.include(include);
  }
  const results = await builder.analyze();
  expect(results.violations).toEqual([]);
};

const waitForAnimations = (locator: Locator): Promise<void> =>
  locator.evaluate(async (element) => {
    await Promise.allSettled(
      element.getAnimations({ subtree: true }).map((animation) => animation.finished),
    );
  });

test("renders the project landscape on its CSR-only route", async ({ page }) => {
  const browserIssues = collectBrowserIssues(page);
  const response = await page.goto("/projects/payments-platform/landscape");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByLabel("Open account menu for Alex Example")).toBeVisible();
  await expect(page.getByRole("article", { name: "Stage dev-us30" })).toBeVisible();
  await expect(page.getByRole("article", { name: "Stage prod-eu30" })).toBeVisible();
  await expect(
    page.getByRole("article", { name: "Stage dev-us30" }).getByText("Deploying", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("article", { name: "Stage prod-eu30" }).getByText("Live", { exact: true }),
  ).toBeVisible();

  const serverHtml = await response?.text();
  expect(serverHtml).not.toContain("Delivery landscape");
  expect(serverHtml).not.toContain("Stage dev-us30");
  expect(browserIssues).toEqual([]);
});

test("shows an SSR-rendered chooser when multiple projects are available", async ({ page }) => {
  const response = await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(page.getByRole("link", { name: /Payments Platform/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Identity Service/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Analytics Pipeline/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Legacy Migration/ })).toBeVisible();
  expect(await response?.text()).toContain("Projects");
});

test("keeps project selector and navigation synchronized with route history", async ({ page }) => {
  await page.goto("/projects");
  await page.getByRole("link", { name: /Identity Service/ }).click();

  const projectSelector = page.locator("#project-select-desktop");
  const landscapeLink = page.getByRole("link", { name: "Landscape" });
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(projectSelector).toHaveValue("identity-service");
  await expect(landscapeLink).toHaveAttribute("href", "/projects/identity-service/landscape");
  await expect(landscapeLink).toHaveAttribute("aria-current", "page");

  await projectSelector.selectOption("payments-platform");
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(projectSelector).toHaveValue("payments-platform");

  await page.goBack();
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(projectSelector).toHaveValue("identity-service");
  await expect(landscapeLink).toHaveAttribute("href", "/projects/identity-service/landscape");
});

test("shows an empty state when no projects are available", async ({ context, page }) => {
  await useScenario(context, "no-projects");
  await page.goto("/");

  await expect(page).toHaveURL("/projects");
  await expect(page.getByText("No projects available", { exact: true })).toBeVisible();
});

test("redirects directly to the only project", async ({ context, page }) => {
  await useScenario(context, "one-project");
  await page.goto("/");

  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByRole("article", { name: "Stage prod-eu30" })).toBeVisible();
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

test("searches, sorts, and opens vector details", async ({ page }) => {
  const response = await page.goto("/projects/payments-platform/vector-deployments");

  await expect(page.getByRole("heading", { name: "Vector Deployments" })).toBeVisible();
  await expect(page.getByText("vector-dev-us30-1", { exact: true })).toBeVisible();
  const serverHtml = await response?.text();
  expect(serverHtml).toContain("Vector Deployments");
  expect(serverHtml).toContain("vector-dev-us30-1");
  await expect(page.getByText("vector-perf-us30-1", { exact: true })).toBeVisible();

  const idHeader = page.getByRole("columnheader", { name: /Vector deployment/ });
  await idHeader.getByRole("button").click();
  await expect(idHeader).toHaveAttribute("aria-sort", "ascending");

  await page.getByRole("searchbox", { name: "Search vector deployments" }).fill("prod-eu30");
  await expect(page.getByText("vector-prod-eu30-1", { exact: true })).toBeVisible();
  await page.getByText("vector-prod-eu30-1", { exact: true }).click();
  await expect(
    page.getByRole("complementary", { name: /Details for vector-prod-eu30-1/ }),
  ).toBeVisible();
});

test("switches projects, opens URL-controlled settings, and persists theme", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");

  await page.getByLabel("Project", { exact: true }).selectOption({ label: "Identity Service" });
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(page.getByRole("article", { name: "Stage dev-us30" })).toBeVisible();

  await page.getByLabel("Open account menu for Alex Example").click();
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(page).toHaveURL("/projects/identity-service/landscape?settings=profile");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();

  const settingsTabs = page.getByRole("tablist", { name: "Settings sections" });
  await expect(settingsTabs).toHaveAttribute("aria-orientation", "vertical");
  await page.getByRole("tab", { name: "Profile" }).focus();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("tab", { name: "Appearance" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await expect(page).toHaveURL("/projects/identity-service/landscape?settings=appearance");
  await page.getByRole("radio", { name: /Konfidence Dark/ }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence-dark");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence-dark");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
});

test("switches language and persists German across CSR and SSR routes", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments?settings=appearance");

  await page.getByLabel("Language").selectOption("de");
  await expect(page.locator("html")).toHaveAttribute("lang", "de");
  await expect(page.locator("html")).toHaveAttribute("dir", "ltr");
  await expect(page.getByRole("dialog", { name: "Einstellungen" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Vektor-Deployments" })).toBeVisible();

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "de");
  await expect(page.getByRole("dialog", { name: "Einstellungen" })).toBeVisible();

  const response = await page.goto("/projects/payments-platform/vector-deployments");
  expect(await response?.text()).toContain("Vektor-Deployments");
  await expect(page.locator("html")).toHaveAttribute("lang", "de");
  await expect(page.getByLabel("Vektor-Deployments durchsuchen")).toBeVisible();
});

test("stacks mobile settings and uses horizontal tab keyboard navigation", async ({ page }) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.goto("/projects/payments-platform/landscape?settings=profile");

  const dialog = page.getByRole("dialog", { name: "Settings" });
  const tablist = page.getByRole("tablist", { name: "Settings sections" });
  const profileTab = page.getByRole("tab", { name: "Profile" });
  const appearancePanel = page.getByRole("tabpanel", { name: "Appearance" });
  await expect(dialog).toBeVisible();
  await expect(tablist).toHaveAttribute("aria-orientation", "horizontal");

  await profileTab.focus();
  await page.keyboard.press("ArrowRight");
  await expect(page).toHaveURL("/projects/payments-platform/landscape?settings=appearance");
  await expect(appearancePanel).toBeVisible();
  await waitForAnimations(dialog);

  const dialogWidth = await dialog.evaluate((element) => ({
    client: element.clientWidth,
    scroll: element.scrollWidth,
  }));
  expect(dialogWidth.scroll).toBeLessThanOrEqual(dialogWidth.client);
  const [tabsBox, panelBox] = await Promise.all([
    tablist.boundingBox(),
    appearancePanel.boundingBox(),
  ]);
  expect(tabsBox?.y ?? 0).toBeLessThan(panelBox?.y ?? 0);
  await expectNoAxeViolations(page);
});

test("keeps mobile content usable and presents navigation and details as sheets", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.goto("/projects/payments-platform/vector-deployments");

  await page.getByLabel("Toggle navigation").click();
  await expect(page.getByRole("dialog", { name: "Project navigation" })).toBeVisible();
  await expect(page.locator('label[for="project-select-mobile"]')).toBeVisible();
  const selectorIds = await page
    .locator('[id^="project-select-"]')
    .evaluateAll((elements) => elements.map((element) => element.id));
  expect(selectorIds).toHaveLength(PROJECT_SELECTOR_INSTANCE_COUNT);
  expect(new Set(selectorIds).size).toBe(selectorIds.length);
  await waitForAnimations(page.getByRole("dialog", { name: "Project navigation" }));
  await expectNoAxeViolations(page);
  await page.getByRole("link", { name: "Vector Deployments" }).click();
  await expect(page.getByRole("heading", { name: "Vector Deployments" })).toBeVisible();

  await page.getByText("vector-dev-us30-1", { exact: true }).click();
  await expect(page.getByRole("dialog", { name: "vector-dev-us30-1" })).toBeVisible();
  await expect(page.getByText("ghcr.io/konfidence/mock", { exact: true }).last()).toBeVisible();
});

test("opens mobile navigation at the exact 768px breakpoint", async ({ page }) => {
  await page.setViewportSize({ height: 900, width: 768 });
  await page.goto("/projects/payments-platform/landscape");

  await page.getByLabel("Toggle navigation").click();
  await expect(page.getByRole("dialog", { name: "Project navigation" })).toBeVisible();
  await expect(page.locator("#project-select-mobile")).toBeVisible();
});

test("landscape view has no axe violations", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");
  await expect(page.locator('[aria-label="Stage prod-eu30"]:visible')).toBeVisible();

  await expectNoAxeViolations(page);
});

test("vector deployments table has no axe violations", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");
  await expect(page.getByRole("table", { name: "Vector deployments" })).toBeVisible();

  await expectNoAxeViolations(page);
});

test("keeps dropdown and dialog overlays accessible when open", async ({ page }) => {
  await page.goto("/projects/payments-platform/landscape");

  await page.getByLabel("Open account menu for Alex Example").click();
  await expect(page.getByRole("menu")).toBeVisible();
  await waitForAnimations(page.getByRole("menu"));
  await expectNoAxeViolations(page, '[role="menu"]');

  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
  await expect(page.locator("[data-bits-floating-content-wrapper]")).toHaveCount(0);
  await waitForAnimations(page.getByRole("dialog", { name: "Settings" }));
  await expectNoAxeViolations(page);
});

test("clears the API session when signing out", async ({ page }) => {
  await page.goto("/projects/payments-platform/vector-deployments");
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
