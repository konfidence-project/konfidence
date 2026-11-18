import { expect, test } from "@playwright/test";

test("flat links scroll to components without hiding the gallery, including history and new tabs", async ({
  page,
  context,
}) => {
  await page.goto("/");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await expect(nav.getByRole("link", { exact: true, name: "Data blocks" })).toHaveCount(0);
  await nav.getByRole("link", { exact: true, name: "Table" }).click();
  await expect(page).toHaveURL(/#component-Table$/);
  await expect(
    page.locator("#component-Table").getByRole("heading", { exact: true, name: "Table" }),
  ).toBeInViewport();
  await expect(page.locator("#component-Button")).toBeAttached();
  await expect(page.locator("#component-Select")).toBeAttached();
  await expect(nav.getByRole("link", { exact: true, name: "Table" })).toHaveAttribute(
    "aria-current",
    "location",
  );
  const popup = context.waitForEvent("page");
  await nav
    .getByRole("link", { exact: true, name: "SearchInput" })
    .click({ modifiers: ["ControlOrMeta"] });
  const newTab = await popup;
  await expect(
    newTab
      .locator("#component-SearchInput")
      .getByRole("heading", { exact: true, name: "SearchInput" }),
  ).toBeInViewport();
  await newTab.close();
  await nav.getByRole("link", { exact: true, name: "Select" }).click();
  await expect(page).toHaveURL(/#component-Select$/);
  // Let the native animated anchor navigation finish before recording the
  // back/forward journey; otherwise history correctly restores a midpoint.
  await expect(
    page.locator("#component-Select").getByRole("heading", { exact: true, name: "Select" }),
  ).toBeInViewport();
  await page.goBack();
  await expect(
    page.locator("#component-Table").getByRole("heading", { exact: true, name: "Table" }),
  ).toBeInViewport();
  await page.goForward();
  await expect(
    page.locator("#component-Select").getByRole("heading", { exact: true, name: "Select" }),
  ).toBeInViewport();
  await page.reload();
  await expect(
    page.locator("#component-Select").getByRole("heading", { exact: true, name: "Select" }),
  ).toBeInViewport();
});

test("one search supports keywords, keyboard navigation and empty result recovery without filtering the gallery", async ({
  page,
}) => {
  await page.goto("/");
  const input = page.getByRole("searchbox", { name: "Search components" });
  await expect(page.getByRole("searchbox", { name: "Search components" })).toHaveCount(1);
  await input.fill("drawer");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await expect(nav.getByRole("link")).toHaveCount(1);
  await expect(page.locator("#component-Table")).toBeAttached();
  await page.keyboard.press("ArrowDown");
  await expect(nav.getByRole("link", { exact: true, name: "SidePanel" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(
    page.locator("#component-SidePanel").getByRole("heading", { exact: true, name: "SidePanel" }),
  ).toBeInViewport();
  await expect(input).toHaveValue("");
  await page.getByRole("button", { name: "Focus catalog search" }).click();
  await expect(input).toBeFocused();
  await input.fill("no-such-component");
  await expect(nav.getByText("No matching components.")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(input).toHaveValue("");
  await expect(nav.getByRole("link", { exact: true, name: "Table" })).toBeAttached();
});

for (const platform of [
  { hint: "⌘K", key: "Meta+k", userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" },
  { hint: "Ctrl+K", key: "Control+k", userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)" },
]) {
  test(`search displays and handles ${platform.hint} for its OS`, async ({ browser }) => {
    const context = await browser.newContext({ userAgent: platform.userAgent });
    const page = await context.newPage();
    await page.goto("http://127.0.0.1:4174/");
    await expect(page.getByRole("button", { name: "Focus catalog search" })).toHaveText(
      platform.hint,
    );
    await page.keyboard.press(platform.key);
    await expect(page.getByRole("searchbox", { name: "Search components" })).toBeFocused();
    await context.close();
  });
}

test("READMEs are truncated, expandable and independent for shared documentation", async ({
  page,
}) => {
  await page.goto("/#component-ExpandableGroup");
  const docs = page
    .locator("#component-ExpandableGroup")
    .getByRole("region", { name: "Documentation for ExpandableGroup" });
  const expand = docs.getByRole("button", { name: "Expand documentation" });
  await expect(expand).toHaveAttribute("aria-expanded", "false");
  await expect(docs.getByRole("button")).toHaveCount(1);
  await expect(page.getByRole("button", { exact: true, name: "Open README" })).toHaveCount(0);
  await expect(docs.getByRole("heading", { name: "Bindings and ownership" })).toHaveCount(0);
  await expand.click();
  await expect(
    page.getByRole("dialog", { exact: true, name: "ExpandableGroup README" }),
  ).toHaveCount(0);
  await expect(docs.getByRole("heading", { name: "Bindings and ownership" })).toBeVisible();
  await expect(docs.locator("pre code").first()).toContainText(
    "import { ExpandableGroup, SelectableRow }",
  );
  await docs.getByRole("button", { name: "Collapse documentation" }).click();
  await expect(docs.getByRole("heading", { name: "Bindings and ownership" })).toHaveCount(0);
  const shared = page
    .locator("#component-SelectableRow")
    .getByRole("region", { name: "Documentation for ExpandableGroup" });
  await expect(shared.getByRole("button", { name: "Expand documentation" })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  await page.goto("/?component=Button");
  await expect(
    page.locator("#component-Button").getByRole("heading", { exact: true, name: "Button" }),
  ).toBeInViewport();
});

test("missing READMEs render nothing while existing documentation stays inside the card", async ({
  page,
}) => {
  await page.goto("/");
  await Promise.all(
    ["Button", "Link", "Brandbar"].map(async (name) => {
      const card = page.locator(`#component-${name} .gallery-section`);
      await expect(card.getByRole("heading", { exact: true, name })).toBeAttached();
      await expect(
        card.getByRole("region", { exact: true, name: `Documentation for ${name}` }),
      ).toHaveCount(0);
    }),
  );
  await expect(
    page.getByText("No component README yet. Explore the live examples below.", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page
      .locator("#component-ExpandableGroup .gallery-section")
      .getByRole("button", { exact: true, name: "Expand documentation" }),
  ).toBeAttached();
});

test("READMEs expand and collapse inline on mobile without a dialog", async ({ page }) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/#component-SelectableRow");
  const docs = page
    .locator("#component-SelectableRow .gallery-section")
    .getByRole("region", { name: "Documentation for ExpandableGroup" });
  await docs.getByRole("button", { exact: true, name: "Expand documentation" }).click();
  await expect(docs.getByRole("heading", { name: "Bindings and ownership" })).toBeAttached();
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await expect(
    page.getByRole("dialog", { exact: true, name: "ExpandableGroup README" }),
  ).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await docs.getByRole("button", { exact: true, name: "Collapse documentation" }).click();
  await expect(docs.getByRole("heading", { name: "Bindings and ownership" })).toHaveCount(0);
});

test("navigation uses accessible status dots and one legend below search, not per-item tags", async ({
  page,
}) => {
  await page.goto("/");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  await expect(legend.getByText("Implemented", { exact: true })).toBeVisible();
  await expect(legend.getByText("Planned", { exact: true })).toBeVisible();
  const searchBounds = (await page
    .getByRole("searchbox", { name: "Search components" })
    .boundingBox())!;
  expect((await legend.boundingBox())!.y).toBeGreaterThanOrEqual(
    searchBounds.y + searchBounds.height,
  );
  await expect(nav.getByText("Implemented", { exact: true })).toHaveCount(0);
  await expect(nav.getByText("Planned", { exact: true })).toHaveCount(0);
  const implemented = nav.getByRole("link", { exact: true, name: "Button" });
  await expect(implemented.getByRole("img", { exact: true, name: "Implemented" })).toHaveCount(1);
  await expect(implemented).toHaveAccessibleDescription("Implemented");
  await legend.getByRole("button", { exact: true, name: "Planned" }).click();
  const planned = nav.getByRole("link", { exact: true, name: "Dialog" });
  await expect(planned.getByRole("img", { exact: true, name: "Planned" })).toHaveCount(1);
  await expect(planned).toHaveAccessibleDescription("Planned");
  await expect(
    page.locator("#component-Button").getByText("Implemented", { exact: true }),
  ).toBeAttached();
});

test("implemented and planned components remain labeled and reachable on mobile in dark mode", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  const input = page.getByRole("searchbox", { name: "Search components" });
  await input.fill("Dialog");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await expect(nav.getByRole("link", { exact: true, name: "Dialog" })).toHaveAccessibleDescription(
    "Planned",
  );
  await page.keyboard.press("Enter");
  await expect(
    page.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
  await expect(
    page.locator("#component-Dialog").getByText("Planned", { exact: true }),
  ).toBeVisible();
  await input.fill("Table");
  await expect(nav.getByRole("link", { exact: true, name: "Table" })).toHaveAccessibleDescription(
    "Implemented",
  );
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
