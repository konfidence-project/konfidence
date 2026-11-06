import { expect, test } from "@playwright/test";

test("list examples keep group selection, disabled rows and actions independent", async ({
  page,
}) => {
  await page.goto("/#lists");
  const lists = page.locator("#lists");
  const development = lists.getByRole("group", {
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
  await expect(lists.getByText("Actions: 1.", { exact: false })).toBeVisible();
  await expect(
    development.getByRole("button", { name: "Unavailable single-line item" }),
  ).toBeDisabled();
  const trigger = development.getByRole("button", { name: "Single-line items" });
  await trigger.click();
  await expect(second).not.toBeVisible();
  await trigger.click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  const production = lists.getByRole("group", {
    exact: true,
    name: "Other items",
  });
  await production.getByRole("button", { name: "Other items" }).click();
  await production.getByRole("button", { name: "Select other item" }).click();
  await expect(second).toHaveAttribute("aria-pressed", "true");
  const files = lists.getByRole("group", { exact: true, name: "Multiple-line items" });
  await files.getByText("Details", { exact: true }).click();
  await expect(
    files.getByRole("button", { name: "Select first multiple-line item" }),
  ).toHaveAttribute("aria-pressed", "true");
  await lists.getByRole("button", { name: "Clear selection" }).click();
  await expect(
    files.getByRole("button", { name: "Select first multiple-line item" }),
  ).toHaveAttribute("aria-pressed", "false");
  const history = lists.getByRole("group", { exact: true, name: "Items with summary" });
  await history.getByRole("button", { name: "Inspect" }).click();
  await expect(history.getByRole("button", { name: "Items with summary" })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  await expect(history.getByRole("button", { name: "Select always-visible item" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
});
