import { test, expect } from '@playwright/test';
import { radioFixtures } from './fixtures/radio-stations';

test.beforeEach(async ({ page }) => {
  await page.goto('/');
});

const radioNavButton = (page: import('@playwright/test').Page) =>
  page.locator('nav.sidebar button', { hasText: 'Radio' });

test('opens station detail with homepage and stream links', async ({ page }) => {
  await radioNavButton(page).click();

  await page.getByRole('button', { name: 'Jazz FM', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Jazz FM', level: 1 })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Links' })).toBeVisible();

  const website = page.getByRole('link', { name: 'https://www.jazzfm.com/' });
  await expect(website).toHaveAttribute('href', 'https://www.jazzfm.com/');
  await expect(website).toHaveAttribute('rel', 'noopener noreferrer');

  const stream = page.getByRole('link', { name: 'https://stream.example.com/jazz-fm' });
  await expect(stream).toHaveAttribute('href', 'https://stream.example.com/jazz-fm');
  await expect(stream).toHaveAttribute('rel', 'noopener noreferrer');
});

test('station detail shows now playing track when station is active', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('button', { name: 'Play Jazz FM' }).click();
  await page.locator('.station-list').getByRole('button', { name: 'Jazz FM', exact: true }).click();
  await expect(page.getByText("Now playing: Live at Ronnie Scott's")).toBeVisible();
});

test('station detail can pin a favourite', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('button', { name: 'Jazz FM', exact: true }).click();
  await page.getByRole('button', { name: 'Add favourite' }).click();
  await page.getByRole('button', { name: 'Pin' }).click();
  await expect(page.getByRole('button', { name: 'Unpin' })).toBeVisible();
});

test('opens SomaFM station detail from browse filter', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('button', { name: 'SomaFM' }).click();
  await page.getByRole('button', { name: 'SomaFM Mission Control', exact: true }).click();

  await expect(page.getByRole('heading', { name: 'SomaFM Mission Control', level: 1 })).toBeVisible();
  const website = page.getByRole('link', { name: 'https://somafm.com/missioncontrol/' });
  await expect(website).toHaveAttribute('href', 'https://somafm.com/missioncontrol/');

  // The station art must be a proxied data: URI (not the raw external favicon URL from backend).
  // This ensures it renders in the Wails webview (see RadioView lists which do proxy).
  const art = page.locator(".station-art");
  await expect(art).toBeVisible();
  await expect(art).toHaveAttribute("src", /^data:/);
});

test('custom station detail shows derived website from stream host', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Custom/ }).click();

  await page.getByRole('textbox', { name: 'Station name' }).fill(radioFixtures.custom.name);
  await page.getByRole('textbox', { name: 'Stream URL' }).fill(radioFixtures.custom.streamUrl);
  await page.getByRole('button', { name: 'Add Station' }).click();
  await expect(
    page.locator('#radio-panel-custom').getByRole('button', { name: radioFixtures.custom.name, exact: true })
  ).toBeVisible();

  await page.locator('#radio-panel-custom').getByRole('button', { name: radioFixtures.custom.name, exact: true }).click();

  const websiteRow = page.locator('.station-view .link-row').filter({ hasText: 'Website' });
  await expect(websiteRow.getByRole('link')).toHaveAttribute('href', radioFixtures.custom.derivedHomepage);
});

test('opens favourite-only station detail from favourites tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();
  await page.getByRole('button', { name: radioFixtures.favouriteOnly.name, exact: true }).click();

  await expect(page.getByRole('heading', { name: radioFixtures.favouriteOnly.name, level: 1 })).toBeVisible();
  await expect(page.getByRole('link', { name: radioFixtures.favouriteOnly.homepage })).toHaveAttribute(
    'href',
    radioFixtures.favouriteOnly.homepage
  );
});

test('opens history-only station detail from history tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /History/ }).click();
  await page.getByRole('button', { name: radioFixtures.historyOnly.name, exact: true }).click();

  await expect(page.getByRole('heading', { name: radioFixtures.historyOnly.name, level: 1 })).toBeVisible();
  await expect(page.getByRole('link', { name: radioFixtures.historyOnly.homepage })).toHaveAttribute(
    'href',
    radioFixtures.historyOnly.homepage
  );
});
