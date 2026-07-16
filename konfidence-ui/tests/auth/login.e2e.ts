import { expect, test } from "@playwright/test";

test("signs in through Dex and logs out of the API session", async ({ page }) => {
  await page.goto("/landscape");
  await expect(page).toHaveURL(/localhost:15556\/dex\/auth/);

  await page.getByRole("link", { name: "Log in with Email" }).click();
  await page.getByLabel("Email Address").fill("alice@example.com");
  await page.getByLabel("Password").fill("password");
  await page.getByRole("button", { name: "Login" }).click();

  await expect(page).toHaveURL("http://127.0.0.1:4173/landscape");
  const accountMenu = page.getByLabel("Open account menu for alice");
  await expect(accountMenu).toBeVisible();
  await expect(page.getByLabel("Stage dev")).toBeVisible();
  await expect(page.getByLabel("Stage staging")).toBeVisible();
  await expect(page.getByLabel("Stage prod")).toBeVisible();
  await expect(page.getByLabel("Stage canary")).toBeVisible();
  await accountMenu.click();
  await expect(page.getByText("alice", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Sign Out" }).click();
  await expect(page).toHaveURL(/localhost:15556\/dex\/auth/);

  const cookies = await page.context().cookies();
  expect(cookies.some((cookie) => cookie.name === "konfidence.sid")).toBe(false);
  expect(cookies.some((cookie) => cookie.name === "kden_session")).toBe(false);
});

test("explains when the signed-in user cannot access stages", async ({ page }) => {
  await page.goto("/landscape");
  await page.getByRole("link", { name: "Log in with Email" }).click();
  await page.getByLabel("Email Address").fill("bob@example.com");
  await page.getByLabel("Password").fill("password");
  await page.getByRole("button", { name: "Login" }).click();

  await expect(page).toHaveURL("http://127.0.0.1:4173/landscape");
  await expect(page.getByRole("heading", { name: "Access denied" })).toBeVisible();
  await expect(page.getByText(/does not have permission to view stages/)).toBeVisible();
  await expect(page.getByText("Error 403")).toBeVisible();
});

test("explains that the Dex Mock identity cannot access stages", async ({ page }) => {
  await page.goto("/landscape");
  await page.getByRole("link", { name: "Log in with Mock" }).click();

  await expect(page).toHaveURL("http://127.0.0.1:4173/landscape");
  await expect(page.getByRole("button", { name: /^Open account menu for / })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Access denied" })).toBeVisible();
  await expect(page.getByText(/does not have permission to view stages/)).toBeVisible();
  await expect(page.getByText("Error 403")).toBeVisible();
});
