import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(__dirname, '..');
const repoDir = path.resolve(frontendDir, '..');
const outputPath = path.join(repoDir, 'docs', 'demo.png');
const port = process.env.FORTE_DEMO_PORT || '5174';
const url = `http://127.0.0.1:${port}/`;
const browserCandidates = [
  process.env.CHROME_PATH,
  '/home/william/.nix-profile/bin/google-chrome-stable',
  '/usr/bin/google-chrome-stable',
  '/usr/bin/google-chrome',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
].filter(Boolean);

function startServer() {
  const viteBin = path.join(frontendDir, 'node_modules', 'vite', 'bin', 'vite.js');
  return spawn(
    process.execPath,
    [viteBin, '--config', 'vite.config.test.ts', '--host', '127.0.0.1', '--port', port, '--strictPort'],
    {
      cwd: frontendDir,
      stdio: ['ignore', 'pipe', 'pipe'],
      env: { ...process.env, FORCE_COLOR: '0', NO_COLOR: '1' },
    },
  );
}

async function waitForServer(processHandle) {
  const deadline = Date.now() + 30_000;
  let lastError;

  while (Date.now() < deadline) {
    if (processHandle.exitCode !== null) {
      throw new Error(`Vite exited before serving ${url}`);
    }

    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch (error) {
      lastError = error;
    }

    await new Promise((resolve) => setTimeout(resolve, 250));
  }

  throw new Error(`Timed out waiting for ${url}: ${lastError?.message || 'no response'}`);
}

async function main() {
  await mkdir(path.dirname(outputPath), { recursive: true });

  const server = startServer();
  server.stdout.on('data', (chunk) => process.stdout.write(chunk));
  server.stderr.on('data', (chunk) => process.stderr.write(chunk));

  try {
    await waitForServer(server);

    const executablePath = browserCandidates.find((candidate) => existsSync(candidate));
    const browser = await chromium.launch(executablePath ? { executablePath } : undefined);
    const page = await browser.newPage({ viewport: { width: 900, height: 1200 }, deviceScaleFactor: 1 });

    await page.goto(url);
    await page.locator('.station-card').first().waitFor({ state: 'visible' });
    await page.getByRole('button', { name: /Play / }).first().click();
    await page.locator('.track-info').getByText('Ishq - Iqqoa').waitFor({ state: 'visible' });
    await page.screenshot({ path: outputPath, fullPage: false });

    await browser.close();
    console.log(`Wrote ${path.relative(repoDir, outputPath)}`);
  } finally {
    server.kill('SIGTERM');
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
