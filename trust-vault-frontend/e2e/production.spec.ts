import { test, expect, Page } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'

const API_BASE = process.env.E2E_ENV === 'production' 
  ? 'https://trust-vault-api.oortfy.com/api/v1'
  : 'http://localhost:8080/api/v1'

const ADMIN_EMAIL = process.env.SUPERADMIN_EMAIL || 'admin@trustvault.local'
const ADMIN_PASSWORD = process.env.SUPERADMIN_PASSWORD || 'TrustVault@2026!'

const SCREENSHOTS_DIR = path.join(__dirname, '..', 'screenshots')

let authToken: string

// Ensure screenshots directory exists
if (!fs.existsSync(SCREENSHOTS_DIR)) {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true })
}

async function takeScreenshot(page: Page, name: string) {
  const screenshotPath = path.join(SCREENSHOTS_DIR, `${name}.png`)
  await page.screenshot({ path: screenshotPath, fullPage: true })
  console.log(`Screenshot saved: ${screenshotPath}`)
  return screenshotPath
}

async function login(page: Page): Promise<string> {
  if (authToken) return authToken
  
  const response = await page.request.post(`${API_BASE}/auth/login`, {
    data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
  })
  
  if (response.ok()) {
    const data = await response.json()
    authToken = data.access_token
    return authToken
  }
  
  throw new Error(`Failed to login via API: ${response.status()}`)
}

async function apiRequest(page: Page, method: string, endpoint: string, data?: any) {
  const token = await login(page)
  const options: any = {
    headers: { Authorization: `Bearer ${token}` }
  }
  if (data) options.data = data
  
  const url = `${API_BASE}${endpoint}`
  if (method === 'GET') return page.request.get(url, options)
  if (method === 'POST') return page.request.post(url, options)
  if (method === 'PUT') return page.request.put(url, options)
  if (method === 'DELETE') return page.request.delete(url, options)
}

async function loginViaUI(page: Page) {
  await page.goto('/login')
  await page.waitForLoadState('networkidle')
  await page.fill('input[name="email"]', ADMIN_EMAIL)
  await page.fill('input[name="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL(/.*dashboard/, { timeout: 30000 })
}

test.describe('TrustVault Production E2E Tests', () => {
  
  test.describe('1. Authentication', () => {
    test('1.1 Login with superadmin credentials', async ({ page }) => {
      await page.goto('/login')
      await page.waitForLoadState('networkidle')
      await takeScreenshot(page, '01-login-page')
      
      await page.fill('input[name="email"]', ADMIN_EMAIL)
      await page.fill('input[name="password"]', ADMIN_PASSWORD)
      await page.click('button[type="submit"]')
      
      await expect(page).toHaveURL(/.*dashboard/, { timeout: 30000 })
      await takeScreenshot(page, '02-dashboard-after-login')
    })

    test('1.2 Session persistence after refresh', async ({ page }) => {
      await loginViaUI(page)
      await page.reload()
      await page.waitForLoadState('networkidle')
      
      const url = page.url()
      expect(url).not.toContain('/login')
      await takeScreenshot(page, '03-session-persisted')
    })
  })

  test.describe('2. Dashboard', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('2.1 Dashboard loads with stats', async ({ page }) => {
      await page.goto('/dashboard')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      
      const content = await page.content()
      expect(content.length).toBeGreaterThan(1000)
      await takeScreenshot(page, '04-dashboard-stats')
    })

    test('2.2 Navigation sidebar works', async ({ page }) => {
      const nav = page.locator('nav, aside, [role="navigation"]').first()
      await expect(nav).toBeVisible({ timeout: 10000 })
      await takeScreenshot(page, '05-navigation-sidebar')
    })
  })

  test.describe('3. Data Sources', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('3.1 List data sources', async ({ page }) => {
      await page.goto('/data-sources')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '06-data-sources-list')
    })

    test('3.2 New data source form', async ({ page }) => {
      await page.goto('/data-sources/new')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '07-data-source-new-form')
    })

    test('3.3 Create data source via API', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/datasources', {
        name: `E2E Test DS ${Date.now()}`,
        type: 'postgres',
        config: { host: 'localhost', port: 5432, database: 'test' }
      })
      expect(response?.ok()).toBeTruthy()
      const data = await response?.json()
      expect(data.id).toBeDefined()
      
      // Cleanup
      await apiRequest(page, 'DELETE', `/datasources/${data.id}`)
    })
  })

  test.describe('4. Classification', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('4.1 Classification page', async ({ page }) => {
      await page.goto('/classification')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '08-classification-page')
    })

    test('4.2 Classification rules', async ({ page }) => {
      await page.goto('/classification/rules')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '09-classification-rules')
    })

    test('4.3 Classification models', async ({ page }) => {
      await page.goto('/classification/models')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '10-classification-models')
    })

    test('4.4 Text classification API', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/classify/text', {
        text: 'My email is john@example.com and SSN is 123-45-6789'
      })
      expect(response?.ok()).toBeTruthy()
      const data = await response?.json()
      expect(data.entities).toBeDefined()
    })
  })

  test.describe('5. Governance', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('5.1 Governance page', async ({ page }) => {
      await page.goto('/governance')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '11-governance-page')
    })

    test('5.2 Policies list', async ({ page }) => {
      await page.goto('/governance/policies')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '12-governance-policies')
    })

    test('5.3 Create policy via API', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/governance/policies', {
        name: `E2E Test Policy ${Date.now()}`,
        type: 'access',
        active: true
      })
      expect(response?.ok()).toBeTruthy()
    })

    test('5.4 Policy evaluation API', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/governance/evaluate', {
        data: 'John Doe email: john@example.com SSN: 123-45-6789'
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('6. AI Gate', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('6.1 AI Gate page', async ({ page }) => {
      await page.goto('/ai-gate')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '13-ai-gate-page')
    })

    test('6.2 AI Gate playground', async ({ page }) => {
      await page.goto('/ai-gate/playground')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '14-ai-gate-playground')
    })

    test('6.3 AI Gate query API', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/gate/query', {
        query: 'What is the customer data for order 12345?',
        max_chunks: 5
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('7. Data Quality', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('7.1 Quality page', async ({ page }) => {
      await page.goto('/quality')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '15-quality-page')
    })

    test('7.2 Quality rules', async ({ page }) => {
      await page.goto('/quality/rules')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '16-quality-rules')
    })
  })

  test.describe('8. Privacy', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('8.1 Privacy page', async ({ page }) => {
      await page.goto('/privacy')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '17-privacy-page')
    })

    test('8.2 DSAR page', async ({ page }) => {
      await page.goto('/privacy/dsar')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '18-privacy-dsar')
    })

    test('8.3 Consent page', async ({ page }) => {
      await page.goto('/privacy/consent')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '19-privacy-consent')
    })
  })

  test.describe('9. Audit', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('9.1 Audit page', async ({ page }) => {
      await page.goto('/audit')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '20-audit-page')
    })

    test('9.2 Lineage page', async ({ page }) => {
      await page.goto('/lineage')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '21-lineage-page')
    })
  })

  test.describe('10. Observability', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('10.1 Observability page', async ({ page }) => {
      await page.goto('/observability')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '22-observability-page')
    })

    test('10.2 Alerts page', async ({ page }) => {
      await page.goto('/observability/alerts')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '23-observability-alerts')
    })
  })

  test.describe('11. Compliance Advisor', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('11.1 Advisor page', async ({ page }) => {
      await page.goto('/advisor')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '24-advisor-page')
    })

    test('11.2 Compliance gaps', async ({ page }) => {
      await page.goto('/advisor/gaps')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '25-advisor-gaps')
    })
  })

  test.describe('12. ROT Data', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('12.1 ROT page', async ({ page }) => {
      await page.goto('/rot')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '26-rot-page')
    })
  })

  test.describe('13. Labels', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('13.1 Labels page', async ({ page }) => {
      await page.goto('/labels')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '27-labels-page')
    })

    test('13.2 Label rules', async ({ page }) => {
      await page.goto('/labels/rules')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '28-labels-rules')
    })
  })

  test.describe('14. Integrations', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('14.1 Integrations page', async ({ page }) => {
      await page.goto('/integrations')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '29-integrations-page')
    })
  })

  test.describe('15. Settings', () => {
    test.beforeEach(async ({ page }) => {
      await loginViaUI(page)
    })

    test('15.1 Settings page', async ({ page }) => {
      await page.goto('/settings')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '30-settings-page')
    })

    test('15.2 Users page', async ({ page }) => {
      await page.goto('/settings/users')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '31-settings-users')
    })

    test('15.3 API keys page', async ({ page }) => {
      await page.goto('/settings/api-keys')
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
      await takeScreenshot(page, '32-settings-api-keys')
    })
  })
})
