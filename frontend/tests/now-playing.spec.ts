import { test, expect } from "@playwright/test";

test.describe("Now playing view", () => {
  test("artwork in now-playing bar is clickable", async ({ page }) => {
    await page.goto("/");
    // The artwork button should exist in the now-playing bar.
    const artworkBtn = page.locator("footer .artwork-btn");
    await expect(artworkBtn).toBeVisible();
  });

  test("clicking artwork expands now-playing view", async ({ page }) => {
    await page.goto("/");
    await page.locator("footer .artwork-btn").click();
    // The expanded view should show.
    await expect(page.locator(".npv-backdrop")).toBeVisible();
    // Close button should be visible.
    await expect(page.locator(".npv-close")).toBeVisible();
  });

  test("close button collapses now-playing view", async ({ page }) => {
    await page.goto("/");
    await page.locator("footer .artwork-btn").click();
    await expect(page.locator(".npv-backdrop")).toBeVisible();
    await page.locator(".npv-close").click();
    await expect(page.locator(".npv-backdrop")).toHaveCount(0);
  });

  test("escape key collapses now-playing view", async ({ page }) => {
    await page.goto("/");
    await page.locator("footer .artwork-btn").click();
    await expect(page.locator(".npv-backdrop")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator(".npv-backdrop")).toHaveCount(0);
  });
});
