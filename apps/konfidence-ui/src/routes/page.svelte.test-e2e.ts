import { expect, test } from "@playwright/test";
import { signIn } from "../../e2e/helpers";

const HTTP_OK = 200;

test("redirects unauthenticated visitors to the login page", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL(/\/login\?returnTo=%2F$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Sign in to Konfidence");
});

test("redirects an authenticated visitor with multiple projects to the chooser", async ({
  page,
}) => {
  await signIn(page, "/projects");

  await expect(page.getByRole("heading", { name: "Choose a project" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence");
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
});

test("serves the SPA document for a deep link", async ({ request }) => {
  const response = await request.get("/projects/example/landscape");

  expect(response.status()).toBe(HTTP_OK);
  expect(await response.text()).toContain('<div style="display: contents">');
});

test("system tracks OS changes but explicit dark stays dark", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/login");
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("mode-watcher-mode")))
    .toBe("system");
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).not.toHaveClass(/\bdark\b/);

  await page.evaluate(() => localStorage.setItem("mode-watcher-mode", "dark"));
  await page.reload();
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await page.emulateMedia({ colorScheme: "dark" });
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("mode-watcher-mode")))
    .toBe("dark");
  await page.goto("/login");
  await expect(page.locator("html")).toHaveClass(/\bdark\b/);
});
