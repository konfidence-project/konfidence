import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH } from "./helpers.ts";

test("arrival updates the collapsed header without revealing any rows", async ({ page }) => {
  await page.clock.install();
  let current = configs;
  await page.route(API, (route) => route.fulfill({ json: { data: current } }));
  await signIn(page, PATH, PATH);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await expect(group).toHaveAttribute("aria-expanded", "false");
  current = structuredClone(configs);
  current[1].promotions.push(promotion("arrived", 20));
  await page.clock.runFor(10_000);
  await expect(group).toContainText("Latest waiting: vector-arrived");
  await expect(group).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByRole("button", { name: "View promotion arrived" })).toBeHidden();
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toBeHidden();
});

const setVisibility = async (page: Page, value: "visible" | "hidden"): Promise<void> => {
  await page.evaluate((visibility) => {
    Object.defineProperty(document, "visibilityState", { configurable: true, value: visibility });
    document.dispatchEvent(new Event("visibilitychange"));
  }, value);
};

test("refreshes every ten visible seconds, suspends while hidden, refreshes on return and cleans up", async ({
  page,
}) => {
  await page.clock.install();
  let requests = 0;
  await page.route(API, (route) => {
    requests += 1;
    return route.fulfill({ json: { data: configs } });
  });
  await signIn(page, PATH, PATH);
  await expect(page.getByRole("button", { name: "Promotion configuration a-flow" })).toBeVisible();
  const initial = requests;
  await page.clock.runFor(9000);
  expect(requests).toBe(initial);
  await page.clock.runFor(1000);
  await expect.poll(() => requests).toBe(initial + 1);
  await setVisibility(page, "hidden");
  const hidden = requests;
  await page.clock.runFor(30_000);
  expect(requests).toBe(hidden);
  await setVisibility(page, "visible");
  await expect.poll(() => requests).toBe(hidden + 1);
  await page.getByTestId("nav-vector-deployments").click();
  await expect(page.getByTestId("page-heading")).toHaveText("Vector Deployments");
  const departed = requests;
  await setVisibility(page, "hidden");
  await setVisibility(page, "visible");
  await page.clock.runFor(30_000);
  expect(requests).toBe(departed);
});

test("keeps expansion, filters and selected rows stable as promotions arrive and states change", async ({
  page,
}) => {
  await page.clock.install();
  let current = configs;
  await page.route(API, (route) => route.fulfill({ json: { data: current } }));
  await signIn(page, PATH, PATH);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await group.click();
  await page.getByRole("searchbox", { name: "Search promotions" }).fill("a-flow");
  await page
    .getByRole("combobox", { name: "Filter by promotion configuration" })
    .selectOption("a-flow");
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  const row = page.getByRole("button", { exact: true, name: "View promotion a-waiting" });
  await row.click();
  await row.evaluate((element) => {
    element.setAttribute("data-continuity-check", "retained");
  });
  current = configs.map((config) =>
    config.id === "a-flow"
      ? {
          ...config,
          promotions: [
            promotion("a-arrived", 20),
            ...config.promotions.map((item) =>
              item.id === "a-waiting"
                ? {
                    ...item,
                    gates: [
                      {
                        setBy: "external-approver",
                        state: "Satisfied" as const,
                        type: "explicit-approval.konfidence.cloud",
                      },
                    ],
                    status: "Succeeded" as const,
                  }
                : item,
            ),
          ],
        }
      : config,
  );
  await page.clock.runFor(10_000);
  await expect(page.getByRole("button", { name: "View promotion a-arrived" })).toBeVisible();
  await expect(group).toHaveAttribute("aria-expanded", "true");
  await expect(row).toHaveAttribute("data-continuity-check", "retained");
  await expect(row).toHaveAttribute("aria-pressed", "true");
  await expect(row).toContainText("Succeeded");
  await expect(row.getByRole("button", { exact: true, name: "Approve" })).toHaveCount(0);
  await expect(page).toHaveURL(/promotionConfig=a-flow/);
  await expect(page.getByRole("searchbox", { name: "Search promotions" })).toHaveValue("a-flow");
});

test("retains data on failed refreshes, retries and removes rows only after success", async ({
  page,
}) => {
  await page.clock.install();
  let fail = false;
  let current = configs;
  await page.route(API, (route) =>
    route.fulfill(fail ? { json: {}, status: 503 } : { json: { data: current } }),
  );
  await signIn(page, PATH, PATH);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await group.click();
  const row = page.getByRole("button", { exact: true, name: "View promotion a-history" });
  await expect(row).toBeVisible();
  fail = true;
  await page.clock.runFor(10_000);
  await expect(page.getByRole("alert")).toContainText("Showing previously loaded data");
  await expect(row).toBeVisible();
  await expect(group).toHaveAttribute("aria-expanded", "true");
  fail = false;
  current = structuredClone(configs);
  current[1].promotions.push(promotion("retried", 20));
  await page.getByRole("button", { name: "Retry" }).click();
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion retried" }),
  ).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(row).toBeVisible();
  current = structuredClone(configs);
  for (const config of current) {
    config.promotions = config.promotions.filter((item) => item.id !== "a-history");
  }
  await page.clock.runFor(10_000);
  await expect(row).toHaveCount(0);
  await expect(group).toHaveAttribute("aria-expanded", "true");
});

test("continues polling without waiting promotions and recovers automatically after failures", async ({
  page,
}) => {
  await page.clock.install();
  let requests = 0;
  let fail = false;
  await page.route(API, (route) => {
    requests += 1;
    return route.fulfill(fail ? { json: {}, status: 503 } : { json: { data: [configs[0]] } });
  });
  await signIn(page, PATH, PATH);
  await expect(
    page.getByRole("button", { name: "Promotion configuration z-template" }),
  ).toBeVisible();
  fail = true;
  await page.clock.runFor(10_000);
  await expect(page.getByRole("alert")).toBeVisible();
  const failed = requests;
  fail = false;
  await page.clock.runFor(10_000);
  await expect.poll(() => requests).toBe(failed + 1);
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Promotion configuration z-template" }),
  ).toHaveAttribute("aria-expanded", "false");
});

test("deduplicates slow refreshes without unmounting loaded content", async ({ page }) => {
  await page.clock.install();
  let requests = 0;
  let release = () => {};
  const ready = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(API, async (route) => {
    requests += 1;
    if (requests > 1) {
      await ready;
    }
    await route.fulfill({ json: { data: configs } });
  });
  await signIn(page, PATH, PATH);
  const group = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await group.click();
  await page.clock.runFor(10_000);
  await expect.poll(() => requests).toBe(2);
  await page.clock.runFor(20_000);
  expect(requests).toBe(2);
  await expect(group).toHaveAttribute("aria-expanded", "true");
  await expect(
    page.getByRole("button", { exact: true, name: "View promotion a-waiting" }),
  ).toBeVisible();
  await expect(page.getByText("Loading promotions…", { exact: true })).toHaveCount(0);
  release();
});
