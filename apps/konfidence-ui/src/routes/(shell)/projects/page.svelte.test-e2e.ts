import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn } from "../../../../e2e/helpers";

const PROJECTS_API = "**/api/v1/projects";
const NOOP = (): void => undefined;

const useScenario = async (page: Page, scenario: string): Promise<void> => {
  await page
    .context()
    .addCookies([
      { name: "konfidence_mock_scenario", url: "http://127.0.0.1:8091", value: scenario },
    ]);
};

const switchToIdentity = async (page: Page): Promise<void> => {
  await page.locator(".topbar").getByTestId("project-switch").click();
  await page
    .locator('[data-scope="menu"][data-part="content"][data-state="open"]')
    .getByTestId("project-option-identity-service")
    .click();
};

test("shows the chooser and switches between two projects", async ({ page }) => {
  await signIn(page, "/projects");
  await expect(page.getByRole("link", { name: "Payments Platform" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Identity Service" })).toBeVisible();

  await page.getByRole("link", { name: "Payments Platform" }).click();
  await page.locator(".topbar").getByTestId("project-switch").click();
  await page
    .locator('[data-scope="menu"][data-part="content"][data-state="open"]')
    .getByTestId("project-option-identity-service")
    .click();

  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await expect(page.locator(".topbar").getByTestId("project-switch")).toContainText(
    "Identity Service",
  );
});

test("automatically opens the only accessible project", async ({ page }) => {
  await useScenario(page, "developer");
  await signIn(page, "/projects/payments-platform/landscape");

  await expect(page.getByTestId("page-heading")).toHaveText("Landscape");
});

test("shows the zero-project state", async ({ page }) => {
  await page.route(PROJECTS_API, (route) => route.fulfill({ json: { data: [] } }));
  await signIn(page, "/projects");

  await expect(page.getByTestId("no-projects")).toBeVisible();
});

test("shows loading, then opens the only project after discovery", async ({ page }) => {
  let releaseResponse = NOOP;
  const responseHeld = new Promise<void>((resolve) => {
    releaseResponse = resolve;
  });
  await page.route(PROJECTS_API, async (route) => {
    await responseHeld;
    await route.fulfill({ json: { data: [{ id: "delayed", name: "Delayed" }] } });
  });

  try {
    await page.goto("/");
    await page.getByTestId("sign-in").click();
    await expect(page.getByText("Loading projects…")).toBeVisible();
    releaseResponse();
    await expect(page).toHaveURL("/projects/delayed/landscape");
  } finally {
    releaseResponse();
  }
});

test("retries a failed project request", async ({ page }) => {
  let attempts = 0;
  await page.route(PROJECTS_API, (route) => {
    attempts += 1;
    return attempts === 1
      ? route.fulfill({ json: { message: "Unavailable" }, status: 503 })
      : route.fulfill({ json: { data: [] } });
  });
  await page.goto("/");
  await page.getByTestId("sign-in").click();
  await expect(page.getByRole("heading", { name: "Projects could not be loaded" })).toBeVisible();

  await page.getByRole("button", { name: "Retry" }).click();

  await expect(page).toHaveURL("/projects");
  await expect(page.getByTestId("no-projects")).toBeVisible();
});

test("keeps an unavailable deep link and recovers to the chooser", async ({ page }) => {
  const target = "/projects/not-accessible/landscape";
  await signIn(page, target, target);

  await expect(page).toHaveURL(target);
  await expect(
    page.getByRole("heading", { name: "Project not found or unavailable" }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Project not found or unavailable" }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Back to project selection" }).click();
  await expect(page).toHaveURL("/projects");
});

test("preserves embedded mode through selection and unavailable recovery", async ({ page }) => {
  await signIn(page, "/projects?embedded=1", "/?embedded=1");
  await expect(page.getByTestId("embedded-main")).toBeVisible();
  await page.getByRole("link", { name: "Payments Platform" }).click();
  await expect(page).toHaveURL("/projects/payments-platform/landscape?embedded=1");

  await page.goto("/projects/missing/landscape?embedded=1");
  await page.getByRole("link", { name: "Back to project selection" }).click();
  await expect(page).toHaveURL("/projects?embedded=1");
});

test("remembers a validated selection without overriding chooser or deep links", async ({
  page,
}) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();
  await switchToIdentity(page);
  await expect(page).toHaveURL("/projects/identity-service/landscape");

  await page.goto("/");
  await expect(page).toHaveURL("/projects/identity-service/landscape");
  await page.goto("/projects");
  await expect(page.getByRole("heading", { name: "Choose a project" })).toBeVisible();
  await page.goto("/projects/payments-platform/landscape");
  await expect(page.locator(".topbar").getByTestId("project-switch")).toContainText(
    "Payments Platform",
  );
});

test("ignores an inaccessible remembered project", async ({ page }) => {
  await signIn(page, "/projects");
  await page.evaluate(() => {
    sessionStorage.setItem("konfidence:last-project:alex.admin@example.com", "not-accessible");
  });
  await page.goto("/");
  await expect(page).toHaveURL("/projects");
});
