import { expect, test } from "@playwright/test";
import type { Locator, Page } from "@playwright/test";

const exampleFor = (page: Page, component: string, title: string): Locator =>
  page
    .locator(`#component-${component}`)
    .getByRole("figure")
    .filter({
      has: page.getByRole("tablist", { exact: true, name: `Example tabs: ${title}` }),
    });
const openCode = async (example: Locator): Promise<Locator> => {
  await example.getByRole("tab", { exact: true, name: "Code" }).click();
  const editor = example.getByRole("textbox");
  await expect(editor).toBeVisible();
  return editor;
};

test("preview is full width by default and editor tabs preserve code and render state", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.scrollIntoViewIfNeeded();
  await expect(example.getByRole("tab", { exact: true, name: "Preview" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await expect(example.getByRole("textbox")).toHaveCount(0);
  const preview = example.frameLocator("iframe");
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
  expect((await example.locator("iframe").boundingBox())!.width).toBeGreaterThan(
    (await example.boundingBox())!.width * 0.8,
  );
  await example.getByRole("button", { exact: true, name: "Copy code" }).click();
  const original = await page.evaluate(() => navigator.clipboard.readText());
  expect(original).toContain('import { Button } from "@konfidence/design-system/components"');
  const editor = await openCode(example);
  await expect(example.locator(".cm-lineNumbers")).toBeVisible();
  await expect(example.getByRole("tabpanel", { exact: true, name: "Preview" })).not.toBeVisible();
  const source =
    '<script lang="ts">\nimport { Button } from "@konfidence/design-system/components";\nlet count = $state(0);\n</script>\n<Button variant="danger" disabled>Edited label</Button>\n<Button onclick={() => count += 1}>Clicks: {count}</Button>';
  await editor.fill(source);
  await example.getByRole("button", { exact: true, name: "Copy code" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(source);
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(preview.getByRole("button", { name: "Edited label" })).toBeDisabled();
  await preview.getByRole("button", { name: "Clicks: 0" }).click();
  await openCode(example);
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(preview.getByRole("button", { name: "Clicks: 1" })).toBeVisible();
  await example.getByRole("button", { exact: true, name: "Reset" }).click();
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
  await example.getByRole("button", { exact: true, name: "Copy code" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(original);
});

test("CodeMirror marks syntax and accessibility problems and clears recovered diagnostics", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  const editor = await openCode(example);
  await editor.fill("<p>unclosed");
  await expect(example.locator(".cm-diagnostic-error")).toBeVisible();
  await expect(example.locator(".cm-lintRange-error")).toHaveCount(1);
  await editor.fill("<img />");
  await expect(example.locator(".cm-diagnostic-warning").filter({ hasText: "alt" })).toBeVisible();
  await editor.fill("<p>Valid</p>");
  await expect(example.locator(".cm-diagnostic-error, .cm-diagnostic-warning")).toHaveCount(0);
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(example.frameLocator("iframe").getByText("Valid", { exact: true })).toBeVisible();
});

test("syntax retains the last render and runtime errors stay isolated and recover", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.scrollIntoViewIfNeeded();
  const preview = example.frameLocator("iframe");
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
  const editor = await openCode(example);
  await editor.fill("<Button");
  await expect(example.getByRole("alert")).toBeVisible();
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
  await openCode(example);
  await editor.fill('<script>throw new Error("demo failure")</script><p>Never rendered</p>');
  await expect(example.getByRole("alert")).toContainText("demo failure");
  await example.getByRole("button", { exact: true, name: "Reset" }).click();
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(example.getByRole("alert")).toHaveCount(0);
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
});

test("sandbox blocks origin access and external network requests", async ({ page }) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  const editor = await openCode(example);
  await editor.fill(
    '<script>let result = ""; try { result = parent.document.title; } catch { result = "Origin access blocked"; }</script><p>{result}</p>',
  );
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  const preview = example.frameLocator("iframe");
  await expect(preview.getByText("Origin access blocked")).toBeVisible();
  await openCode(example);
  await editor.fill(
    '<script>let result = $state("pending"); fetch("https://example.com/").catch(() => result = "Network blocked");</script><p>{result}</p>',
  );
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(preview.getByText("Network blocked")).toBeVisible();
  await expect(example.locator("iframe")).toHaveAttribute("sandbox", "allow-scripts");
});

test("mobile tabs and preview follow appearance changes without horizontal overflow", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.scrollIntoViewIfNeeded();
  const preview = example.frameLocator("iframe");
  await expect(preview.getByRole("button", { exact: true, name: "Primary" })).toBeVisible();
  await expect(preview.locator("html")).toHaveClass(/dark/);
  const copy = example.getByRole("button", { exact: true, name: "Copy code" });
  const foreground = await copy.evaluate((button) => getComputedStyle(button).color);
  await expect(copy.locator("[ui5-icon]")).toHaveCSS("color", foreground);
  await page.locator(".preview-topbar").getByRole("button", { name: "Change color mode" }).click();
  await page.getByRole("menuitemradio", { name: "Light" }).click();
  await expect(preview.locator("html")).not.toHaveClass(/dark/);
  await openCode(example);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test("reset retries a failed compiler load without changing source", async ({ page }) => {
  await page.route(/compiler\.worker-.*\.js$/, (route) => route.abort());
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.scrollIntoViewIfNeeded();
  await expect(example.getByRole("alert")).toContainText("Compiler worker failed");
  await page.unroute(/compiler\.worker-.*\.js$/);
  await example.getByRole("button", { exact: true, name: "Reset" }).click();
  await expect(
    example.frameLocator("iframe").getByRole("button", { exact: true, name: "Primary" }),
  ).toBeVisible();
});

test("menus and dialogs remain interactive inside their own example", async ({ page }) => {
  await page.goto("/#component-Menu");
  const menuExample = exampleFor(page, "Menu", "User menu");
  await menuExample.scrollIntoViewIfNeeded();
  const menu = menuExample.frameLocator("iframe");
  await menu.getByRole("button", { name: "Open user menu" }).click();
  await menu.getByRole("menuitem", { exact: true, name: "Sign out" }).click();
  await expect(menu.getByText("Selected: signout")).toBeVisible();
  await page
    .getByRole("navigation", { exact: true, name: "Components" })
    .getByRole("link", { exact: true, name: "SidePanel" })
    .click();
  const panelExample = exampleFor(page, "SidePanel", "SidePanel — details drawer");
  await panelExample.scrollIntoViewIfNeeded();
  const panel = panelExample.frameLocator("iframe");
  await panel.getByRole("button", { name: "Open details" }).click();
  await expect(panel.getByRole("dialog")).toBeVisible();
  await panel.getByRole("button", { name: "Close details" }).click();
  await expect(panel.getByRole("dialog")).not.toBeVisible();
});

test("rapid edits and component navigation preserve the latest code and rendered state", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  const editor = await openCode(example);
  await editor.fill("<p>Intermediate edit</p>");
  await editor.fill("<p>Latest edit</p>");
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await expect(example.frameLocator("iframe").getByText("Latest edit")).toBeVisible();
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await nav.getByRole("link", { exact: true, name: "Select" }).click();
  await nav.getByRole("link", { exact: true, name: "Button" }).click();
  await expect(example.frameLocator("iframe").getByText("Latest edit")).toBeVisible();
});

test("Cmd/Ctrl+K focuses the one search from both CodeMirror and sandboxed previews", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  const editor = await openCode(example);
  await editor.focus();
  const modifier = await page.getByRole("button", { name: "Focus catalog search" }).textContent();
  const shortcut = modifier === "⌘K" ? "Meta+k" : "Control+k";
  await page.keyboard.press(shortcut);
  await expect(page.getByRole("searchbox", { name: "Search components" })).toBeFocused();
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  await example
    .frameLocator("iframe")
    .getByRole("button", { exact: true, name: "Primary" })
    .focus();
  await page.keyboard.press(shortcut);
  await expect(page.getByRole("searchbox", { name: "Search components" })).toBeFocused();
});

test("an editor loading failure offers an explicit reload and recovers", async ({ page }) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.scrollIntoViewIfNeeded();
  await expect(
    example.frameLocator("iframe").getByRole("button", { exact: true, name: "Primary" }),
  ).toBeVisible();
  let blockNextChunk = true;
  await page.route("**/_app/immutable/chunks/*.js", (route) => {
    if (blockNextChunk) {
      blockNextChunk = false;
      return route.abort();
    }
    return route.continue();
  });
  await example.getByRole("tab", { exact: true, name: "Code" }).click();
  await expect(example.getByRole("alert")).toContainText("Could not load the editor");
  await expect(example.getByText("Reloading resets local example edits.")).toBeVisible();
  await example.getByRole("button", { exact: true, name: "Reload preview" }).click();
  await expect(example.getByRole("tab", { exact: true, name: "Preview" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await openCode(example);
});

test("status filtering preserves edited examples without destroying their frames", async ({
  page,
}) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  const editor = await openCode(example);
  await editor.fill(
    "<script>let count = $state(0);</script><button onclick={() => count += 1}>Filtered clicks: {count}</button>",
  );
  await example.getByRole("tab", { exact: true, name: "Preview" }).click();
  const preview = example.frameLocator("iframe");
  await preview.getByRole("button", { exact: true, name: "Filtered clicks: 0" }).click();
  await expect(
    preview.getByRole("button", { exact: true, name: "Filtered clicks: 1" }),
  ).toBeVisible();
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  const implemented = legend.getByRole("button", { exact: true, name: "Implemented" });
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await nav.getByRole("link", { exact: true, name: "Select" }).click();
  await expect(
    page.locator("#component-Select").getByRole("heading", { exact: true, name: "Select" }),
  ).toBeInViewport();
  await implemented.click();
  await expect(example).not.toBeVisible();
  await expect(
    page.locator('#component-Button iframe[title="Live preview: Variants"]'),
  ).toHaveCount(1);
  await implemented.click();
  await nav.getByRole("link", { exact: true, name: "Button" }).click();
  await expect(
    preview.getByRole("button", { exact: true, name: "Filtered clicks: 1" }),
  ).toBeVisible();
});

test("tabs support arrow keys and the editor does not trap Tab", async ({ page }) => {
  await page.goto("/#component-Button");
  const example = exampleFor(page, "Button", "Variants");
  await example.getByRole("tab", { exact: true, name: "Preview" }).focus();
  await page.keyboard.press("ArrowRight");
  await expect(example.getByRole("tab", { exact: true, name: "Code" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  const editor = example.getByRole("textbox");
  await expect(editor).toBeVisible();
  await editor.focus();
  await page.keyboard.press("Tab");
  await expect(editor).not.toBeFocused();
});
