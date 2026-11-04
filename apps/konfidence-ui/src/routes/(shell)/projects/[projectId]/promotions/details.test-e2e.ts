import { expect, test } from "@playwright/test";
import type { Landscape } from "#lib/konfidence-api/types.ts";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH, openPromotions } from "./helpers.ts";

test("qualifies the saved endpoints in both the detail flow and information without changing the current config flow", async ({
  page,
}) => {
  const landscapes: Landscape[] = [
    { id: "development", name: "Development" },
    { id: "production", name: "Production" },
    { id: "production-us", name: "Production" },
    { id: "production-eu", name: "Production" },
  ];
  const history = promotion("qualified-snapshot", 1, {
    source: { kind: "Stage", landscape: "production-us", name: "prod" },
    status: "Succeeded",
    target: { kind: "Stage", landscape: "production-eu", name: "prod" },
  });
  await page.route("**/v1/projects/payments-platform/landscapes", (route) =>
    route.fulfill({ json: { data: landscapes } }),
  );
  await page.route(API, (route) =>
    route.fulfill({ json: { data: [{ ...configs[1], promotions: [history] }] } }),
  );
  const path = `${PATH}?promotion=qualified-snapshot`;
  await signIn(page, path, path);

  const group = page.getByRole("group", { name: "Promotion configuration a-flow" });
  await expect(group).toContainText("Development/dev → Production (production)/prod");
  const panel = page.getByTestId("side-panel");
  const flow = panel.getByRole("region", { name: "Promotion flow" });
  const info = panel.getByRole("region", { name: "Promotion information" });
  await Promise.all(
    [flow, info].flatMap((section) => [
      expect(section.getByText("Production (production-us)/prod", { exact: true })).toBeVisible(),
      expect(section.getByText("Production (production-eu)/prod", { exact: true })).toBeVisible(),
      expect(section).not.toContainText("Development/dev"),
    ]),
  );
});

test("the latest overlapping A-to-B-to-A selection wins without losing URL state or history", async ({
  page,
}) => {
  const path = `${PATH}?promotion=a-waiting&promotionConfig=a-flow&embedded=1&custom=retained`;
  await openPromotions(page, path);
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  const length = await page.evaluate(() => history.length);
  await page.evaluate(async () => {
    // Native activations in one turn exercise requests before goto can commit the URL.
    document.getElementById("promotion-row-a-history")!.click();
    document.getElementById("promotion-row-a-waiting")!.click();
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
    );
  });
  await expect(page).toHaveURL(path);
  await expect(panel).toContainText("vector-a-waiting");
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-history" }),
  ).toHaveAttribute("aria-pressed", "false");
  expect(await page.evaluate(() => history.length)).toBe(length);
});

test("overlapping first selections and filter changes preserve URL state and create only one history entry", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?embedded=1&custom=retained`);
  await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
  const length = await page.evaluate(() => history.length);
  await page.evaluate(async () => {
    document.getElementById("promotion-row-a-history")!.click();
    const config = document.querySelector<HTMLSelectElement>(
      '[aria-label="Filter by promotion configuration"]',
    )!;
    config.value = "a-flow";
    config.dispatchEvent(new Event("change", { bubbles: true }));
    document.getElementById("promotion-row-a-waiting")!.click();
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
    );
  });
  await expect(page).toHaveURL(
    `${PATH}?embedded=1&custom=retained&promotion=a-waiting&promotionConfig=a-flow`,
  );
  await expect(page.getByTestId("side-panel")).toContainText("vector-a-waiting");
  expect(await page.evaluate(() => history.length)).toBe(length + 1);
  await page.getByTestId("side-panel-close").click();
  await expect(page).toHaveURL(`${PATH}?embedded=1&custom=retained&promotionConfig=a-flow`);
});

test("keyboard Latest waiting opens details and explicit close restores the selected row", async ({
  page,
}) => {
  await openPromotions(page);
  const shortcut = page.getByRole("button", { name: "Latest waiting promotion a-waiting" });
  await shortcut.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByTestId("side-panel")).toContainText("vector-a-waiting");
  await expect(page.getByTestId("side-panel-close")).toBeFocused();
  await page.getByTestId("side-panel-close").click();
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toBeFocused();
});

test("closing after a compatible filter change does not revert the filter or embedded mode", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?embedded=1`);
  await page.getByRole("button", { name: "Latest waiting promotion a-waiting" }).click();
  await expect(page.getByTestId("side-panel")).toBeVisible();
  await page
    .getByRole("combobox", { name: "Filter by promotion configuration" })
    .selectOption("a-flow");
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  await page.getByTestId("side-panel-close").click();
  await expect(page).toHaveURL(`${PATH}?embedded=1&promotionConfig=a-flow`);
  await expect(
    page.getByRole("combobox", { name: "Filter by promotion configuration" }),
  ).toHaveValue("a-flow");
});

test("filter cleanup preserves a compatible newer pending selection and search focus", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?promotion=a-waiting&embedded=1`);
  await expect(page.getByTestId("side-panel")).toBeVisible();
  await page.evaluate(async () => {
    document.getElementById("promotion-row-a-history")!.click();
    const search = document.querySelector<HTMLInputElement>('[aria-label="Search promotions"]')!;
    search.value = "a-history";
    search.focus();
    search.dispatchEvent(new Event("input", { bubbles: true }));
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
    );
  });
  await expect(page).toHaveURL(`${PATH}?promotion=a-history&embedded=1`);
  await expect(page.getByTestId("side-panel")).toContainText("vector-a-history");
  await expect(page.getByRole("searchbox", { name: "Search promotions" })).toBeFocused();
});

test("a pending selection commit does not steal focus from an independently focused search field", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?promotion=a-waiting`);
  await expect(page.getByTestId("side-panel")).toBeVisible();
  await page.evaluate(async () => {
    document.getElementById("promotion-row-a-history")!.click();
    document.querySelector<HTMLElement>('[aria-label="Search promotions"]')!.focus();
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
    );
  });
  await expect(page).toHaveURL(`${PATH}?promotion=a-history`);
  await expect(page.getByRole("searchbox", { name: "Search promotions" })).toBeFocused();
});

test("details retain the complete vector reference while the overview uses its compact label", async ({
  page,
}) => {
  const vector = "https://registry.example.com:5000/project/delivery-vector:2026.9.29-201426000Z";
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [promotion("full-reference", 1, { vector })] }] },
    }),
  );
  const path = `${PATH}?promotion=full-reference`;
  await signIn(page, path, path);
  const row = page.getByRole("button", { name: "View promotion full-reference" });
  await expect(
    row.getByText("delivery-vector:2026.9.29-201426000Z", { exact: true }),
  ).toBeVisible();
  await expect(page.getByTestId("side-panel").getByText(vector, { exact: true })).toBeVisible();
});

test("opens one detail context, retains repeated selection and supports close, Back and Forward", async ({
  page,
}) => {
  await openPromotions(page);
  await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
  const row = page.getByRole("button", { exact: true, name: "View promotion a-waiting" });
  const initialLength = await page.evaluate(() => history.length);
  await row.focus();
  await page.keyboard.press("Enter");
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  await expect(page).toHaveURL(`${PATH}?promotion=a-waiting`);
  await expect(panel).toContainText("vector-a-waiting");
  await expect(panel).toContainText("Awaiting approval");
  await expect(panel.getByRole("button", { exact: true, name: "Approve" })).toBeDisabled();
  await expect(panel.getByRole("button", { exact: true, name: "Reject" })).toBeDisabled();
  await expect(page.getByTestId("side-panel-close")).toBeFocused();
  const length = await page.evaluate(() => history.length);
  expect(length).toBe(initialLength + 1);
  await row.click();
  await expect(panel).toBeVisible();
  await expect(row).toHaveAttribute("aria-pressed", "true");
  expect(await page.evaluate(() => history.length)).toBe(length);
  await page.getByTestId("side-panel-close").click();
  await expect(panel).toBeHidden();
  await expect(page).toHaveURL(PATH);
  await expect(row).toHaveAttribute("aria-pressed", "false");
  await expect(row).toBeFocused();
  expect(await page.evaluate(() => history.length)).toBe(length);
  await page.keyboard.press("Space");
  await expect(panel).toBeVisible();
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await page.goBack();
  await expect(panel).toBeHidden();
  await expect(page).toHaveURL(PATH);
  await page.goForward();
  await expect(panel).toBeVisible();
  await expect(page).toHaveURL(`${PATH}?promotion=a-waiting`);
});

test("explicit close removes deep-link selection locally and preserves filters and embedded mode", async ({
  page,
}) => {
  const path = `${PATH}?promotion=a-history&promotionConfig=a-flow&embedded=1`;
  await openPromotions(page, path);
  const row = page.getByRole("button", { name: "View promotion a-history" });
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  const length = await page.evaluate(() => history.length);
  await row.click();
  await expect(panel).toBeVisible();
  await page.getByTestId("side-panel-close").click();
  await expect(panel).toBeHidden();
  await expect(page).not.toHaveURL(/promotion=/);
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  await expect(page).toHaveURL(/embedded=1/);
  expect(await page.evaluate(() => history.length)).toBe(length);
  await row.click();
  await expect(panel).toBeVisible();
  await row.focus();
  await page.keyboard.press("Enter");
  await expect(panel).toBeVisible();
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { name: "Promotion configuration a-flow" }),
  ).toHaveAttribute("aria-expanded", "true");
});

test("reveals historical deep links, shows saved references and approval identity, and preserves collapse", async ({
  page,
}) => {
  const history = promotion("snapshot", 3, {
    gates: [
      {
        setBy: "approver@example.com",
        state: "Satisfied",
        type: "explicit-approval.konfidence.cloud",
      },
    ],
    source: { kind: "VectorTemplate", name: "historical-template" },
    status: "Succeeded",
    target: { kind: "Stage", landscape: "old-landscape", name: "old-prod" },
  });
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [configs[1].promotions[1], history] }] },
    }),
  );
  const path = `${PATH}?promotion=snapshot&embedded=1`;
  await signIn(page, path, path);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  const panel = page.getByTestId("side-panel");
  await expect(group).toHaveAttribute("aria-expanded", "true");
  await expect(panel).toContainText("VectorTemplate: historical-template");
  await expect(panel).toContainText("old-landscape/old-prod");
  await expect(panel.locator("dt", { hasText: /^Source$/ })).toHaveCount(1);
  await expect(panel.locator("dt", { hasText: /^Target$/ })).toHaveCount(1);
  await expect(panel).not.toContainText(/Saved source|Saved target/);
  await expect(panel).toContainText("approver@example.com");
  await expect(panel).toContainText("does not confirm deployment rollout");
  await expect(panel.getByRole("button", { exact: true, name: "Approve" })).toHaveCount(0);
  await group.click();
  await expect(group).toHaveAttribute("aria-expanded", "false");
  await expect(panel).toBeVisible();
  await page.getByTestId("side-panel-close").click();
  await expect(panel).toBeHidden();
  await expect(page).not.toHaveURL(/promotion=/);
  await expect(page).toHaveURL(/embedded=1/);
  await expect(group).toBeFocused();
});

test("keeps compatible filtering attached but closes excluded selection without reverting filters", async ({
  page,
}) => {
  await openPromotions(page);
  await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
  await page.getByRole("button", { exact: true, name: "View promotion a-waiting" }).click();
  await expect(page.getByTestId("side-panel")).toBeVisible();
  await page
    .getByRole("combobox", { name: "Filter by promotion configuration" })
    .selectOption("a-flow");
  await expect(page.getByTestId("side-panel")).toBeVisible();
  await page.getByRole("searchbox", { name: "Search promotions" }).fill("a-history");
  await expect(page.getByTestId("side-panel")).toBeHidden();
  await expect(page).not.toHaveURL(/promotion=/);
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  await expect(page.getByRole("searchbox", { name: "Search promotions" })).toHaveValue("a-history");
  await expect(page.getByRole("searchbox", { name: "Search promotions" })).toBeFocused();
});

test("uses the mobile panel with truthful missing optional approval data and restores row focus", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.route(API, (route) =>
    route.fulfill({
      json: {
        data: [
          {
            ...configs[0],
            promotions: [promotion("unknown", 1, { gates: undefined, status: undefined })],
          },
        ],
      },
    }),
  );
  await signIn(page, PATH, PATH);
  await page.getByRole("button", { name: "Promotion configuration z-template" }).click();
  const row = page.getByRole("button", { name: "View promotion unknown" });
  await row.focus();
  await page.keyboard.press("Space");
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  await expect(panel).toContainText("Unknown");
  await expect(panel).toContainText("No explicit-approval gate recorded");
  await expect(page.getByTestId("side-panel-close")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(panel).toBeHidden();
  await expect(row).toBeFocused();
});

test("even a latest-promotion deep link reveals its row without auto-reopening after collapse", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?promotion=a-waiting`);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await expect(group).toHaveAttribute("aria-expanded", "true");
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toBeVisible();
  await group.click();
  await expect(group).toHaveAttribute("aria-expanded", "false");
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toBeHidden();
  await expect(page.getByTestId("side-panel")).toBeVisible();
});

test("shows actual gate states and metadata in details without fabricated gate names or actions", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ height: 1200, width: 1440 });
  const gates = [
    {
      message: "Source vector activated",
      setBy: "controller",
      state: "Satisfied",
      type: "activated-on-source-stage.konfidence.cloud",
    },
    {
      message: "A vulnerability was found",
      reason: "ScanFailed",
      state: "Error",
      type: "security.example.com",
    },
    { state: "Pending", type: "checks.example.com" },
    { state: "Pending", type: "explicit-approval.konfidence.cloud" },
  ];
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [{ ...promotion("gated", 1), gates }] }] },
    }),
  );
  const path = `${PATH}?promotion=gated`;
  await signIn(page, path, path);
  const panel = page.getByTestId("side-panel");
  const section = panel.getByRole("region", { name: "Gates" });
  await expect(section.getByRole("listitem")).toHaveCount(4);
  await expect(section).toContainText("1 of 4 gates satisfied");
  await expect(section).toContainText("Source stage activation");
  await expect(section).toContainText("security.example.com");
  await expect(section).toContainText("A vulnerability was found");
  await expect(section).toContainText("ScanFailed");
  await expect(section).toContainText("Promotion approval");
  await expect(section).not.toContainText("DECIDED HERE");
  await expect(panel.getByRole("button", { exact: true, name: "Approve" })).toBeDisabled();
  await panel.screenshot({ path: testInfo.outputPath("promotion-gate-details.png") });
});

test("switches selected rows without stacking panel history and closes direct links locally", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?promotion=a-history`);
  const panel = page.getByTestId("side-panel");
  await expect(panel).toBeVisible();
  const length = await page.evaluate(() => history.length);
  await page.getByRole("button", { exact: true, name: "View promotion a-waiting" }).click();
  await expect(page).toHaveURL(`${PATH}?promotion=a-waiting`);
  await expect(panel).toContainText("vector-a-waiting");
  expect(await page.evaluate(() => history.length)).toBe(length);
  await page.getByTestId("side-panel-close").click();
  await expect(panel).toBeHidden();
  await expect(page).toHaveURL(PATH);
});
