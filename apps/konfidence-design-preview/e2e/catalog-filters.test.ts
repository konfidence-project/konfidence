import { expect, test } from "@playwright/test";

test("status legend filters default to implemented and support either, both and neither", async ({
  page,
}) => {
  await page.goto("/");
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  const implemented = legend.getByRole("button", { exact: true, name: "Implemented" });
  const planned = legend.getByRole("button", { exact: true, name: "Planned" });
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await expect(implemented).toHaveAttribute("aria-pressed", "true");
  await expect(planned).toHaveAttribute("aria-pressed", "false");
  await expect(nav.getByRole("link", { exact: true, name: "Button" })).toBeAttached();
  await expect(nav.getByRole("link", { exact: true, name: "Dialog" })).toHaveCount(0);
  await expect(page.locator("#component-Dialog")).toBeAttached();
  await expect(page.locator("#component-Dialog")).not.toBeVisible();
  await planned.click();
  await expect(planned).toHaveAttribute("aria-pressed", "true");
  await expect(nav.getByRole("link", { exact: true, name: "Dialog" })).toBeAttached();
  await expect(page.locator("#component-Dialog")).toBeVisible();
  await implemented.click();
  await expect(implemented).toHaveAttribute("aria-pressed", "false");
  await expect(nav.getByRole("link", { exact: true, name: "Button" })).toHaveCount(0);
  await expect(page.locator("#component-Button")).not.toBeVisible();
  await page.getByRole("searchbox", { name: "Search components" }).fill("unknown-component");
  await expect(nav.getByText("No matching components.")).toBeVisible();
  await expect(page.locator("#component-Dialog")).toBeVisible();
  await expect(page.locator("#component-Button")).toBeVisible();
  await page.getByRole("searchbox", { name: "Search components" }).fill("");
  await expect(page.locator("#component-Button")).not.toBeVisible();
  await planned.focus();
  await page.keyboard.press("Space");
  await expect(planned).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByText("Select Implemented or Planned to show components.")).toBeVisible();
  await implemented.click();
  await expect(page.locator("#component-Button")).toBeVisible();
  await page.getByRole("searchbox", { name: "Search components" }).fill("");
  await expect(nav.getByRole("link", { exact: true, name: "Button" })).toBeAttached();
});

test("direct planned links reveal their target without losing native anchor navigation", async ({
  page,
}) => {
  await page.goto("/#component-Dialog");
  await expect(
    page.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  await expect(legend.getByRole("button", { exact: true, name: "Planned" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(legend.getByRole("button", { exact: true, name: "Implemented" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await page.goto("/?component=Dialog");
  await expect(page).toHaveURL(/#component-Dialog$/);
  await expect(
    page.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
  await expect(page.locator("#component-KpiTile")).not.toBeVisible();
});

test("filter choices reset on reload and never change to match a selected URL", async ({
  page,
}) => {
  await page.goto("/#component-Dialog");
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  const implemented = legend.getByRole("button", { exact: true, name: "Implemented" });
  const planned = legend.getByRole("button", { exact: true, name: "Planned" });
  await expect(implemented).toHaveAttribute("aria-pressed", "true");
  await expect(planned).toHaveAttribute("aria-pressed", "false");
  await implemented.click();
  await expect(page.locator("#component-Dialog")).toBeVisible();
  await expect(page.getByText("Select Implemented or Planned to show components.")).toHaveCount(0);
  await planned.click();
  await page.reload();
  await expect(implemented).toHaveAttribute("aria-pressed", "true");
  await expect(planned).toHaveAttribute("aria-pressed", "false");
  await expect(page.locator("#component-Dialog")).toBeVisible();
  await expect(page.locator("#component-KpiTile")).not.toBeVisible();
});

test("modified search-result clicks preserve the current page and open a default-filtered selected target", async ({
  page,
  context,
}) => {
  await page.goto("/");
  const search = page.getByRole("searchbox", { name: "Search components" });
  await search.fill("Dialog");
  const opened = context.waitForEvent("page");
  await page
    .getByRole("navigation", { exact: true, name: "Components" })
    .getByRole("link", { exact: true, name: "Dialog" })
    .click({ modifiers: ["ControlOrMeta"] });
  const newTab = await opened;
  await expect(search).toHaveValue("Dialog");
  await expect(page.locator("#component-KpiTile")).toBeVisible();
  await expect(
    newTab.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
  const legend = newTab.getByRole("group", { exact: true, name: "Component status legend" });
  await expect(legend.getByRole("button", { exact: true, name: "Implemented" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(legend.getByRole("button", { exact: true, name: "Planned" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await newTab.close();
});

test("search includes both statuses and selection restores filters with only the selected exception", async ({
  page,
}) => {
  await page.goto("/");
  const search = page.getByRole("searchbox", { name: "Search components" });
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  const legend = page.getByRole("group", { exact: true, name: "Component status legend" });
  await search.fill("Dialog");
  await expect(nav.getByRole("link", { exact: true, name: "Dialog" })).toBeVisible();
  await expect(page.locator("#component-KpiTile")).toBeVisible();
  await nav.getByRole("link", { exact: true, name: "Dialog" }).click();
  await expect(search).toHaveValue("");
  await expect(
    page.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
  await expect(legend.getByRole("button", { exact: true, name: "Implemented" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(legend.getByRole("button", { exact: true, name: "Planned" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(page.locator("#component-KpiTile")).not.toBeVisible();
  await expect(nav.getByRole("link", { exact: true, name: "KpiTile" })).toHaveCount(0);
  await search.fill("Dialog");
  await page.keyboard.press("Enter");
  await expect(search).toHaveValue("");
  await nav.getByRole("link", { exact: true, name: "Button" }).click();
  await expect(
    page.locator("#component-Button").getByRole("heading", { exact: true, name: "Button" }),
  ).toBeInViewport();
  await expect(page.locator("#component-Dialog")).not.toBeVisible();
  await page.goBack();
  await expect(
    page.locator("#component-Dialog").getByRole("heading", { exact: true, name: "Dialog" }),
  ).toBeInViewport();
});
