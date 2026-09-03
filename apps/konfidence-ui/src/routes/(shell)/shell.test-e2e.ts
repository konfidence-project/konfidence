import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn } from "../../../e2e/helpers";

/**
 * Extracts the current project id from the URL. Path shape is
 * `/projects/<id>/<destination>`; matching the id keeps the intent explicit
 * and avoids fragile positional splits.
 */
const PROJECT_ID_PATTERN = /^\/projects\/(?<projectId>[^/]+)(?:\/|$)/;
const projectIdFromUrl = (page: Page): string => {
  const match = PROJECT_ID_PATTERN.exec(new URL(page.url()).pathname);
  const projectId = match?.groups?.projectId;
  if (!projectId) {
    throw new Error(`Unexpected URL shape: ${page.url()}`);
  }
  return projectId;
};

const MOBILE_VIEWPORT = { height: 900, width: 480 } as const;

/**
 * Opens the mobile drawer via the topbar hamburger and asserts it stays open.
 * Extracted so the drawer test itself stays under the max-statements cap.
 */
const openMobileDrawer = async (page: Page): Promise<void> => {
  const sidebar = page.locator(".app-shell__sidebar");
  await expect(sidebar).toHaveAttribute("data-open", "false");
  await page.getByTestId("drawer-toggle").click();
  await expect(sidebar).toHaveAttribute("data-open", "true");
};

/**
 * Asserts that the project switcher lives inside the sidebar drawer and is
 * absent from the topbar. Only true below the `md` breakpoint (@media
 * max-width: 767px), when the switcher slot moves into the drawer.
 */
const expectProjectSwitcherInDrawer = async (page: Page): Promise<void> => {
  await expect(page.locator(".topbar").getByTestId("project-switch")).not.toBeVisible();
  await expect(page.locator(".sidebar").getByTestId("project-switch")).toBeVisible();
};

test.describe("application shell", () => {
  test("renders branding, primary nav, and the user menu when authenticated", async ({ page }) => {
    await signIn(page);

    await expect(page.getByTestId("brand-home")).toBeVisible();
    await expect(page.locator(".topbar").getByTestId("project-switch")).toBeVisible();
    await expect(page.getByTestId("user-menu-trigger")).toBeVisible();
    await expect(page.locator('nav[aria-label="Primary"]')).toBeVisible();
  });

  test("navigates between destinations via client-side routing", async ({ page }) => {
    await signIn(page);

    const projectId = projectIdFromUrl(page);
    const vectorHref = `/projects/${projectId}/vector-deployments`;

    await page.getByTestId("nav-vector-deployments").click();

    await expect(page).toHaveURL(vectorHref);
    await expect(page.getByTestId("page-heading")).toHaveText("Vector Deployments");
    await expect(page.getByTestId("nav-vector-deployments")).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  test("renders the shell for a direct URL to a destination", async ({ page }) => {
    await signIn(page);
    const projectId = projectIdFromUrl(page);

    await page.goto(`/projects/${projectId}/artifact-deployments`);

    await expect(page.getByTestId("page-heading")).toHaveText("Artifact Deployments");
    await expect(page.getByTestId("brand-home")).toBeVisible();
  });

  test("shows a mobile drawer and closes it after a destination is picked", async ({ page }) => {
    await page.setViewportSize(MOBILE_VIEWPORT);
    await signIn(page);
    const projectId = projectIdFromUrl(page);

    await openMobileDrawer(page);
    await expectProjectSwitcherInDrawer(page);

    await page.getByTestId("nav-vector-deployments").click();

    await expect(page).toHaveURL(`/projects/${projectId}/vector-deployments`);
    await expect(page.locator(".app-shell__sidebar")).toHaveAttribute("data-open", "false");
  });

  test("reaches Sign out through the user menu with the keyboard", async ({ page }) => {
    await signIn(page);

    const trigger = page.getByTestId("user-menu-trigger");
    await trigger.focus();
    await page.keyboard.press("Enter");

    const signOut = page.getByTestId("sign-out");
    await expect(signOut).toBeVisible();
    // Zag/Skeleton menu items use `[data-highlighted]` (roving tabindex) rather
    // than DOM focus. Pressing Enter on the highlighted item selects it.
    await expect(signOut).toHaveAttribute("data-highlighted", "");
    await page.keyboard.press("Enter");

    await expect(page).toHaveURL("/login");
  });
});

test.describe("embedded mode", () => {
  test("hides the shell chrome when ?embedded=1 is present", async ({ page }) => {
    await signIn(page);
    const projectId = projectIdFromUrl(page);

    await page.goto(`/projects/${projectId}/landscape?embedded=1`);

    await expect(page.getByTestId("embedded-main")).toBeVisible();
    await expect(page.getByTestId("brand-home")).toHaveCount(0);
    await expect(page.getByTestId("user-menu-trigger")).toHaveCount(0);
    await expect(page.getByTestId("page-heading")).toHaveText("Landscape");
  });

  test("preserves ?embedded=1 across internal client-side navigation", async ({ page }) => {
    await signIn(page);
    const projectId = projectIdFromUrl(page);

    await page.goto(`/projects/${projectId}/landscape?embedded=1`);
    const vectorHref = `/projects/${projectId}/vector-deployments`;

    // Wait for the SPA to hydrate before dispatching the synthetic click:
    // `page.goto` resolves on the browser `load` event, which can race ahead
    // of SvelteKit mounting its click-interception listener, especially on a
    // cold worker.
    await expect(page.getByTestId("embedded-main")).toBeVisible();

    // Anchor click uses client-side routing; the sticky beforeNavigate hook
    // must re-attach the flag to the target URL.
    await page.evaluate((href) => {
      const anchor = globalThis.document.createElement("a");
      anchor.href = href;
      anchor.dataset.testid = "embedded-nav-probe";
      globalThis.document.body.appendChild(anchor);
      anchor.click();
    }, vectorHref);

    await expect(page).toHaveURL(`${vectorHref}?embedded=1`);
    await expect(page.getByTestId("embedded-main")).toBeVisible();
    await expect(page.getByTestId("brand-home")).toHaveCount(0);
  });
});
