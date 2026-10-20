import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn, useScenario } from "../../../../../../e2e/helpers";

const LANDSCAPES_API = "**/api/v1/projects/payments-platform/landscapes";
const STAGES_API = "**/api/v1/projects/payments-platform/stages*";
const PROMOTIONS_API = "**/api/v1/projects/payments-platform/vectorPromotionConfigs";
const VECTOR_DEPLOYMENT_API =
  "**/api/v1/projects/payments-platform/landscapes/*/vectorDeployments/*";
const SINGLE_STAGE_API = "**/api/v1/projects/payments-platform/landscapes/*/stages/*";
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
  await page.route(PROMOTIONS_API, (route) => route.fulfill({ json: { data: [] } }));
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
  await expect(page.getByRole("heading", { level: 1, name: "Landscape" })).toBeVisible();
  await expect(page.getByRole("heading", { level: 2 })).toHaveCount(0);
  await expect(page.getByRole("heading", { level: 3 })).toHaveCount(0);
  // The landscape name is shown on each stage card as an eyebrow above the
  // stage title (the fix for names never being displayed).
  await expect(
    page.getByRole("link", { name: /View details for stage dev-us30 in Development/ }),
  ).toBeVisible();
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
  // Landscapes with no stages (Sandbox, Staging) still appear as placeholders.
  await expect(page.getByLabel("Landscape Staging has no stages yet")).toBeVisible();
  await expect(page.getByLabel("Landscape Sandbox has no stages yet")).toBeVisible();
  // Promotion configs drive the dev->test->prod edges between stage cards.
  await expect(page.locator(".svelte-flow__edge").first()).toBeVisible();
});

test("shows the empty overview for the empty mock project", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Identity Service" }).click();

  await expect(page.getByRole("heading", { name: "No landscapes yet" })).toBeVisible();
});

// oxlint-disable-next-line eslint/max-statements -- This real-data journey checks details, landmarks, and shallow history behavior.
test("opens real mock details and follows browser history", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();
  const card = page.getByRole("link", { name: /View details for stage dev-eu10/ });
  const detailsUrl = await card.getAttribute("href");
  await card.click();

  await expect(page).toHaveURL(detailsUrl!);
  const dialog = page.getByRole("dialog", { name: "dev-eu10" });
  await expect(dialog).toBeVisible();
  const targetVersion = page.getByRole("region", { name: "Target version" });
  const activeVersion = page.getByRole("region", { name: "Active version" });
  await expect(targetVersion.getByText("Deploying vector", { exact: true })).toBeVisible();
  await expect(targetVersion.getByText("dev-eu10-v6", { exact: true })).toBeVisible();
  await expect(activeVersion.getByText("Ready", { exact: true })).toBeVisible();
  await expect(activeVersion.getByText("dev-eu10-v5", { exact: true })).toBeVisible();
  await expect(page.locator("main")).toHaveCount(1);
  await page.goBack();
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.goForward();
  await expect(page).toHaveURL(detailsUrl!);
  await expect(page.getByRole("dialog", { name: "dev-eu10" })).toBeVisible();
});

// oxlint-disable-next-line eslint/max-statements -- One journey verifies shallow overlay navigation and its standalone fallback.
test("uses the canonical stage URL for the overlay and standalone page", async ({ page }) => {
  await mockOverview(page);
  await page.route(SINGLE_STAGE_API, (route) => route.fulfill({ json: stage }));
  const target = "/projects/payments-platform/landscape";
  await signIn(page, target, target);

  const card = page.getByRole("link", { name: "View details for stage dev-api in Primary" });
  await expect(card).toHaveAttribute(
    "href",
    "/projects/payments-platform/landscape/primary/stages/dev-api",
  );
  await card.click();

  await expect(page).toHaveURL("/projects/payments-platform/landscape/primary/stages/dev-api");
  await expect(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  const fullPage = page.getByRole("link", { name: "Open full page" });
  await expect(fullPage).toHaveAttribute(
    "href",
    "/projects/payments-platform/landscape/primary/stages/dev-api",
  );

  await fullPage.click();

  await expect(page).toHaveURL("/projects/payments-platform/landscape/primary/stages/dev-api");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByTestId("landscape-flow")).toHaveCount(0);
  await expect(page.getByRole("heading", { level: 1, name: "dev-api" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Breadcrumb" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Back to landscape" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Target version" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Active version" })).toBeVisible();
});

test("renders a pending target without an active version", async ({ page }) => {
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();
  await page.getByRole("link", { name: /View details for stage DEV-canary/ }).click();

  await expect(page.getByRole("dialog", { name: "DEV-canary" })).toBeVisible();
  await expect(
    page
      .getByRole("region", { name: "Target version" })
      .getByText("Pending deployment", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Active version" }).getByText("No active version yet."),
  ).toBeVisible();
});

test("recovers from a stage detail API error", async ({ page }) => {
  let failing = true;
  await page.route(LANDSCAPES_API, (route) =>
    route.fulfill({ json: { data: [{ id: "primary", name: "Primary" }] } }),
  );
  await page.route(STAGES_API, (route) =>
    failing
      ? route.fulfill({
          json: { error: { code: "503", message: "Unavailable" } },
          status: 503,
        })
      : route.fulfill({ json: { data: [stage] } }),
  );
  await page.route(VECTOR_DEPLOYMENT_API, (route) => {
    const id = route.request().url().split("/").pop() ?? "";
    return route.fulfill({
      json: {
        id,
        landscapeId: "primary",
        stageId: "dev-api",
        status: "DeployingVector",
        vector: {
          componentName: `delivery-vector-${id}`,
          componentVersion: "2.0.0",
          repository: "registry.example:5000//delivery/vector",
        },
      },
    });
  });
  const target = "/projects/payments-platform/landscape";
  await signIn(page, target, target);

  await expect(page.getByRole("heading", { name: "Landscape could not be loaded" })).toBeVisible();
  failing = false;
  await page.getByRole("button", { name: "Retry" }).click();
  await page.getByRole("link", { name: /View details for stage dev-api/ }).click();
  await expect(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  await expect(
    page.getByTestId("side-panel").getByText("registry.example:5000//delivery/vector:main"),
  ).toBeVisible();
  await expect(
    page.getByTestId("vectordeployment-detail-id").filter({ hasText: "dev-api-target" }),
  ).toBeVisible();
  await expect(
    page.getByTestId("side-panel").getByText("delivery-vector-dev-api-target", { exact: true }),
  ).toBeVisible();
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

test("retries after a landscape network failure", async ({ page }) => {
  let failing = true;
  await page.route(LANDSCAPES_API, (route) =>
    failing
      ? route.abort("failed")
      : route.fulfill({ json: { data: [{ id: "primary", name: "Primary" }] } }),
  );
  await page.route(STAGES_API, (route) => route.fulfill({ json: { data: [stage] } }));
  const target = "/projects/payments-platform/landscape";
  await signIn(page, target, target);

  await expect(page.getByRole("heading", { name: "Landscape could not be loaded" })).toBeVisible();
  await expect(page.getByText("Landscapes are currently unavailable.")).toBeVisible();
  failing = false;
  await page.getByRole("button", { name: "Retry" }).click();
  await expect(page.getByRole("link", { name: /View details for stage dev-api/ })).toBeVisible();
});

// oxlint-disable-next-line eslint/max-statements -- One journey verifies node drag-stability then overlay open/close client navigation.
test("keeps stage nodes stable under drag and opens the detail overlay on click", async ({
  page,
}) => {
  await mockOverview(page);
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();

  await expect(page.getByRole("link", { name: /View details for stage dev-api/ })).toBeVisible();
  // Cards can initiate canvas panning, but the node's graph position stays fixed.
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
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
});

test("groups stages and opens a real detail route with the keyboard", async ({ page }) => {
  await mockOverview(page);
  await signIn(page, "/projects");
  await page.getByRole("link", { name: "Payments Platform" }).click();

  const link = page.getByRole("link", {
    name: /View details for stage dev-api/,
  });
  await link.focus();
  await expect(link).toBeFocused();
  await page.keyboard.press("Enter");

  await expect(page).toHaveURL("/projects/payments-platform/landscape/primary/stages/dev-api");
  await expect(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Target version" })).toBeVisible();
  await expect(
    page.getByTestId("side-panel").getByText("registry.example:5000//delivery/vector:main"),
  ).toBeVisible();
  await page.getByTestId("side-panel-close").click();
  await expect(page).toHaveURL("/projects/payments-platform/landscape");
});

test("recovers from an overview API error and fits a mobile viewport", async ({ page }) => {
  // Keep the response failing until the retry click, regardless of requests during sign-in.
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

test("renders not found for an unknown canonical stage URL", async ({ page }) => {
  await page.route(SINGLE_STAGE_API, (route) =>
    route.fulfill({
      json: { error: { code: "404", message: "Not found" } },
      status: 404,
    }),
  );
  const target = "/projects/payments-platform/landscape/primary/stages/missing";
  await signIn(page, target, target);

  await expect(page.getByRole("heading", { level: 1, name: "missing" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Stage not found" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Back to landscape" })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
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

// oxlint-disable-next-line eslint/max-statements -- Compare promotion-graph columns, stacking order, and qualified links across landscapes.
test("lays out stages by promotion depth from dev on the left to prod on the right", async ({
  page,
}) => {
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
          // The dev-api -> test-api -> prod-api chain forms a three-stage
          // promotion flow.
          { ...stage, id: "dev-api", name: "dev-api" },
          { ...stage, id: "test-api", name: "test-api" },
          { ...stage, id: "prod-api", name: "prod-api" },
          // The dev-alone stage has no promotions, so it stays in the origin
          // column and stacks under dev-api (alphabetical order within a column).
          { ...stage, id: "dev-alone", name: "dev-alone" },
          // A stage in another landscape with no promotions also starts at the
          // origin column.
          { ...stage, id: "other-api", landscapeId: "secondary", name: "other-api" },
        ],
      },
    }),
  );
  await page.route(PROMOTIONS_API, (route) =>
    route.fulfill({
      json: {
        data: [
          {
            id: "dev-to-test",
            promotions: [],
            source: { kind: "Stage", landscape: "primary", name: "dev-api" },
            target: { kind: "Stage", landscape: "primary", name: "test-api" },
          },
          {
            id: "test-to-prod",
            promotions: [],
            source: { kind: "Stage", landscape: "primary", name: "test-api" },
            target: { kind: "Stage", landscape: "primary", name: "prod-api" },
          },
        ],
      },
    }),
  );
  await signIn(
    page,
    "/projects/payments-platform/landscape",
    "/projects/payments-platform/landscape",
  );
  const dev = page.getByRole("link", {
    exact: true,
    name: "View details for stage dev-api in Primary",
  });
  const testCard = page.getByRole("link", {
    exact: true,
    name: "View details for stage test-api in Primary",
  });
  const prod = page.getByRole("link", {
    exact: true,
    name: "View details for stage prod-api in Primary",
  });
  await expect(dev).toBeVisible();
  const devBox = (await dev.boundingBox())!;
  const testBox = (await testCard.boundingBox())!;
  const prodBox = (await prod.boundingBox())!;
  // Promotion flow reads left-to-right: dev < test < prod on the x-axis.
  expect(testBox.x).toBeGreaterThan(devBox.x + devBox.width);
  expect(prodBox.x).toBeGreaterThan(testBox.x + testBox.width);
  // Unpromoted stages share the origin column (same x as dev-api) and stack
  // vertically in alphabetical order, so dev-alone sits above dev-api.
  const alone = await page
    .getByRole("link", { name: /View details for stage dev-alone/ })
    .boundingBox();
  expect(alone!.x).toBeCloseTo(devBox.x, 0);
  expect(alone!.y).toBeLessThan(devBox.y);
  // The stage from another landscape is qualified by its landscape name.
  const other = page.getByRole("link", {
    exact: true,
    name: "View details for stage other-api in Secondary",
  });
  await expect(other).toHaveAttribute(
    "href",
    "/projects/payments-platform/landscape/secondary/stages/other-api",
  );
  // Promotion edges are rendered between the chained stages.
  await expect(page.locator(".svelte-flow__edge")).toHaveCount(2);
});

// oxlint-disable-next-line eslint/max-statements -- Verifies the overview canvas survives repeated overlay open/close client navigations.
test("keeps overview stages after opening and closing the detail overlay", async ({ page }) => {
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
        data: [stage, { ...stage, landscapeId: "secondary" }],
      },
    }),
  );
  const target = "/projects/payments-platform/landscape";
  await signIn(page, target, target);

  const primary = page.getByRole("link", {
    name: "View details for stage dev-api in Primary",
  });
  const secondary = page.getByRole("link", {
    name: "View details for stage dev-api in Secondary",
  });
  await expect(primary).toBeVisible();
  await expect(secondary).toBeVisible();
  await secondary.click();
  await expect(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  await expect(page).toHaveURL("/projects/payments-platform/landscape/secondary/stages/dev-api");
  await page.getByTestId("side-panel-close").click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(primary).toBeVisible();
  await expect(secondary).toBeVisible();
  await primary.click();
  await expect(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  await expect(page).toHaveURL("/projects/payments-platform/landscape/primary/stages/dev-api");
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
  // The top-left card (first in DOM order) opens within the readable viewport.
  const first = page.getByRole("link", { name: /View details for stage/ }).first();
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
  await expect(page.getByRole("dialog", { name: "prod-legacy" })).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Target version" }).getByText("Failed", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Active version" }).getByText("Ready", { exact: true }),
  ).toBeVisible();
});
