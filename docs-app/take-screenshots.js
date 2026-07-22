const { chromium } = require('playwright');
const path = require('path');

const BASE_URL = 'https://app.securelens.ai';
const SCREENSHOT_DIR = path.join(__dirname, 'public', 'screenshots');

const pages = [
  { path: '/dashboard', name: 'dashboard.png' },
  { path: '/data-sources', name: 'data-sources.png' },
  { path: '/classification', name: 'classification.png' },
  { path: '/labels', name: 'sensitivity-labels.png' },
  { path: '/quality', name: 'data-quality.png' },
  { path: '/rot', name: 'rot-detection.png' },
  { path: '/governance', name: 'governance-policies.png' },
  { path: '/gate', name: 'ai-gate.png' },
  { path: '/privacy', name: 'privacy-center.png' },
  { path: '/advisor', name: 'compliance-advisor.png' },
  { path: '/audit', name: 'audit-trail.png' },
  { path: '/observability', name: 'observability.png' },
  { path: '/integrations', name: 'integrations.png' },
  { path: '/jobs', name: 'scheduled-jobs.png' },
  { path: '/remediation', name: 'remediation.png' },
  { path: '/reports', name: 'reports.png' },
  { path: '/feedback', name: 'feedback.png' },
  { path: '/documents', name: 'documents.png' },
  { path: '/settings', name: 'settings.png' },
  { path: '/data-map', name: 'data-map.png' },
];

async function takeScreenshots() {
  console.log('Launching browser...');
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
  });
  const page = await context.newPage();

  try {
    console.log('Navigating to login page...');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });

    console.log('Logging in...');
    await page.fill('input[type="email"]', 'admin@securelens.local');
    await page.fill('input[type="password"]', 'SecureLens@2026!');
    await page.click('button[type="submit"]');

    console.log('Waiting for dashboard...');
    await page.waitForURL('**/dashboard', { timeout: 30000 });
    await page.waitForLoadState('networkidle');

    console.log('Login successful! Taking screenshots...\n');

    for (const { path: pagePath, name } of pages) {
      const url = `${BASE_URL}${pagePath}`;
      const screenshotPath = path.join(SCREENSHOT_DIR, name);

      console.log(`Navigating to ${pagePath}...`);
      await page.goto(url, { waitUntil: 'networkidle' });
      await page.waitForTimeout(1500);

      console.log(`  Taking screenshot: ${name}`);
      await page.screenshot({
        path: screenshotPath,
        fullPage: false,
      });
      console.log(`  Saved: ${screenshotPath}\n`);
    }

    console.log('All screenshots taken successfully!');
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

takeScreenshots();
