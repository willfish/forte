import { test, expect } from "@playwright/test";

test.describe("Search", () => {
  test("search input is visible in library view", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".search-input")).toBeVisible();
  });

  test("typing in search shows results", async ({ page }) => {
    await page.goto("/");
    await page.locator(".search-input").fill("Airbag");
    // Wait for debounce (300ms) + render.
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });
  });

  test("clearing search returns to album grid", async ({ page }) => {
    await page.goto("/");
    await page.locator(".search-input").fill("Airbag");
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });

    await page.locator(".search-clear").click();
    await expect(page.locator(".album-title").first()).toBeVisible();
  });

  test("search is not visible in settings view", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
    await expect(page.locator(".search-input")).not.toBeVisible();
  });
});
