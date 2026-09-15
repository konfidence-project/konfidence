import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn, useScenario } from "../../../../../../e2e/helpers";

const LANDSCAPES_API = "**/api/v1/projects/payments-platform/landscapes";
const STAGES_API = "**/api/v1/projects/payments-platform/stages*";
const stage = {
  activeStageVersion: {
    id: "dev-api-old",
    stageGeneration: 1,
    status: "Ready",
    vector: "registry.example:5000//delivery/vector:old",
  },
  id: "dev-api",
  landscapeId: "primary",
  name: "dev-api",
  targetStageVersion: {
    id: "dev-api-target",
    stageGeneration: 2,
    status: "DeployingVector",
    vector: "registry.example:5000//delivery/vector:main",
  },
};

const mockOverview = async (page: Page): Promise<void> => {
  await page.route(LANDSCAPES_API, (route) =>
    route.fulfill({ json: { data: [{ id: "primary", name: "Primary" }] } }),
  );
  await page.route(STAGES_API, (route) => route.fulfill({ json: { data: [stage] } }));
};

// Pointer offsets for the drag-stability check on a stage card.
const DRAG_START_X = 20;
const DRAG_START_Y = 20;
const DRAG_MOVE_X = 160;
const DRAG_MOVE_Y = 120;

// oxlint-disable-next-line eslint/max-statements -- One journey covers every landscape label, category, and status shape on the single canvas.
test("renders the admin mock landscape data on one flow canvas", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();

  await expect(page.getByTestId("landscape-flow")).toBeVisible();
  await expect(page.getByRole("heading", { level: 1, name: "Landscapes" })).toBeVisible();
  await expect(page.getByRole("heading", { level: 2 })).toHaveCount(0);
  await expect(page.getByRole("heading", { level: 3 })).toHaveCount(0);
  await expect(
    page
      .getByRole("link", { name: /View details for stage dev-rollback/ })
      .getByRole("group", { exact: true, name: "Failed" }),
  ).toBeVisible();
  await expect(
    page
      .getByRole("link", { name: /View details for stage dev-us30/ })
      .getByRole("group", { exact: true, name: "Ready" }),
  ).toBeVisible();
  await expect(page.getByRole("link", { name: /View details for stage dev-new/ })).toContainText(
    "No target version yet",
  );
  await expect(page.locator(".svelte-flow__edge")).toHaveCount(0);
});

test("shows the empty overview for the empty mock project", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Identity Service" }).click();

  await expect(page.getByRole("heading", { name: "No landscapes yet" })).toBeVisible();
});

// oxlint-disable-next-line eslint/max-statements -- This real-data journey checks the minimum detail contract, reload, landmark, and return navigation.
test("opens and reloads real mock details without nesting main landmarks", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();
  await page.getByRole("link", { name: /View details for stage dev-eu10/ }).click();

  await expect(page.getByRole("heading", { level: 1, name: "dev-eu10" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Target version" })).toBeVisible();
  await expect(page.getByText("Deploying vector").first()).toBeVisible();
  await expect(page.getByText("dev-eu10-v5", { exact: true })).toBeVisible();
  await expect(page.locator("main")).toHaveCount(1);
  await page.reload();
  await expect(page.locator("main")).toHaveCount(1);
  await page
    .getByRole("navigation", { name: "Breadcrumb" })
    .getByRole("link", { name: "Landscapes" })
    .click();
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
});

test("renders the real degraded mock scenario as retryable", async ({ page }) => {
  await useScenario(page, "degraded");
  await signIn(
    page,
    "/projects/payments-platform/landscape",
    "/projects/payments-platform/landscape",
  );

  await expect(page.getByRole("heading", { name: "Landscape could not be loaded" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Retry" })).toBeVisible();
});

// oxlint-disable-next-line eslint/max-statements -- One journey verifies overview-to-detail keyboard navigation, node stability, reload/back behavior.
test("groups stages and opens a real detail route with the keyboard", async ({ page }) => {
  await mockOverview(page);
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();

  await expect(page.getByRole("link", { name: /View details for stage dev-api/ })).toBeVisible();
  // Stage cards are pinned: dragging on a card neither moves the node nor pans.
  const card = page.locator(".svelte-flow__node-stage").first();
  const transformBefore = await card.evaluate((element) => element.style.transform);
  expect(transformBefore).toContain("translate");
  const box = await card.boundingBox();
  await page.mouse.move(box!.x + DRAG_START_X, box!.y + DRAG_START_Y);
  await page.mouse.down();
  await page.mouse.move(box!.x + DRAG_MOVE_X, box!.y + DRAG_MOVE_Y, {
    steps: 5,
  });
  await page.mouse.up();
  expect(await card.evaluate((element) => element.style.transform)).toBe(transformBefore);

  const link = page.getByRole("link", {
    name: /View details for stage dev-api/,
  });
  await link.focus();
  await page.keyboard.press("Enter");

  await expect(page).toHaveURL("/projects/payments-platform/landscape/primary/stages/dev-api");
  await expect(page.getByRole("heading", { level: 1, name: "dev-api" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Target version" })).toBeVisible();
  await expect(page.getByText("registry.example:5000//delivery/vector:main")).toBeVisible();
  await page.reload();
  await page
    .getByRole("navigation", { name: "Breadcrumb" })
    .getByRole("link", { name: "Landscapes" })
    .click();
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
});

test("recovers from an overview API error and fits a mobile viewport", async ({ page }) => {
  // The +page.ts loader fires an API call at load-time — including
  // during the pre-auth navigation that happens inside `signIn()`. Using
  // a simple attempts counter would burn the 503 on that discarded pre-
  // auth load; drive the mock via an explicit phase flag instead so the
  // 503 lands on the authenticated landscape render, and the retry-click
  // resolves against the success payload.
  let failing = true;
  await page.route(LANDSCAPES_API, (route) =>
    failing
      ? route.fulfill({
          json: { error: { code: "503", message: "Unavailable" } },
          status: 503,
        })
      : route.fulfill({ json: { data: [{ id: "primary", name: "Primary" }] } }),
  );
  await page.route(STAGES_API, (route) => route.fulfill({ json: { data: [stage] } }));
  await page.setViewportSize({ height: 800, width: 375 });
  await signIn(
    page,
    "/projects/payments-platform/landscape",
    "/projects/payments-platform/landscape",
  );

  await expect(page.getByRole("heading", { name: "Landscape could not be loaded" })).toBeVisible();
  failing = false;
  await page.getByRole("button", { name: "Retry" }).click();
  await expect(page.getByRole("link", { name: /View details for stage dev-api/ })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test("shows not found for an unknown stage in a valid landscape", async ({ page }) => {
  await mockOverview(page);
  const target = "/projects/payments-platform/landscape/primary/stages/missing";
  await signIn(page, target, target);

  await expect(page.getByRole("heading", { name: "Stage not found" })).toBeVisible();
  await page.getByRole("link", { name: "Back to landscapes" }).click();
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
});

test("shows not found for an unknown landscape returned as 404 by the mock API", async ({
  page,
}) => {
  const target = "/projects/payments-platform/landscape/nowhere/stages/missing";
  await signIn(page, target, target);

  await expect(page.getByRole("heading", { name: "Stage not found" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Retry" })).toHaveCount(0);
});

const MOBILE_WIDTH = 375;
const DESKTOP_WIDTH = 1280;
const VIEWPORT_HEIGHT = 800;
const MIN_CANVAS_HEIGHT = 600;

for (const embedded of [false, true]) {
  for (const width of [MOBILE_WIDTH, DESKTOP_WIDTH]) {
    // oxlint-disable-next-line eslint/max-statements -- Verify the complete shell-to-canvas sizing contract in each viewport.
    test(`canvas fills the available section at ${width}px, embedded=${embedded}`, async ({
      page,
    }) => {
      await page.setViewportSize({ height: VIEWPORT_HEIGHT, width });
      await mockOverview(page);
      const target = `/projects/payments-platform/landscape${embedded ? "?embedded=1" : ""}`;
      await signIn(page, target, target);
      const canvas = page.getByTestId("landscape-flow");
      await expect(canvas).toBeVisible();
      const bounds = await canvas.boundingBox();
      const main = await page.getByRole("main").boundingBox();
      expect(bounds!.height).toBeGreaterThan(MIN_CANVAS_HEIGHT);
      expect(bounds!.y + bounds!.height).toBeCloseTo(VIEWPORT_HEIGHT, 0);
      expect(bounds!.width).toBeCloseTo(main!.width, 0);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true,
      );
    });
  }
}

// oxlint-disable-next-line eslint/max-statements -- Compare column positions, API order, and qualified links across landscapes.
test("stacks stages across landscapes in top-aligned category columns", async ({ page }) => {
  await page.route(LANDSCAPES_API, (route) =>
    route.fulfill({
      json: {
        data: [
          { id: "primary", name: "Primary" },
          { id: "secondary", name: "Secondary" },
        ],
      },
    }),
  );
  await page.route(STAGES_API, (route) =>
    route.fulfill({
      json: {
        data: [
          stage,
          { ...stage, landscapeId: "secondary" },
          { ...stage, id: "dev-second", name: "dev-second" },
          { ...stage, id: "test-api", name: "test-api" },
        ],
      },
    }),
  );
  await signIn(
    page,
    "/projects/payments-platform/landscape",
    "/projects/payments-platform/landscape",
  );
  const primary = page.getByRole("link", {
    exact: true,
    name: "View details for stage dev-api in Primary, Dev",
  });
  const secondary = page.getByRole("link", {
    exact: true,
    name: "View details for stage dev-api in Secondary, Dev",
  });
  await expect(secondary).toBeVisible();
  const first = await primary.boundingBox();
  const second = await secondary.boundingBox();
  expect(first!.x).toBeCloseTo(second!.x, 0);
  expect(second!.y).toBeGreaterThan(first!.y + first!.height);
  await expect(secondary).toHaveAttribute(
    "href",
    "/projects/payments-platform/landscape/secondary/stages/dev-api",
  );
  const next = await page
    .getByRole("link", { name: /View details for stage dev-second/ })
    .boundingBox();
  expect(next!.y).toBeGreaterThan(second!.y + second!.height);
  const testStage = await page
    .getByRole("link", { name: /View details for stage test-api/ })
    .boundingBox();
  expect(testStage!.y).toBeCloseTo(first!.y, 0);
  expect(testStage!.x).toBeGreaterThan(first!.x + first!.width);
  await expect(page.locator(".svelte-flow__node-landscape")).toHaveCount(0);
});

// oxlint-disable-next-line eslint/max-statements -- Verify initial readability, explicit fit, and keyboard recovery to an offscreen stage.
test("opens at a readable zoom and reveals offscreen stages for keyboard navigation", async ({
  page,
}) => {
  await signIn(
    page,
    "/projects/payments-platform/landscape",
    "/projects/payments-platform/landscape",
  );
  const first = page.getByRole("link", {
    name: /View details for stage dev-us30/,
  });
  await expect(first).toBeInViewport();
  const readableWidth = (await first.boundingBox())!.width;
  expect(readableWidth).toBeGreaterThan(200);
  await page.getByRole("button", { name: "Fit view" }).click();
  await expect.poll(async () => (await first.boundingBox())!.width).toBeLessThan(readableWidth);
  await page.keyboard.press("Tab");
  const last = page.getByRole("link", {
    name: /View details for stage prod-legacy/,
  });
  await last.focus();
  await expect(last).toBeInViewport();
  expect((await last.boundingBox())!.width).toBeGreaterThan(200);
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { level: 1, name: "prod-legacy" })).toBeVisible();
});
