import { expect, test } from "@playwright/test";
import { readFile, readdir } from "node:fs/promises";

const origin = "http://cli.invalid";
const output = new URL("../../../../internal/kden/auth/pages/generated/", import.meta.url);
const csp =
  "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'";

test.beforeEach(async ({ context }) => {
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== origin || !["/success.html", "/failure.html"].includes(url.pathname)) {
      await route.abort();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: "text/html; charset=utf-8",
      headers: { "Content-Security-Policy": csp, "Cache-Control": "no-store" },
      body: await readFile(new URL(`.${url.pathname}`, output)),
    });
  });
});

const results = [
  { name: "success", heading: "Login successful" },
  { name: "failure", heading: "Login failed" },
] as const;

test("generated documents have no external assets or client bundle", async () => {
  expect((await readdir(output)).toSorted()).toEqual(["failure.html", "success.html"]);
  for (const result of results) {
    const html = await readFile(new URL(`${result.name}.html`, output), "utf8");
    expect(html).toContain("<style>");
    expect(html).toContain("<svg ");
    const links = html.match(/<link\b[^>]*>/g) ?? [];
    for (const link of links) {
      expect(link).toMatch(
        /^<link href="\.\/_app\/immutable\/assets\/[^" ]+\.css" rel="stylesheet" disabled media="\(max-width: 0\)">$/,
      );
    }
    expect(html.replaceAll(/<link\b[^>]*>/g, "")).not.toMatch(
      /<(?:img|link)\b|<script\b[^>]*\bsrc\s*=|_app\//,
    );
  }
});

for (const result of results) {
  for (const colorScheme of ["light", "dark"] as const) {
    test(`${result.name} ${colorScheme} desktop`, async ({ page }) => {
      await page.setViewportSize({ width: 1280, height: 800 });
      await page.emulateMedia({ colorScheme, reducedMotion: "reduce" });
      const requests: string[] = [];
      const violations: string[] = [];
      page.on("request", (request) => requests.push(request.url()));
      page.on("console", (message) => {
        if (message.type() === "error") violations.push(message.text());
      });
      await page.goto(`/${result.name}.html`);

      await expect(page).toHaveTitle(`Konfidence – ${result.heading}`);
      await expect(page.getByRole("heading", { level: 1 })).toHaveText(result.heading);
      await expect(page.getByRole("img", { name: "Konfidence" })).toBeVisible();
      await expect(
        page.getByText("You can close this window and return to your terminal."),
      ).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("data-theme", "konfidence");
      await expect(page.locator("html")).toHaveClass(colorScheme === "dark" ? "dark" : "");
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(1280);
      await expect(page).toHaveScreenshot(`${result.name}-${colorScheme}-desktop.png`, {
        fullPage: true,
      });
      expect(requests).toEqual([`${origin}/${result.name}.html`]);
      expect(violations).toEqual([]);
    });
  }

  test(`${result.name} remains readable when closing is blocked`, async ({ page }) => {
    await page.clock.install();
    await page.addInitScript(() => {
      window.close = () => {
        document.documentElement.dataset.closeAttempted = "true";
      };
    });
    await page.goto(`/${result.name}.html`);
    await page.clock.fastForward(5000);
    await expect(page.locator("html")).toHaveAttribute("data-close-attempted", "true");
    await expect(
      page.getByText("You can close this window and return to your terminal."),
    ).toBeVisible();
  });
}

test.describe("without JavaScript", () => {
  test.use({ javaScriptEnabled: false });
  for (const result of results) {
    test(`${result.name} renders without JavaScript`, async ({ page }) => {
      await page.goto(`/${result.name}.html`);
      await expect(page.getByRole("heading", { level: 1 })).toHaveText(result.heading);
      await expect(
        page.getByText("You can close this window and return to your terminal."),
      ).toBeVisible();
    });
  }
});

test("follows changes to the system appearance", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/success.html");
  await expect(page.locator("html")).not.toHaveClass("dark");
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveClass("dark");
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).not.toHaveClass("dark");
});
