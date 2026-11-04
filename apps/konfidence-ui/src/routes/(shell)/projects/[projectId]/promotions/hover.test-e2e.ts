import { expect, test } from "@playwright/test";
import { openPromotions } from "./helpers.ts";

test("expanded header ends at the row separator without an unpainted band", async ({
  page,
}, testInfo) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await openPromotions(page);
  const header = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await header.click();
  await header.hover();
  const headerBox = await header.boundingBox();
  const rowBox = await page.locator('[data-row-id="a-newest"]').boundingBox();
  expect(headerBox).not.toBeNull();
  expect(rowBox).not.toBeNull();
  expect(rowBox!.y - (headerBox!.y + headerBox!.height)).toBeLessThanOrEqual(1);
  await page
    .getByRole("group", { name: "Promotion configuration a-flow" })
    .screenshot({ path: testInfo.outputPath("header-hover.png") });
});

test("promotion hover paints the complete row including its left and right gutters", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ height: 1000, width: 1440 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await openPromotions(page);
  await page.getByRole("button", { name: "Promotion configuration a-flow" }).click();
  const row = page.locator('[data-row-id="a-waiting"]');
  await row.hover();
  const paint = await row.evaluate((element) => {
    const box = element.getBoundingClientRect();
    const backgroundAt = (horizontal: number): string | undefined =>
      document
        .elementsFromPoint(horizontal, box.y + box.height / 2)
        .map((node) => getComputedStyle(node).backgroundColor)
        .find((color) => color !== "rgba(0, 0, 0, 0)" && color !== "transparent");
    return { left: backgroundAt(box.left + 2), right: backgroundAt(box.right - 2) };
  });
  expect(paint.left).toBe(paint.right);
  const groupBackground = await page
    .getByRole("group", { name: "Promotion configuration a-flow" })
    .evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(paint.left).not.toBe(groupBackground);
  const box = await row.boundingBox();
  if (!box) {
    throw new Error("Row has no painted bounds");
  }
  await page.mouse.move(box.x + 2, box.y + box.height / 2);
  await row.screenshot({ path: testInfo.outputPath("row-hover.png") });
});
