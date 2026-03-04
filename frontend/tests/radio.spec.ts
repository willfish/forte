import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.goto('/');
});

// Use the exact nav button text (icon + label) to avoid matching "Radiohead" album cards.
const radioNavButton = (page: any) =>
  page.locator('nav.sidebar button', { hasText: 'Radio' });

test('shows Radio in sidebar navigation', async ({ page }) => {
  await expect(radioNavButton(page)).toBeVisible();
});

test('navigates to radio view', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByRole('heading', { name: 'Radio' })).toBeVisible();
});

test('shows Browse and Favourites tabs', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByRole('button', { name: 'Browse' })).toBeVisible();
  await expect(page.getByRole('button', { name: /Favourites/ })).toBeVisible();
});

test('displays featured stations on Browse tab', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByText('Jazz FM')).toBeVisible();
  await expect(page.getByText('Classical 24')).toBeVisible();
});

test('shows search bar on Browse tab', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByPlaceholder('Search stations by name or genre...')).toBeVisible();
});

test('displays favourite stations on Favourites tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('button', { name: /Favourites/ }).click();
  await expect(page.getByText('Jazz FM')).toBeVisible();
});

test('shows station tags', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByText('jazz', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('classical', { exact: true })).toBeVisible();
});

test('shows play buttons for stations', async ({ page }) => {
  await radioNavButton(page).click();
  const playButtons = page.getByRole('button', { name: /Play / });
  await expect(playButtons.first()).toBeVisible();
});
