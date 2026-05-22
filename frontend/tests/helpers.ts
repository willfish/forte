import { expect, type Page } from "@playwright/test";

export async function enableLibraryMode(page: Page) {
  await page.goto("/");
  await page.locator("nav.sidebar button", { hasText: "Settings" }).click();
  const libraryNav = page.locator("nav.sidebar button", { hasText: "Library" });
  if ((await libraryNav.count()) === 0) {
    await page.getByText("Library mode").click();
  }
  await expect(libraryNav).toBeVisible();
}

export async function openLibrary(page: Page) {
  await enableLibraryMode(page);
  await page.locator("nav.sidebar button", { hasText: "Library" }).click();
  await expect(page.locator("nav.sidebar button", { hasText: "Library" })).toHaveClass(/active/);
}
