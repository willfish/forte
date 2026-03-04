import { test, expect } from "@playwright/test";

test.describe("Settings", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
  });

  test("shows theme options", async ({ page }) => {
    await expect(page.getByText("Dark", { exact: true })).toBeVisible();
    await expect(page.getByText("Light", { exact: true })).toBeVisible();
    await expect(page.getByText("System", { exact: true })).toBeVisible();
  });

  test("shows servers section", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Servers" })).toBeVisible();
    await expect(page.getByText("No servers configured")).toBeVisible();
  });

  test("add server button opens form", async ({ page }) => {
    await page.getByRole("button", { name: "Add server" }).click();
    await expect(page.locator("#srv-name")).toBeVisible();
    await expect(page.locator("#srv-url")).toBeVisible();
    await expect(page.locator("#srv-user")).toBeVisible();
    await expect(page.getByText("Subsonic")).toBeVisible();
    await expect(page.getByText("Jellyfin")).toBeVisible();
  });

  test("cancel closes server form", async ({ page }) => {
    await page.getByRole("button", { name: "Add server" }).click();
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByText("No servers configured")).toBeVisible();
  });

  test("shows Last.fm section", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Last.fm" })).toBeVisible();
    // No API key configured - should show the API key form.
    await expect(page.locator("#lfm-key")).toBeVisible();
  });

  test("shows ListenBrainz section", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "ListenBrainz" })).toBeVisible();
    await expect(page.locator("#lb-token")).toBeVisible();
  });
});
