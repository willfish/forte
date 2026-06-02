import { test, expect } from "@playwright/test";
import { enableLibraryMode } from "./helpers";

test.describe("Settings", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
  });

  test("shows theme options", async ({ page }) => {
    await expect(page.getByText("Dark", { exact: true })).toBeVisible();
    await expect(page.getByText("Light", { exact: true })).toBeVisible();
    await expect(page.getByText("Green", { exact: true })).toBeVisible();
    await expect(page.getByText("Blue", { exact: true })).toBeVisible();
    await expect(page.getByText("Financial Times", { exact: true })).toBeVisible();
  });

  test("combines theme mode and colour", async ({ page }) => {
    await page.locator('input[name="theme-mode"][value="light"]').check();
    await page.locator('input[name="theme-colour"][value="blue"]').check();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "blue-light");
    await expect(page.locator(".brand-mark rect")).toHaveCSS("fill", "rgb(243, 247, 252)");
    await expect(page.locator(".brand-mark path")).toHaveCSS("stroke", "rgb(37, 99, 235)");

    await page.locator('input[name="theme-mode"][value="dark"]').check();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "blue-dark");
    await expect(page.locator(".brand-mark rect")).toHaveCSS("fill", "rgb(16, 27, 43)");

    await page.locator('input[name="theme-colour"][value="financial-times"]').check();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "financial-times-dark");
    await expect(page.locator(".brand-mark path")).toHaveCSS("stroke", "rgb(199, 66, 106)");
  });

  test("configures theme transparency", async ({ page }) => {
    const html = page.locator("html");
    const opacity = page.getByLabel("Theme opacity");

    await expect(page.getByLabel("Transparent theme")).not.toBeChecked();
    await expect(opacity).toHaveValue("0.8");
    await expect(opacity).toBeDisabled();
    await expect(html).toHaveAttribute("data-theme-transparency", "off");
    await expect(html).toHaveCSS("--theme-opacity", "1");

    await page.getByLabel("Transparent theme").check();
    await expect(opacity).toBeEnabled();
    await expect(html).toHaveAttribute("data-theme-transparency", "on");
    await expect(html).toHaveCSS("--theme-opacity", "0.8");
    await expect(page.locator("body")).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
    await expect(page.locator(".shell")).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");

    await opacity.fill("0.65");
    await expect(html).toHaveCSS("--theme-opacity", "0.65");

    await page.reload();
    await page.getByRole("button", { name: "Settings" }).click();
    await expect(page.getByLabel("Transparent theme")).toBeChecked();
    await expect(page.getByLabel("Theme opacity")).toHaveValue("0.65");
    await expect(html).toHaveCSS("--theme-opacity", "0.65");
  });

  test("shows servers section", async ({ page }) => {
    await enableLibraryMode(page);
    await expect(page.getByRole("heading", { name: "Servers" })).toBeVisible();
    await expect(page.getByText("No servers configured")).toBeVisible();
  });

  test("add server button opens form", async ({ page }) => {
    await enableLibraryMode(page);
    await page.getByRole("button", { name: "Add server" }).click();
    await expect(page.locator("#srv-name")).toBeVisible();
    await expect(page.locator("#srv-url")).toBeVisible();
    await expect(page.locator("#srv-user")).toBeVisible();
    await expect(page.getByText("Subsonic")).toBeVisible();
    await expect(page.getByText("Jellyfin")).toBeVisible();
  });

  test("cancel closes server form", async ({ page }) => {
    await enableLibraryMode(page);
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
