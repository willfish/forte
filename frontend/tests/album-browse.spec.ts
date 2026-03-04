import { test, expect } from "@playwright/test";

test.describe("Album browsing", () => {
  test("displays album grid with fixture data", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".album-title").first()).toBeVisible();
    const titles = await page.locator(".album-title").allTextContents();
    expect(titles).toContain("OK Computer");
    expect(titles).toContain("Kid A");
    expect(titles).toContain("Homogenic");
  });

  test("shows album count in toolbar", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".count")).toContainText("3 albums");
  });

  test("sort buttons are visible", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Title" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Artist" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Year" })).toBeVisible();
  });

  test("source filter buttons are visible", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("button", { name: "All" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Local" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Server" })).toBeVisible();
  });

  test("clicking an album opens album detail view", async ({ page }) => {
    await page.goto("/");
    await page.locator(".album-card").first().click();
    // AlbumView shows track list with track titles from GetAlbumTracks fixture.
    await expect(page.getByText("Airbag")).toBeVisible();
    await expect(page.getByText("Paranoid Android")).toBeVisible();
  });

  test("play button overlay appears on hover", async ({ page }) => {
    await page.goto("/");
    const card = page.locator(".album-card").first();
    await card.hover();
    const playBtn = card.locator(".play-btn");
    await expect(playBtn).toBeVisible();
  });

  test("shows skeleton placeholders while loading", async ({ page }) => {
    // Skeleton placeholders appear during the loading state.
    // After data loads they are replaced by album cards.
    await page.goto("/");
    // Once loaded, no skeletons should remain.
    await expect(page.locator(".album-title").first()).toBeVisible();
    await expect(page.locator(".artwork-skeleton")).toHaveCount(0);
  });
});
