import { expect, test } from "@playwright/test";

test("list examples keep bound and local selection and actions independent", async ({ page }) => {
  await page.goto("/#lists");
  const lists = page.locator("#component-ExpandableGroup");
  await lists.getByText("Single-line rows", { exact: true }).scrollIntoViewIfNeeded();
  const singleLine = lists.frameLocator('iframe[title="Live preview: Single-line rows"]');
  const development = singleLine.getByRole("group", {
    exact: true,
    name: "Single-line items",
  });
  const first = development.getByRole("button", { name: "Select first single-line item" });
  const second = development.getByRole("button", { name: "Select second single-line item" });
  await expect(first).toHaveAttribute("aria-pressed", "true");
  await second.click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  await expect(first).toHaveAttribute("aria-pressed", "false");
  await development.getByRole("button", { name: "Inspect" }).click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  await expect(singleLine.getByText("Actions: 1.", { exact: false })).toBeVisible();
  const trigger = development.getByRole("button", { name: "Single-line items" });
  await trigger.click();
  await expect(second).not.toBeVisible();
  await trigger.click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  await lists.getByText("Multiple-line rows", { exact: true }).scrollIntoViewIfNeeded();
  const multipleLine = lists.frameLocator('iframe[title="Live preview: Multiple-line rows"]');
  const production = multipleLine.getByRole("group", {
    exact: true,
    name: "Other items",
  });
  await production.getByRole("button", { name: "Other items" }).click();
  await production.getByRole("button", { name: "Select other item" }).click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  const files = multipleLine.getByRole("group", { exact: true, name: "Multiple-line items" });
  await files.getByText("Details", { exact: true }).click();
  await expect(
    files.getByRole("button", { name: "Select first multiple-line item" }),
  ).toHaveAttribute("aria-pressed", "true");
  await multipleLine.getByRole("button", { name: "Clear selection" }).click();
  await expect(
    files.getByRole("button", { name: "Select first multiple-line item" }),
  ).toHaveAttribute("aria-pressed", "false");
  await lists.getByText("Interactive header", { exact: true }).scrollIntoViewIfNeeded();
  const interactive = lists.frameLocator('iframe[title="Live preview: Interactive header"]');
  const history = interactive.getByRole("group", { exact: true, name: "Items with header action" });
  const shortcut = history.getByRole("button", { name: "Show current" });
  await shortcut.click();
  await expect(history.getByRole("button", { name: "Items with header action" })).toHaveAttribute(
    "aria-expanded",
    "true",
  );
  await expect(history.getByRole("button", { name: "Select current item" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await shortcut.click();
  await expect(history.getByRole("button", { name: "Select current item" })).toBeVisible();
  await expect(history.getByRole("button", { name: "Unavailable action" })).toBeDisabled();
  await lists.getByText("Local selection", { exact: true }).scrollIntoViewIfNeeded();
  const local = lists
    .frameLocator('iframe[title="Live preview: Local selection"]')
    .getByRole("button", { name: "Select local item" });
  await local.click();
  await local.click();
  await expect(local).toHaveAttribute("aria-pressed", "true");
});
