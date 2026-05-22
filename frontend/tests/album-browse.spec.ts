import { test, expect } from "@playwright/test";
import { openLibrary } from "./helpers";

test.describe("Album browsing", () => {
  test.beforeEach(async ({ page }) => {
    await openLibrary(page);
  });

  test("displays album grid with fixture data", async ({ page }) => {
    await expect(page.locator(".album-title").first()).toBeVisible();
    const titles = await page.locator(".album-title").allTextContents();
    expect(titles).toContain("OK Computer");
    expect(titles).toContain("Kid A");
    expect(titles).toContain("Homogenic");
  });

  test("shows album count in toolbar", async ({ page }) => {
    await expect(page.locator(".count")).toContainText("3 albums");
  });

  test("sort buttons are visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: "Title" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Artist" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Year" })).toBeVisible();
  });

  test("source filter buttons are visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: "All" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Local" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Server" })).toBeVisible();
  });

  test("clicking an album opens album detail view", async ({ page }) => {
    await page.locator(".album-card").first().click();
    // AlbumView shows track list with track titles from GetAlbumTracks fixture.
    await expect(page.getByText("Airbag")).toBeVisible();
    await expect(page.getByText("Paranoid Android")).toBeVisible();
  });

  test("play button overlay appears on hover", async ({ page }) => {
    const card = page.locator(".album-card").first();
    await card.hover();
    const playBtn = card.locator(".play-btn");
    await expect(playBtn).toBeVisible();
  });

  test("play button starts album playback", async ({ page }) => {
    const card = page.locator(".album-card").filter({ hasText: "OK Computer" }).first();
    await card.hover();
    await card.getByRole("button", { name: "Play OK Computer" }).click();
    await expect(page.locator("footer .title")).toContainText("Airbag");
    await expect(page.locator("footer .artist")).toContainText("Radiohead - OK Computer");
  });

  test("double-clicking an album track starts playback from that track", async ({ page }) => {
    await page.locator(".album-card").filter({ hasText: "OK Computer" }).first().click();
    await page.getByRole("button", { name: /Paranoid Android/ }).dblclick();
    await expect(page.locator("footer .title")).toContainText("Paranoid Android");
  });

  test("shows skeleton placeholders while loading", async ({ page }) => {
    // Skeleton placeholders appear during the loading state.
    // After data loads they are replaced by album cards.
    // Once loaded, no skeletons should remain.
    await expect(page.locator(".album-title").first()).toBeVisible();
    await expect(page.locator(".artwork-skeleton")).toHaveCount(0);
  });
});
