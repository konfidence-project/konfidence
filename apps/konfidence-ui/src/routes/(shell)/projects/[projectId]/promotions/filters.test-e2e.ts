import { expect, test } from "@playwright/test";
import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH, openPromotions } from "./helpers.ts";

test("deployment-style landscape links prefilter promotions and survive reload and history navigation", async ({
  page,
}) => {
  await openPromotions(page, `${PATH}?landscapeId=development&embedded=1&other=keep`);
  const landscape = page.getByRole("combobox", { name: "Filter by landscape" });
  const search = page.getByRole("searchbox", { name: "Search promotions" });
  const flow = page.getByRole("button", { name: "Promotion configuration a-flow" });
  const group = page.getByRole("group", { name: "Promotion configuration a-flow" });
  await expect(landscape).toHaveValue("development");
  await expect(page.getByRole("group", { name: /^Promotion configuration/ })).toHaveCount(2);
  await expect(page.getByRole("group", { name: "Promotion configuration empty-flow" })).toHaveCount(
    0,
  );
  await page.reload();
  await expect(landscape).toHaveValue("development");
  await flow.click();
  await search.fill("a-flow");

  await landscape.selectOption("production");
  await expect(landscape).toHaveValue("production");
  await expect(page).toHaveURL(/landscapeId=production/);
  await page.goBack();
  await expect(landscape).toHaveValue("development");
  await expect(flow).toHaveAttribute("aria-expanded", "true");
  await expect(search).toHaveValue("a-flow");
  await page.goForward();
  await expect(landscape).toHaveValue("production");

  await page.getByTestId("promotion-view-clear-filters").click();
  await expect(landscape).toHaveValue("");
  await expect(search).toHaveValue("");
  await expect(group).toBeVisible();
  await expect(flow).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("group", { name: /^Promotion configuration/ })).toHaveCount(3);
  const params = new URL(page.url()).searchParams;
  expect(params.has("landscapeId")).toBe(false);
  expect(params.has("promotionLandscape")).toBe(false);
  expect(params.get("embedded")).toBe("1");
  expect(params.get("other")).toBe("keep");
});

test("page-load filters use the last repeated values and normalize stage JSON without consuming unrelated params", async ({
  page,
}) => {
  const params = new URLSearchParams({ _data: "unrelated", embedded: "1", other: "keep" });
  params.append("landscapeId", "production");
  params.append("landscapeId", "development");
  params.append("promotionStage", JSON.stringify(["production", "prod"]));
  params.append("promotionStage", '["development", "dev"]');
  params.append("promotionConfig", "z-template");
  params.append("promotionConfig", "a-flow");
  await openPromotions(page, `${PATH}?${params}`);
  const landscape = page.getByRole("combobox", { name: "Filter by landscape" });
  const stage = page.getByRole("combobox", { name: "Filter by stage" });
  const config = page.getByRole("combobox", { name: "Filter by promotion configuration" });
  await expect(landscape).toHaveValue("development");
  await expect(stage).toHaveValue(JSON.stringify(["development", "dev"]));
  await expect(config).toHaveValue("a-flow");
  await expect(page.getByRole("group", { name: /^Promotion configuration/ })).toHaveCount(1);

  await landscape.selectOption("production");
  await expect(stage).toHaveValue("");
  await expect(config).toHaveValue("");
  const updated = new URL(page.url()).searchParams;
  expect(updated.getAll("landscapeId")).toEqual(["production"]);
  expect(updated.has("promotionStage")).toBe(false);
  expect(updated.has("promotionConfig")).toBe(false);
  expect(updated.get("_data")).toBe("unrelated");
  expect(updated.get("other")).toBe("keep");
  expect(updated.get("embedded")).toBe("1");
});

test("stage URL filters preserve special characters and malformed JSON stays unmatched", async ({
  page,
}) => {
  const landscapeId = "eu &+ ü";
  const stageName = 'pre"prod +& ü';
  const reference = { kind: "Stage", landscape: landscapeId, name: stageName } as const;
  const fixture: VectorPromotionConfig = {
    ...configs[1],
    promotions: [promotion("special-reference", 1, { source: reference })],
    source: reference,
  };
  await page.route(API, (route) => route.fulfill({ json: { data: [fixture] } }));
  const stageValue = JSON.stringify([landscapeId, stageName]);
  const params = new URLSearchParams({ embedded: "1", landscapeId, promotionStage: stageValue });
  const path = `${PATH}?${params}`;
  await signIn(page, path, path);
  const landscape = page.getByRole("combobox", { name: "Filter by landscape" });
  const stage = page.getByRole("combobox", { name: "Filter by stage" });
  await expect(landscape).toHaveValue(landscapeId);
  await expect(stage).toHaveValue(stageValue);
  await expect(page.getByRole("group", { name: "Promotion configuration a-flow" })).toBeVisible();
  await page.reload();
  await expect(stage).toHaveValue(stageValue);

  params.set("promotionStage", "[unbalanced");
  await page.goto(`${PATH}?${params}`);
  await expect(
    page.getByText("No promotions match the current filters", { exact: true }),
  ).toBeVisible();
  await page.getByTestId("promotion-view-clear-filters").click();
  await expect(landscape).toHaveValue("");
  await expect(stage).toHaveValue("");
  await expect(page.getByRole("group", { name: "Promotion configuration a-flow" })).toBeVisible();
});

test("qualifies header and filter references with IDs only for duplicate landscape names", async ({
  page,
}) => {
  const landscapes: Landscape[] = [
    { id: "development", name: "Development" },
    { id: "production-us", name: "Production" },
    { id: "production-eu", name: "Production" },
  ];
  const flow: VectorPromotionConfig = {
    ...configs[1],
    promotions: [],
    source: { kind: "Stage", landscape: "production-us", name: "prod" },
    target: { kind: "Stage", landscape: "production-eu", name: "prod" },
  };
  await page.route("**/v1/projects/payments-platform/landscapes", (route) =>
    route.fulfill({ json: { data: landscapes } }),
  );
  await page.route(API, (route) => route.fulfill({ json: { data: [flow] } }));
  await signIn(page, PATH, PATH);

  const group = page.getByRole("group", { name: "Promotion configuration a-flow" });
  await expect(group).toContainText(
    "Production (production-us)/prod → Production (production-eu)/prod",
  );
  await expect(group.getByText("Production", { exact: true })).toHaveCount(0);
  const landscape = page.getByRole("combobox", { name: "Filter by landscape" });
  await expect(landscape.getByRole("option", { exact: true, name: "Development" })).toHaveAttribute(
    "value",
    "development",
  );
  await expect(
    landscape.getByRole("option", { exact: true, name: "Production (production-us)" }),
  ).toHaveAttribute("value", "production-us");
  await expect(
    landscape.getByRole("option", { exact: true, name: "Production (production-eu)" }),
  ).toHaveAttribute("value", "production-eu");
  const stage = page.getByRole("combobox", { name: "Filter by stage" });
  await expect(
    stage.getByRole("option", { exact: true, name: "Production (production-us)/prod" }),
  ).toHaveAttribute("value", JSON.stringify(["production-us", "prod"]));
  await expect(
    stage.getByRole("option", { exact: true, name: "Production (production-eu)/prod" }),
  ).toHaveAttribute("value", JSON.stringify(["production-eu", "prod"]));
  await stage.selectOption(JSON.stringify(["production-eu", "prod"]));
  await expect(group).toBeVisible();
});

test("Clear filters is visible and keyboard-accessible only with active filters or search", async ({
  page,
}) => {
  await openPromotions(page);
  await expect(page.getByRole("button", { name: "Promotion configuration a-flow" })).toBeVisible();
  const clear = page.getByTestId("promotion-view-clear-filters");
  await expect(clear).toBeHidden();
  await page.getByRole("searchbox", { name: "Search promotions" }).fill("a-flow");
  await expect(clear).toBeVisible();
  await clear.click();
  await expect(clear).toBeHidden();
  await page.getByRole("combobox", { name: "Filter by landscape" }).selectOption("development");
  await expect(clear).toBeVisible();
  await clear.click();
  await expect(clear).toBeHidden();
  await expect(clear).toHaveAttribute("tabindex", "-1");
});

test("configuration search admits children, promotion search narrows priority and counts, and expansion survives", async ({
  page,
}) => {
  await openPromotions(page);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await group.click();
  const search = page.getByRole("searchbox", { name: "Search promotions" });
  await search.fill("  A-FLOW  ");
  await expect(page.getByRole("group", { name: /^Promotion configuration/ })).toHaveCount(1);
  await expect(page.getByRole("button", { name: /^View promotion/ })).toHaveCount(4);
  await search.fill("vector-a-history");
  await expect(page.getByRole("button", { name: /Latest waiting/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "View promotion a-history" })).toBeVisible();
  await expect(page.getByRole("group", { name: "Promotion configuration a-flow" })).toContainText(
    "1 of 4 matching promotions",
  );
  await search.fill("approver@example.com");
  await expect(page.getByRole("button", { name: "View promotion a-history" })).toBeVisible();
  await search.fill("a-older-waiting");
  await expect(
    page.getByRole("button", { name: "Latest waiting promotion a-older-waiting" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "View promotion a-older-waiting" }),
  ).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("button", { name: "View promotion a-older-waiting" })).toBeVisible();
  await search.fill("no-such-result");
  await expect(
    page.getByText("No promotions match the current filters", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { exact: true, name: "Clear filters" }).first().click();
  await expect(group).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("button", { name: "View promotion a-history" })).toBeVisible();
});

test("filters snapshots separately, preserves empty current-reference groups, and cascades compatible options", async ({
  page,
}) => {
  const historical = promotion("old-test", 5, {
    source: { kind: "Stage", landscape: "old-landscape", name: "dev" },
    target: { kind: "Stage", landscape: "production", name: "test" },
  });
  const fixture = [
    { ...configs[1], promotions: [historical] },
    ...configs.filter((config) => config.id !== "a-flow"),
  ];
  await page.route(API, (route) => route.fulfill({ json: { data: fixture } }));
  await signIn(page, PATH, PATH);
  const landscape = page.getByRole("combobox", { name: "Filter by landscape" });
  const stage = page.getByRole("combobox", { name: "Filter by stage" });
  const config = page.getByRole("combobox", { name: "Filter by promotion configuration" });
  await landscape.selectOption("development");
  await expect(page).toHaveURL(/landscapeId=development/);
  await expect(page.getByText("No matching promotions", { exact: true })).toBeVisible();
  await stage.selectOption(JSON.stringify(["development", "dev"]));
  await config.selectOption("a-flow");
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  await landscape.selectOption("old-landscape");
  await expect(stage).toHaveValue("");
  await expect(config).toHaveValue("");
  await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
  await expect(page.getByRole("button", { name: "View promotion old-test" })).toBeVisible();
  await expect(page).not.toHaveURL(/promotionStage=/);
  await stage.selectOption(JSON.stringify(["old-landscape", "dev"]));
  await config.selectOption("a-flow");
  await stage.selectOption("");
  await expect(config).toHaveValue("");
  const sharedUrl = page.url();
  await page.reload();
  await expect(page).toHaveURL(sharedUrl);
  await expect(landscape).toHaveValue("old-landscape");
});

test("rejects cross-endpoint pairs and distinguishes identically named stages", async ({
  page,
}) => {
  const fixture = [
    {
      ...configs[1],
      promotions: [
        promotion("cross-flow", 1, {
          source: { kind: "Stage", landscape: "us", name: "dev" },
          target: { kind: "Stage", landscape: "eu", name: "test" },
        }),
      ],
      source: { kind: "Stage", landscape: "us", name: "dev" },
      target: { kind: "Stage", landscape: "eu", name: "test" },
    },
    { ...configs[0], promotions: [], source: { kind: "Stage", landscape: "us", name: "test" } },
  ];
  await page.route(API, (route) => route.fulfill({ json: { data: fixture } }));
  await signIn(page, PATH, PATH);
  const stage = page.getByRole("combobox", { name: "Filter by stage" });
  await expect(stage.locator("option", { hasText: "eu/test" })).toHaveCount(1);
  await expect(stage.locator("option", { hasText: "us/test" })).toHaveCount(1);
  await page.getByRole("combobox", { name: "Filter by landscape" }).selectOption("us");
  await stage.selectOption(JSON.stringify(["us", "test"]));
  await expect(page.getByRole("group", { name: "Promotion configuration a-flow" })).toHaveCount(0);
  await expect(
    page.getByRole("group", { name: "Promotion configuration z-template" }),
  ).toBeVisible();
});
