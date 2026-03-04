import { test, expect } from "@playwright/test";

test.describe("Stats view", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Stats" }).click();
  });

  test("shows listening stats heading", async ({ page }) => {
    await expect(page.getByText("Listening Stats")).toBeVisible();
  });

  test("shows period tabs", async ({ page }) => {
    await expect(page.getByRole("button", { name: "7 days" })).toBeVisible();
    await expect(page.getByRole("button", { name: "30 days" })).toBeVisible();
    await expect(page.getByRole("button", { name: "12 months" })).toBeVisible();
    await expect(page.getByRole("button", { name: "All time" })).toBeVisible();
  });

  test("30 days tab is active by default", async ({ page }) => {
    await expect(page.getByRole("button", { name: "30 days" })).toHaveClass(/active/);
  });

  test("shows top artists from fixture data", async ({ page }) => {
    await expect(page.getByText("Top Artists")).toBeVisible();
    await expect(page.getByText("Radiohead").first()).toBeVisible();
    await expect(page.getByText("Bjork").first()).toBeVisible();
  });

  test("shows top albums from fixture data", async ({ page }) => {
    await expect(page.getByText("Top Albums")).toBeVisible();
    await expect(page.getByText("OK Computer").first()).toBeVisible();
  });

  test("shows top tracks from fixture data", async ({ page }) => {
    await expect(page.getByText("Top Tracks")).toBeVisible();
    await expect(page.getByText("Airbag").first()).toBeVisible();
  });

  test("shows recently played section", async ({ page }) => {
    await expect(page.getByText("Recently Played")).toBeVisible();
  });

  test("switching period tab updates active state", async ({ page }) => {
    await page.getByRole("button", { name: "All time" }).click();
    await expect(page.getByRole("button", { name: "All time" })).toHaveClass(/active/);
    await expect(page.getByRole("button", { name: "30 days" })).not.toHaveClass(/active/);
  });
});
