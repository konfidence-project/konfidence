import { expect, test } from "@playwright/test";
import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH, openPromotions } from "./helpers.ts";

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
