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

  test("volume slider is exposed to assistive technology", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("slider", { name: "Volume" })).toBeVisible();
  });

  test("radio playback backs off high-frequency status polling", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Play Jazz FM" }).click();
    await expect(page.locator("footer")).toContainText("LIVE");

    await page.evaluate(() => {
      (window as any).__wailsCallCounts.clear();
    });
    await page.waitForTimeout(1300);

    const statusCalls = await page.evaluate(() => (window as any).__wailsCallCounts.get(958915679) ?? 0);
    expect(statusCalls).toBeLessThanOrEqual(2);
  });

  test("idle toast polling stays low during unattended playback", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Play Jazz FM" }).click();
    await expect(page.locator("footer")).toContainText("LIVE");

    await page.evaluate(() => {
      (window as any).__wailsCallCounts.clear();
    });
    await page.waitForTimeout(1300);

    const toastCalls = await page.evaluate(() => (window as any).__wailsCallCounts.get(327853480) ?? 0);
    expect(toastCalls).toBeLessThanOrEqual(1);
  });
});
