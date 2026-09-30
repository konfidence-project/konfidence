import { expect, test } from "@playwright/test";

test("selector persists explicit mode and exposes checked radio semantics", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Change color mode" }).click();
  await page.getByRole("menuitemradio", { name: "Dark" }).click();
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("mode-watcher-mode")))
    .toBe("dark");
  await page.reload();
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await page.getByRole("button", { name: "Change color mode" }).click();
  await expect(page.getByRole("menuitemradio", { name: "Dark" })).toHaveAttribute(
    "aria-checked",
    "true",
  );
  await page.getByRole("menuitemradio", { name: "Light" }).click();
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
});

test("system tracks OS appearance and explicit light ignores it", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await page.getByRole("button", { name: "Change color mode" }).click();
  await expect(page.getByRole("menuitemradio", { name: "System" })).toHaveAttribute(
    "aria-checked",
    "true",
  );
  await page.getByRole("menuitemradio", { name: "Light" }).click();
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
  await page.emulateMedia({ colorScheme: "light" });
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
});

test("another open tab follows the selection", async ({ context, page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  const second = await context.newPage();
  await second.emulateMedia({ colorScheme: "light" });
  await second.goto("/");
  await expect(second.locator("html")).not.toHaveClass(/\bdark\b/);
  await page.getByRole("button", { name: "Change color mode" }).click();
  await page.getByRole("menuitemradio", { name: "Dark" }).click();
  await expect(second.locator("html")).toHaveClass(/\bdark\b/);
});
