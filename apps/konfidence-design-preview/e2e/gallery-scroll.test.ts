import { expect, test } from "@playwright/test";

test("scrolling replaces the deep link with the visible component without adding history or moving focus", async ({
  page,
}) => {
  await page.setViewportSize({ height: 1000, width: 1440 });
  await page.goto("/#component-Button");
  await expect(
    page.locator("#component-Button").getByRole("heading", { exact: true, name: "Button" }),
  ).toBeInViewport();
  // Let the initial browser anchor animation land before exercising wheel input.
  await expect
    .poll(async () => Math.round((await page.locator("#component-Button").boundingBox())!.y))
    .toBe(96);
  const focus = page.getByRole("button", { name: "Focus catalog search" });
  await focus.focus();
  const historyLength = await page.evaluate(() => globalThis.history.length);
  const target = (await page.locator("#component-SearchInput").boundingBox())!;
  await page.mouse.move(1400, 200);
  await page.mouse.wheel(0, target.y - 96);
  await expect(page).toHaveURL(/#component-SearchInput$/);
  await expect(
    page
      .getByRole("navigation", { exact: true, name: "Components" })
      .getByRole("link", { exact: true, name: "SearchInput" }),
  ).toHaveAttribute("aria-current", "location");
  expect(await page.evaluate(() => globalThis.history.length)).toBe(historyLength);
  await expect(focus).toBeFocused();
  await page.reload();
  await expect(
    page
      .locator("#component-SearchInput")
      .getByRole("heading", { exact: true, name: "SearchInput" }),
  ).toBeInViewport();
});

test("scroll deep links update the selected-component exception without changing status filters", async ({
  page,
}) => {
  await page.setViewportSize({ height: 1000, width: 1440 });
  await page.goto("/#component-Dialog");
  await expect
    .poll(async () => Math.round((await page.locator("#component-Dialog").boundingBox())!.y))
    .toBe(96);
  const target = (await page.locator("#component-EmptyState").boundingBox())!;
  await page.mouse.move(1400, 200);
  await page.mouse.wheel(0, target.y - 96);
  await expect(page).toHaveURL(/#component-EmptyState$/);
  await expect(page.locator("#component-Dialog")).not.toBeVisible();
  await expect(
    page.locator("#component-EmptyState").getByRole("heading", { exact: true, name: "EmptyState" }),
  ).toBeInViewport();
  await expect(
    page
      .getByRole("group", { exact: true, name: "Component status legend" })
      .getByRole("button", { exact: true, name: "Planned" }),
  ).toHaveAttribute("aria-pressed", "false");
});

test("native link navigation keeps its destination rather than components passed during animated scrolling", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await nav.getByRole("link", { exact: true, name: "Table" }).click();
  await expect(
    page.locator("#component-Table").getByRole("heading", { exact: true, name: "Table" }),
  ).toBeInViewport();
  await expect(page).toHaveURL(/#component-Table$/);
  await nav.getByRole("link", { exact: true, name: "Select" }).click();
  await expect(
    page.locator("#component-Select").getByRole("heading", { exact: true, name: "Select" }),
  ).toBeInViewport();
  await expect(page).toHaveURL(/#component-Select$/);
  await page.goBack();
  await expect(page).toHaveURL(/#component-Table$/);
});
