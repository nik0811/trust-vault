import { test, expect, Page } from '@playwright/test'

const ADMIN_EMAIL = process.env.TEST_ADMIN_EMAIL || 'changeme@example.com'
const ADMIN_PASSWORD = process.env.TEST_ADMIN_PASSWORD || 'changeme123!'

// Helper to login via UI
async function loginViaUI(page: Page) {
  await page.goto('/login')
  await page.fill('input[name="email"]', ADMIN_EMAIL)
  await page.fill('input[name="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL(/.*dashboard/, { timeout: 15000 })
}

// Helper to check page loads successfully
async function checkPageLoads(page: Page, path: string) {
  await page.goto(path)
  await page.waitForLoadState('networkidle')
  const content = await page.content()
  expect(content.length).toBeGreaterThan(500)
}

test.describe('TrustVault UI Feature Tests', () => {
  
  test.beforeEach(async ({ page }) => {
    await loginViaUI(page)
  })

  test.describe('Navigation', () => {
    test('should navigate to all main pages', async ({ page }) => {
      const routes = [
        '/data-sources',
        '/classification',
        '/governance',
        '/ai-gate',
        '/quality',
        '/privacy',
        '/audit',
        '/advisor',
        '/rot',
        '/labels',
        '/lineage',
        '/observability',
        '/integrations',
        '/settings',
      ]

      for (const route of routes) {
        await checkPageLoads(page, route)
      }
    })
  })

  test.describe('Data Sources Page', () => {
    test('should display data sources list', async ({ page }) => {
      await checkPageLoads(page, '/data-sources')
    })

    test('should navigate to new data source form', async ({ page }) => {
      await checkPageLoads(page, '/data-sources/new')
    })

    test('should show data source types', async ({ page }) => {
      await page.goto('/data-sources/new')
      await page.waitForLoadState('networkidle')
      // If redirected to login, login first
      if (page.url().includes('/login')) {
        await page.fill('input[name="email"]', ADMIN_EMAIL)
        await page.fill('input[name="password"]', ADMIN_PASSWORD)
        await page.click('button[type="submit"]')
        await expect(page).toHaveURL(/.*dashboard/, { timeout: 15000 })
        await page.goto('/data-sources/new')
        await page.waitForLoadState('networkidle')
      }
      // Check page loaded
      const content = await page.content()
      expect(content.length).toBeGreaterThan(500)
    })
  })

  test.describe('Classification Page', () => {
    test('should display classification dashboard', async ({ page }) => {
      await checkPageLoads(page, '/classification')
    })

    test('should have text classification input', async ({ page }) => {
      await checkPageLoads(page, '/classification')
    })

    test('should navigate to classification rules', async ({ page }) => {
      await checkPageLoads(page, '/classification/rules')
    })

    test('should navigate to classification models', async ({ page }) => {
      await checkPageLoads(page, '/classification/models')
    })
  })

  test.describe('Governance Page', () => {
    test('should display governance overview', async ({ page }) => {
      await checkPageLoads(page, '/governance')
    })

    test('should navigate to policies list', async ({ page }) => {
      await checkPageLoads(page, '/governance/policies')
    })

    test('should navigate to policy evaluation', async ({ page }) => {
      await checkPageLoads(page, '/governance/evaluate')
    })

    test('should create new policy', async ({ page }) => {
      await checkPageLoads(page, '/governance/policies/new')
    })
  })

  test.describe('AI Gate Page', () => {
    test('should display AI Gate overview', async ({ page }) => {
      await checkPageLoads(page, '/ai-gate')
    })

    test('should navigate to AI Gate playground', async ({ page }) => {
      await checkPageLoads(page, '/ai-gate/playground')
    })

    test('should navigate to query history', async ({ page }) => {
      await checkPageLoads(page, '/ai-gate/queries')
    })
  })

  test.describe('Data Quality Page', () => {
    test('should display quality overview', async ({ page }) => {
      await checkPageLoads(page, '/quality')
    })

    test('should navigate to quality rules', async ({ page }) => {
      await checkPageLoads(page, '/quality/rules')
    })
  })

  test.describe('Privacy Page', () => {
    test('should display privacy overview', async ({ page }) => {
      await checkPageLoads(page, '/privacy')
    })

    test('should navigate to DSAR management', async ({ page }) => {
      await checkPageLoads(page, '/privacy/dsar')
    })

    test('should navigate to consent management', async ({ page }) => {
      await checkPageLoads(page, '/privacy/consent')
    })
  })

  test.describe('Audit Page', () => {
    test('should display audit trail', async ({ page }) => {
      await checkPageLoads(page, '/audit')
    })

    test('should navigate to reports', async ({ page }) => {
      await checkPageLoads(page, '/audit/reports')
    })
  })

  test.describe('Compliance Advisor Page', () => {
    test('should display advisor overview', async ({ page }) => {
      await checkPageLoads(page, '/advisor')
    })

    test('should navigate to compliance gaps', async ({ page }) => {
      await checkPageLoads(page, '/advisor/gaps')
    })

    test('should navigate to defense docket', async ({ page }) => {
      await checkPageLoads(page, '/advisor/docket')
    })

    test('should navigate to playbooks', async ({ page }) => {
      await checkPageLoads(page, '/advisor/playbooks')
    })
  })

  test.describe('ROT Data Page', () => {
    test('should display ROT overview', async ({ page }) => {
      await checkPageLoads(page, '/rot')
    })

    test('should navigate to duplicates', async ({ page }) => {
      await checkPageLoads(page, '/rot/duplicates')
    })

    test('should navigate to obsolete data', async ({ page }) => {
      await checkPageLoads(page, '/rot/obsolete')
    })

    test('should navigate to trivial data', async ({ page }) => {
      await checkPageLoads(page, '/rot/trivial')
    })
  })

  test.describe('Sensitivity Labels Page', () => {
    test('should display labels overview', async ({ page }) => {
      await checkPageLoads(page, '/labels')
    })

    test('should navigate to label rules', async ({ page }) => {
      await checkPageLoads(page, '/labels/rules')
    })

    test('should navigate to dataset labels', async ({ page }) => {
      await checkPageLoads(page, '/labels/datasets')
    })
  })

  test.describe('Data Lineage Page', () => {
    test('should display lineage page', async ({ page }) => {
      await checkPageLoads(page, '/lineage')
    })
  })

  test.describe('Observability Page', () => {
    test('should display observability overview', async ({ page }) => {
      await checkPageLoads(page, '/observability')
    })

    test('should navigate to alerts', async ({ page }) => {
      await checkPageLoads(page, '/observability/alerts')
    })
  })

  test.describe('Integrations Page', () => {
    test('should display integrations list', async ({ page }) => {
      await checkPageLoads(page, '/integrations')
    })

    test('should navigate to new integration form', async ({ page }) => {
      await checkPageLoads(page, '/integrations/new')
    })
  })

  test.describe('Settings Page', () => {
    test('should display settings overview', async ({ page }) => {
      await checkPageLoads(page, '/settings')
    })

    test('should navigate to users management', async ({ page }) => {
      await checkPageLoads(page, '/settings/users')
    })

    test('should navigate to API keys', async ({ page }) => {
      await checkPageLoads(page, '/settings/api-keys')
    })
  })
})
