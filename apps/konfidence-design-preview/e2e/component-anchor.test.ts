import { expect, test } from "@playwright/test";

test("implemented, planned and composed component cards have their own heading anchors", async ({
  page,
}) => {
  await page.goto("/");
  await Promise.all(
    ["Button", "Table", "TableRow", "Dialog", "ExpandableGroup", "StageCardPlaceholder"].map(
      async (name) => {
        await expect(
          page
            .locator(`#component-${name}`)
            .getByRole("link", { exact: true, includeHidden: true, name: `Copy link to ${name}` }),
        ).toHaveAttribute("href", `/#component-${name}`);
      },
    ),
  );
});

test("component anchors copy their own full deep link without navigation", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/#component-Button");
  const component = page.locator("#component-Button");
  await expect.poll(async () => Math.round((await component.boundingBox())!.y)).toBe(96);
  const before = await page.evaluate(() => ({
    history: history.length,
    scroll: scrollY,
    url: location.href,
  }));
  await component.getByRole("link", { exact: true, name: "Copy link to Button" }).click();
  await expect(component.getByText("Link copied", { exact: true })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    "http://127.0.0.1:4174/#component-Button",
  );
  expect(
    await page.evaluate(() => ({ history: history.length, scroll: scrollY, url: location.href })),
  ).toEqual(before);
  await page.getByRole("searchbox", { name: "Search components" }).fill("Dialog");
  const planned = page.locator("#component-Dialog");
  await planned.getByRole("link", { exact: true, name: "Copy link to Dialog" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    "http://127.0.0.1:4174/#component-Dialog",
  );
});

test("component anchors preserve native new-tab behavior and provide a clipboard fallback", async ({
  page,
  context,
}) => {
  await page.goto("/#component-Button");
  const anchor = page
    .locator("#component-Button")
    .getByRole("link", { exact: true, name: "Copy link to Button" });
  await expect(anchor).toHaveAttribute("href", "/#component-Button");
  const opened = context.waitForEvent("page");
  await anchor.click({ modifiers: ["ControlOrMeta"] });
  const newTab = await opened;
  await expect(
    newTab.locator("#component-Button").getByRole("heading", { exact: true, name: "Button" }),
  ).toBeInViewport();
  await newTab.close();
  await page.evaluate(() => {
    navigator.clipboard.writeText = async () => {
      throw new Error("Clipboard denied");
    };
  });
  await anchor.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.locator("#component-Button").getByText("Clipboard unavailable.", { exact: false }),
  ).toBeVisible();
});
