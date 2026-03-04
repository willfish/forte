import { test, expect } from "@playwright/test";

test.describe("Sidebar navigation", () => {
  test("shows brand name", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".brand")).toContainText("Forte");
  });

  test("library view is active by default", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Library" })).toHaveClass(/active/);
  });

  test("navigates to playlists view", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Playlists" }).click();
    await expect(page.getByRole("button", { name: "Playlists" })).toHaveClass(/active/);
    // PlaylistView should show the fixture playlists.
    await expect(page.getByText("Favourites")).toBeVisible();
    await expect(page.getByText("Chill")).toBeVisible();
  });

  test("navigates to stats view", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Stats" }).click();
    await expect(page.getByRole("button", { name: "Stats" })).toHaveClass(/active/);
    await expect(page.getByText("Listening Stats")).toBeVisible();
  });

  test("navigates to settings view", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
    await expect(page.getByRole("button", { name: "Settings" })).toHaveClass(/active/);
    await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  });

  test("navigates back to library from settings", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
    await page.getByRole("button", { name: "Library" }).click();
    await expect(page.getByRole("button", { name: "Library" })).toHaveClass(/active/);
    await expect(page.locator(".album-title").first()).toBeVisible();
  });
});
