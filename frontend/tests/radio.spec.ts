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

test('keeps the newest radio search results when an older request finishes late', async ({ page }) => {
  await page.evaluate(() => {
    localStorage.setItem('forte.radioSearchDelays', JSON.stringify({ jazz: 800, bbc: 0 }));
  });
  await page.reload();
  await radioNavButton(page).click();

  const search = page.getByPlaceholder('Search stations by name...');
  await search.fill('jazz');
  await page.waitForTimeout(350);
  await search.fill('bbc');

  await expect(page.getByRole('listitem').filter({ hasText: 'BBC World Service' })).toBeVisible();
  await page.waitForTimeout(700);
  await expect(page.getByRole('listitem').filter({ hasText: 'BBC World Service' })).toBeVisible();
  await expect(page.getByRole('listitem').filter({ hasText: 'Adroit Jazz Underground' })).toHaveCount(0);
});

test('pressing slash focuses Browse search', async ({ page }) => {
  await radioNavButton(page).click();
  const search = page.getByPlaceholder('Search stations by name...');

  await page.keyboard.press('/');
  await expect(search).toBeFocused();
});

test('escape from Browse search returns to normal mode without clearing text', async ({ page }) => {
  await radioNavButton(page).click();
  const search = page.getByPlaceholder('Search stations by name...');

  await page.keyboard.press('/');
  await search.fill('jazz');
  await page.keyboard.press('Escape');
  await expect(search).not.toBeFocused();
  await expect(search).toHaveValue('jazz');

  await page.keyboard.press('f');
  await expect(page.locator('.action-hint').first()).toBeVisible();
});

test('pressing slash from any radio tab returns to Browse search', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();

  await page.keyboard.press('/');
  await expect(page.getByRole('tab', { name: 'Browse' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByPlaceholder('Search stations by name...')).toBeFocused();
});

test('h and l move across radio tabs', async ({ page }) => {
  await radioNavButton(page).click();

  await page.keyboard.press('l');
  await expect(page.getByRole('tab', { name: /Favourites/ })).toHaveAttribute('aria-selected', 'true');

  await page.keyboard.press('l');
  await expect(page.locator('.tabs').getByRole('tab', { name: 'History', exact: true })).toHaveAttribute('aria-selected', 'true');

  await page.keyboard.press('l');
  await expect(page.getByRole('tab', { name: /Custom/ })).toHaveAttribute('aria-selected', 'true');

  await page.keyboard.press('h');
  await expect(page.locator('.tabs').getByRole('tab', { name: 'History', exact: true })).toHaveAttribute('aria-selected', 'true');
});

test('pressing f shows button hints and activates the first station', async ({ page }) => {
  await radioNavButton(page).click();
  const station = page.getByRole('listitem').filter({ hasText: 'Radio Paradise Main Mix' });
  await expect(station).toBeVisible();

  await page.keyboard.press('f');
  await expect(page.locator('.action-hint', { hasText: 'aa' })).toBeVisible();
  const visibleViewportButtonCount = await page.locator('.content-area').evaluate((scope) => {
    const viewport = scope.getBoundingClientRect();
    return Array.from(scope.querySelectorAll('button'))
      .filter((button) => {
        const style = window.getComputedStyle(button);
        const rect = button.getBoundingClientRect();
        return !button.hasAttribute('disabled') &&
          style.visibility !== 'hidden' &&
          style.display !== 'none' &&
          rect.width > 0 &&
          rect.height > 0 &&
          rect.bottom >= viewport.top &&
          rect.top <= viewport.bottom &&
          rect.right >= viewport.left &&
          rect.left <= viewport.right;
      }).length;
  });
  await expect(page.locator('.action-hint')).toHaveCount(visibleViewportButtonCount);

  await page.keyboard.press('a');
  await page.keyboard.press('a');
  await expect(station.getByText('Playing')).toBeVisible();
});

test('j and k scroll the current view like vertical arrows', async ({ page }) => {
  await radioNavButton(page).click();
  const initialScrollTop = await page.locator('.content').evaluate((el) => el.scrollTop);

  await page.keyboard.press('j');
  await expect.poll(async () => page.locator('.content').evaluate((el) => el.scrollTop)).toBeGreaterThan(initialScrollTop);

  const scrolledTop = await page.locator('.content').evaluate((el) => el.scrollTop);
  await page.keyboard.press('k');
  await expect.poll(async () => page.locator('.content').evaluate((el) => el.scrollTop)).toBeLessThan(scrolledTop);
});

test('G and gg scroll the current view to bottom and top', async ({ page }) => {
  await radioNavButton(page).click();

  await page.keyboard.press('G');
  await expect.poll(async () => page.locator('.content').evaluate((el) => el.scrollTop)).toBeGreaterThan(0);

  await page.keyboard.press('g');
  await page.keyboard.press('g');
  await expect.poll(async () => page.locator('.content').evaluate((el) => el.scrollTop)).toBe(0);
});

test('displays favourite stations on Favourites tab', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();
  await expect(page.getByText('Jazz FM')).toBeVisible();
});

test('selecting a tag from another radio tab opens Browse for that tag', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();

  await page.getByRole('button', { name: 'jazz', exact: true }).click();
  await expect(page.getByRole('tab', { name: 'Browse' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByText('Tag: jazz')).toBeVisible();
});

test('combines country and tag filters on Browse', async ({ page }) => {
  await radioNavButton(page).click();

  await page.getByRole('button', { name: 'UK', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Clear all filters' })).toHaveCount(0);

  await page.getByRole('button', { name: 'eclectic', exact: true }).click();
  await page.getByRole('button', { name: 'talk', exact: true }).click();

  await expect(page.getByText('Country: UK')).toBeVisible();
  await expect(page.getByText('Tag: eclectic')).toBeVisible();
  await expect(page.getByText('Tag: talk')).toBeVisible();
  await expect(page.getByText('BBC World Service')).toBeVisible();
  await expect(page.getByText('Radio Paradise Main Mix')).toHaveCount(0);

  await page.getByRole('button', { name: 'Remove tag filter talk' }).click();
  await expect(page.getByText('Tag: talk')).toHaveCount(0);
  await expect(page.getByText('Tag: eclectic')).toBeVisible();
  await expect(page.getByText('Country: UK')).toBeVisible();
  await expect(page.getByText('BBC World Service')).toBeVisible();

  await page.getByRole('button', { name: 'Remove country filter UK' }).click();
  await expect(page.getByText('Country: UK')).toHaveCount(0);
});

test('pins favourite stations', async ({ page }) => {
  await radioNavButton(page).click();
  await page.getByRole('tab', { name: /Favourites/ }).click();
  await page
    .locator('.station-list')
    .getByRole('listitem')
    .filter({ hasText: 'Jazz FM' })
    .getByRole('button', { name: 'Pin favourite' })
    .click();
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

  const station = page.getByRole('listitem').filter({ hasText: 'Jazz FM' });
  await station.dblclick();
  await expect(station.getByText('Playing')).toBeVisible();

  await station.dblclick();
  await expect(station.getByText('Paused')).toBeVisible();

  await station.dblclick();
  await expect(station.getByText('Playing')).toBeVisible();
});

test('pressing enter on a focused radio station toggles play and pause', async ({ page }) => {
  await radioNavButton(page).click();

  const station = page.getByRole('listitem').filter({ hasText: 'Jazz FM' });
  const rowAction = station.getByRole('button', { name: 'Play or pause Jazz FM' });
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

  await page.getByRole('button', { name: 'Play Jazz FM' }).click();
  await expect(page.getByRole('alert')).toContainText("Couldn't play Jazz FM.");
  await expect(page.getByRole('alert')).toContainText('stream unavailable');

  await page.getByRole('button', { name: 'Dismiss radio error' }).click();
  await expect(page.getByRole('alert')).toHaveCount(0);
});
