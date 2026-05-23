import { test, expect } from "@playwright/test";
import { openLibrary } from "./helpers";

test.describe("Search", () => {
  test("search input is visible in library view", async ({ page }) => {
    await openLibrary(page);
    await expect(page.locator(".search-input")).toBeVisible();
  });

  test("typing in search shows results", async ({ page }) => {
    await openLibrary(page);
    await page.locator(".search-input").fill("Airbag");
    // Wait for debounce (300ms) + render.
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });
  });

  test("clearing search returns to album grid", async ({ page }) => {
    await openLibrary(page);
    await page.locator(".search-input").fill("Airbag");
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });

    await page.locator(".search-clear").click();
    await expect(page.locator(".album-title").first()).toBeVisible();
  });

  test("escape blurs search without clearing the query", async ({ page }) => {
    await openLibrary(page);
    const search = page.locator(".search-input");
    await search.fill("Airbag");
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });

    await page.keyboard.press("Escape");
    await expect(search).not.toBeFocused();
    await expect(search).toHaveValue("Airbag");
    await expect(page.getByText("Airbag").first()).toBeVisible();

    await page.keyboard.press("f");
    await expect(page.locator(".action-hint").first()).toBeVisible();
  });

  test("selecting an artist tag opens radio Browse for that tag", async ({ page }) => {
    await openLibrary(page);
    await page.locator(".search-input").fill("Airbag");
    await expect(page.getByText("Airbag").first()).toBeVisible({ timeout: 2000 });

    await page.getByText("Radiohead", { exact: true }).click();
    await page.getByRole("button", { name: "alternative" }).click();

    await expect(page.getByRole("heading", { name: "Radio" })).toBeVisible();
    await expect(page.getByRole("tab", { name: "Browse" })).toHaveAttribute("aria-selected", "true");
    await expect(page.getByText("Tag: alternative")).toBeVisible();
  });

  test("search is not visible in settings view", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
    await expect(page.locator(".search-input")).not.toBeVisible();
  });
});
