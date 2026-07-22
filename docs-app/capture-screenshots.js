const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const BASE_URL = 'https://app.securelens.ai';
const SCREENSHOTS_DIR = path.join(__dirname, 'public', 'screenshots');

const CREDENTIALS = {
  email: 'admin@securelens.local',
  password: 'SecureLens@2026!'
};

const PAGES_TO_CAPTURE = [
  { name: 'dashboard', path: '/dashboard', waitFor: '.dashboard-content' },
  { name: 'data-sources', path: '/data-sources', waitFor: '.data-sources-list' },
  { name: 'classification', path: '/classification', waitFor: '.classification-table' },
  { name: 'data-map', path: '/data-map', waitFor: '.data-map-view' },
  { name: 'rot', path: '/rot', waitFor: '.rot-analysis' },
  { name: 'labels', path: '/labels', waitFor: '.labels-list' },
  { name: 'quality', path: '/quality', waitFor: '.quality-dashboard' },
  { name: 'governance', path: '/governance', waitFor: '.policies-list' },
  { name: 'ai-gate', path: '/ai-gate', waitFor: '.ai-gate-dashboard' },
  { name: 'privacy', path: '/privacy', waitFor: '.privacy-center' },
  { name: 'advisor', path: '/advisor', waitFor: '.compliance-advisor' },
  { name: 'audit', path: '/audit', waitFor: '.audit-log' },
  { name: 'observability', path: '/observability', waitFor: '.observability-dashboard' },
  { name: 'integrations', path: '/integrations', waitFor: '.integrations-list' },
  { name: 'jobs', path: '/jobs', waitFor: '.jobs-list' },
  { name: 'remediation', path: '/remediation', waitFor: '.remediation-dashboard' },
  { name: 'reports', path: '/reports', waitFor: '.reports-list' },
  { name: 'feedback', path: '/feedback', waitFor: '.feedback-dashboard' },
  { name: 'documents', path: '/documents', waitFor: '.documents-list' },
  { name: 'settings', path: '/settings', waitFor: '.settings-page' },
];

async function captureScreenshots() {
  // Ensure screenshots directory exists
  if (!fs.existsSync(SCREENSHOTS_DIR)) {
    fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
  }

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    deviceScaleFactor: 2,
  });
  const page = await context.newPage();

  try {
    // Login
    console.log('Logging in...');
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="email"]', CREDENTIALS.email);
    await page.fill('input[name="password"]', CREDENTIALS.password);
    await page.click('button[type="submit"]');
    
    // Wait for login to complete
    await page.waitForURL('**/dashboard', { timeout: 30000 });
    console.log('Login successful!');

    // Capture each page
    for (const pageConfig of PAGES_TO_CAPTURE) {
      try {
        console.log(`Capturing ${pageConfig.name}...`);
        await page.goto(`${BASE_URL}${pageConfig.path}`, { waitUntil: 'networkidle' });
        
        // Wait for specific element if specified
        if (pageConfig.waitFor) {
          try {
            await page.waitForSelector(pageConfig.waitFor, { timeout: 10000 });
          } catch (e) {
            console.log(`  Warning: Could not find ${pageConfig.waitFor}, capturing anyway`);
          }
        }
        
        // Small delay for animations
        await page.waitForTimeout(1000);
        
        // Take screenshot
        const screenshotPath = path.join(SCREENSHOTS_DIR, `${pageConfig.name}.png`);
        await page.screenshot({ path: screenshotPath, fullPage: false });
        console.log(`  Saved: ${screenshotPath}`);
      } catch (error) {
        console.error(`  Error capturing ${pageConfig.name}: ${error.message}`);
      }
    }

    console.log('\nScreenshot capture complete!');
  } catch (error) {
    console.error('Error during screenshot capture:', error);
  } finally {
    await browser.close();
  }
}

// Run if called directly
if (require.main === module) {
  captureScreenshots().catch(console.error);
}

module.exports = { captureScreenshots };
