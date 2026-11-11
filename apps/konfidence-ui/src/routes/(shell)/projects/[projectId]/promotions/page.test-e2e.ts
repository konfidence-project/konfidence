import { expect, test } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH, openPromotions } from "./helpers.ts";

test("Latest waiting opens its group, selects details and never toggles the selection off", async ({
  page,
}) => {
  await openPromotions(page);
  const trigger = page.getByRole("button", { name: "Promotion configuration a-flow" });
  const shortcut = page.getByRole("button", { name: "Latest waiting promotion a-waiting" });
  const row = page.getByRole("button", { exact: true, name: "View promotion a-waiting" });
  await shortcut.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await expect(page).toHaveURL(`${PATH}?promotion=a-waiting`);
  await expect(page.getByTestId("side-panel")).toContainText("vector-a-waiting");
  await shortcut.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await row.click();
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await trigger.click();
  await shortcut.click();
  await expect(row).toBeVisible();
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Promotion configuration z-template" }).click();
  await page.getByRole("button", { name: "View promotion z-newest" }).focus();
  await page.keyboard.press("Enter");
  await expect(row).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("button", { name: "View promotion z-newest" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(page.getByTestId("side-panel")).toContainText("vector-z-newest");
});

test("a single-promotion group expands and collapses all content", async ({ page }) => {
  await page.route(API, (route) =>
    route.fulfill({ json: { data: [{ ...configs[0], promotions: [configs[0].promotions[0]] }] } }),
  );
  await signIn(page, PATH, PATH);
  const trigger = page.getByRole("button", { name: "Promotion configuration z-template" });
  const row = page.getByRole("button", { name: "View promotion z-newest" });
  await expect(row).toBeHidden();
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
  await trigger.click();
  await expect(row).toBeVisible();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await trigger.click();
  await expect(row).toBeHidden();
});

test.describe("promotions overview", () => {
  test("uses project-scoped navigation and shows sorted groups, priority and empty configurations", async ({
    page,
  }) => {
    await openPromotions(page);
    await expect(page.getByTestId("page-heading")).toHaveText("Promotions");
    await expect(page.getByTestId("nav-promotions")).toHaveAttribute("href", PATH);
    const groups = page.getByRole("group", { name: /^Promotion configuration/ });
    await expect(groups).toHaveCount(3);
    await expect(groups.nth(0)).toHaveAttribute("aria-label", "Promotion configuration a-flow");
    await expect(groups.nth(1)).toHaveAttribute("aria-label", "Promotion configuration empty-flow");
    await expect(groups.nth(2)).toContainText("VectorTemplate: release-template");
    await expect(
      page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
    ).toBeHidden();
    await expect(
      page.getByRole("button", { exact: true, name: "View promotion a-newest" }),
    ).toBeHidden();
    await expect(groups.nth(0)).toContainText("2 waiting");
    await expect(page.getByText("No promotions yet")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Promotion configuration empty-flow" }),
    ).toHaveCount(0);
    await expect(page.getByRole("button", { name: "View promotion z-newest" })).toBeHidden();
    await expect(groups.nth(0)).toContainText("Latest waiting: vector-a-waiting");
    await expect(groups.nth(0)).toContainText("Development/dev → Production/prod");
    await expect(groups.nth(2).getByRole("button", { name: /Latest/ })).toHaveCount(0);
    await expect(groups.nth(2)).not.toContainText("Latest:");
    await page.getByTestId("nav-landscape").click();
    await page.getByTestId("nav-promotions").click();
    await expect(page.getByTestId("page-heading")).toHaveText("Promotions");
  });

  test("expands histories independently without duplicating rows and never mutates", async ({
    page,
  }) => {
    let mutations = 0;
    page.on("request", (request) => {
      if (
        ["POST", "PATCH", "PUT", "DELETE"].includes(request.method()) &&
        request.url().includes("vectorPromotion")
      ) {
        mutations += 1;
      }
    });
    await openPromotions(page);
    const flow = page.getByRole("button", { name: "Promotion configuration a-flow" });
    const template = page.getByRole("button", { name: "Promotion configuration z-template" });
    await flow.focus();
    await page.keyboard.press("Enter");
    await template.click();
    await expect(flow).toHaveAttribute("aria-expanded", "true");
    await expect(template).toHaveAttribute("aria-expanded", "true");
    await expect(
      page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
    ).toHaveCount(1);
    await expect(page.getByRole("button", { name: "View promotion a-history" })).toBeVisible();
    await expect(page.getByRole("button", { name: /^View promotion a-/ })).toHaveText([
      /vector-a-newest/,
      /vector-a-waiting/,
      /vector-a-older-waiting/,
      /vector-a-history/,
    ]);
    await expect(page.getByRole("button", { exact: true, name: "Approve" }).first()).toBeDisabled();
    await expect(page.getByRole("button", { exact: true, name: "Reject" }).first()).toBeDisabled();
    await expect(page.getByText(/Approval and rejection are unavailable/).first()).toBeVisible();
    expect(mutations).toBe(0);
    await flow.click();
    await expect(
      page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
    ).toBeHidden();
  });

  test("distinguishes project-empty and retryable initial error", async ({ page }) => {
    await page.route(API, (route) => route.fulfill({ json: {}, status: 503 }));
    await signIn(page, PATH, PATH);
    await expect(page.getByText("Failed to load promotions", { exact: true })).toBeVisible();
    await page.route(API, (route) => route.fulfill({ json: { data: [] } }));
    await page.getByRole("button", { name: "Retry" }).click();
    await expect(page.getByText("No promotion configurations", { exact: true })).toBeVisible();
  });
});

test("shows initial loading until project data arrives", async ({ page }) => {
  let release = () => {};
  const ready = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(API, async (route) => {
    await ready;
    await route.fulfill({ json: { data: configs } });
  });
  await signIn(page, PATH, PATH);
  await expect(page.getByText("Loading promotions…", { exact: true })).toBeVisible();
  release();
  await expect(page.getByRole("button", { name: "Promotion configuration a-flow" })).toBeVisible();
});

test("handles missing optional data truthfully and keeps other projects scoped", async ({
  page,
}) => {
  await page.route(API, (route) =>
    route.fulfill({
      json: {
        data: [
          {
            ...configs[0],
            promotions: [
              promotion("unknown", 1, { gates: undefined, sequence: undefined, status: undefined }),
            ],
          },
        ],
      },
    }),
  );
  await signIn(page, PATH, PATH);
  await page.getByRole("button", { name: "Promotion configuration z-template" }).click();
  await expect(page.getByRole("button", { name: "View promotion unknown" })).toContainText(
    "Unknown",
  );
  await expect(page.getByRole("button", { exact: true, name: "Approve" })).toHaveCount(0);
  await page.route("**/v1/projects/identity-service/vectorPromotionConfigs", (route) =>
    route.fulfill({ json: { data: [] } }),
  );
  await page.goto("/projects/identity-service/promotions?embedded=1");
  await expect(page.getByText("No promotion configurations", { exact: true })).toBeVisible();
  await expect(page.getByTestId("nav-promotions")).toBeHidden();
});
