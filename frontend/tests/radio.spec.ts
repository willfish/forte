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

test('opens on radio and hides library-first navigation by default', async ({ page }) => {
  await expect(page.getByRole('heading', { name: 'Radio' })).toBeVisible();
  await expect(page.locator('nav.sidebar button', { hasText: 'History' })).toHaveCount(0);
  await expect(page.locator('nav.sidebar button', { hasText: 'Library' })).toHaveCount(0);
  await expect(page.locator('nav.sidebar button', { hasText: 'Playlists' })).toHaveCount(0);
  await expect(page.locator('nav.sidebar button', { hasText: 'Stats' })).toHaveCount(0);
});

test('navigates to radio view', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByRole('heading', { name: 'Radio' })).toBeVisible();
});

test('shows radio-focused tabs', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByRole('tab', { name: 'Browse' })).toBeVisible();
  await expect(page.getByRole('tab', { name: /Favourites/ })).toBeVisible();
  await expect(page.getByRole('tab', { name: /Custom/ })).toBeVisible();
  await expect(page.locator('.tabs').getByRole('tab', { name: 'History', exact: true })).toBeVisible();
});

test('displays featured stations on Browse tab', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByText('Jazz FM')).toBeVisible();
  await expect(page.getByText('Classical 24')).toBeVisible();
});

test('shows search bar on Browse tab', async ({ page }) => {
  await radioNavButton(page).click();
  await expect(page.getByPlaceholder('Search stations by name...')).toBeVisible();
});

test('displays favourite stations on Favourites tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();
  await expect(page.getByText('Jazz FM')).toBeVisible();
});

test('pins favourite stations', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();
  await page.getByRole('button', { name: 'Pin favourite' }).click();
  await expect(page.getByText('Pinned')).toBeVisible();
});

test('adds and plays a custom station', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Custom/ }).click();
  await page.getByLabel('Station name').fill('My Stream');
  await page.getByLabel('Stream URL').fill('https://stream.example.com/mine');
  await page.getByLabel('Tags').fill('ambient');
  await page.getByRole('button', { name: 'Add Station' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'My Stream' })).toBeVisible();

  await page.locator('.tabs').getByRole('tab', { name: 'History', exact: true }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'My Stream' })).toBeVisible();
});

test('shows radio history from the Radio tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.locator('.tabs').getByRole('tab', { name: 'History', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Radio' })).toBeVisible();
  await expect(page.getByText('Classical 24')).toBeVisible();
  await expect(page.getByText('Morning Concert')).toBeVisible();
});

test('library mode preference restores library navigation', async ({ page }) => {
  await page.locator('nav.sidebar button', { hasText: 'Settings' }).click();
  await page.getByText('Library mode').click();
  await expect(page.locator('nav.sidebar button', { hasText: 'Library' })).toBeVisible();
  await expect(page.locator('nav.sidebar button', { hasText: 'Playlists' })).toBeVisible();
  await expect(page.locator('nav.sidebar button', { hasText: 'Stats' })).toBeVisible();
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

test('double-clicking the current radio station toggles play and pause', async ({ page }) => {
  await radioNavButton(page).click();

  const station = page.getByRole('listitem').filter({ hasText: 'Ishq - Iqqoa' });
  await station.dblclick();
  await expect(station.getByText('Playing')).toBeVisible();

  await station.dblclick();
  await expect(station.getByText('Paused')).toBeVisible();

  await station.dblclick();
  await expect(station.getByText('Playing')).toBeVisible();
});

test('pressing enter on a focused radio station toggles play and pause', async ({ page }) => {
  await radioNavButton(page).click();

  const station = page.getByRole('listitem').filter({ hasText: 'Ishq - Iqqoa' });
  const rowAction = station.getByRole('button', { name: 'Play or pause Ishq - Iqqoa' });
  await rowAction.focus();
  await page.keyboard.press('Enter');
  await expect(station.getByText('Playing')).toBeVisible();

  await page.keyboard.press('Enter');
  await expect(station.getByText('Paused')).toBeVisible();
});

test('shows a recoverable error when a station cannot play', async ({ page }) => {
  await page.evaluate(() => localStorage.setItem('forte.failPlayRadioStation', 'true'));
  await page.reload();
  await radioNavButton(page).click();

  await page.getByRole('button', { name: 'Play Ishq - Iqqoa' }).click();
  await expect(page.getByRole('alert')).toContainText("Couldn't play Ishq - Iqqoa.");
  await expect(page.getByRole('alert')).toContainText('stream unavailable');

  await page.getByRole('button', { name: 'Dismiss radio error' }).click();
  await expect(page.getByRole('alert')).toHaveCount(0);
});
