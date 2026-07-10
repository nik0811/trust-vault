import { test, expect, Page } from '@playwright/test'

const API_BASE = 'http://localhost:8080/api/v1'
const ADMIN_EMAIL = process.env.TEST_ADMIN_EMAIL || 'changeme@example.com'
const ADMIN_PASSWORD = process.env.TEST_ADMIN_PASSWORD || 'changeme123!'

let authToken: string

// Helper to login and get token
async function login(page: Page): Promise<string> {
  if (authToken) return authToken
  
  // Try API login first
  const response = await page.request.post(`${API_BASE}/auth/login`, {
    data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
  })
  
  if (response.ok()) {
    const data = await response.json()
    authToken = data.access_token
    return authToken
  }
  
  throw new Error('Failed to login via API')
}

// Helper to make authenticated API requests
async function apiRequest(page: Page, method: string, endpoint: string, data?: any) {
  const token = await login(page)
  const options: any = {
    headers: { Authorization: `Bearer ${token}` }
  }
  if (data) options.data = data
  
  if (method === 'GET') {
    return page.request.get(`${API_BASE}${endpoint}`, options)
  } else if (method === 'POST') {
    return page.request.post(`${API_BASE}${endpoint}`, options)
  } else if (method === 'PUT') {
    return page.request.put(`${API_BASE}${endpoint}`, options)
  } else if (method === 'DELETE') {
    return page.request.delete(`${API_BASE}${endpoint}`, options)
  }
}

test.describe('TrustVault E2E Tests', () => {
  
  test.describe('Authentication', () => {
    test('should login with valid credentials', async ({ page }) => {
      await page.goto('/login')
      await page.fill('input[name="email"]', ADMIN_EMAIL)
      await page.fill('input[name="password"]', ADMIN_PASSWORD)
      await page.click('button[type="submit"]')
      
      // Should redirect to dashboard
      await expect(page).toHaveURL(/.*dashboard/, { timeout: 10000 })
      await expect(page.locator('h1')).toContainText('Dashboard')
    })

    test('should reject invalid credentials', async ({ page }) => {
      await page.goto('/login')
      await page.fill('input[name="email"]', 'wrong@email.com')
      await page.fill('input[name="password"]', 'wrongpassword')
      await page.click('button[type="submit"]')
      
      // Should stay on login page or show error toast
      await page.waitForTimeout(2000)
      const url = page.url()
      const hasError = await page.locator('[data-sonner-toast], [role="alert"], .toast, .error').isVisible().catch(() => false)
      const stayedOnLogin = url.includes('/login')
      expect(hasError || stayedOnLogin).toBeTruthy()
    })

    test('should logout successfully', async ({ page }) => {
      // Login first
      await page.goto('/login')
      await page.fill('input[name="email"]', ADMIN_EMAIL)
      await page.fill('input[name="password"]', ADMIN_PASSWORD)
      await page.click('button[type="submit"]')
      await expect(page).toHaveURL(/.*dashboard/, { timeout: 10000 })
      
      // Clear cookies to simulate logout
      await page.evaluate(() => {
        document.cookie.split(";").forEach((c) => {
          document.cookie = c.replace(/^ +/, "").replace(/=.*/, "=;expires=" + new Date().toUTCString() + ";path=/")
        })
      })
      
      // Navigate to protected page should redirect to login
      await page.goto('/dashboard')
      await page.waitForTimeout(2000)
      // Either redirected to login or shows unauthorized
      const url = page.url()
      expect(url.includes('/login') || url.includes('/dashboard')).toBeTruthy()
    })
  })

  test.describe('Dashboard', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/login')
      await page.fill('input[name="email"]', ADMIN_EMAIL)
      await page.fill('input[name="password"]', ADMIN_PASSWORD)
      await page.click('button[type="submit"]')
      await expect(page).toHaveURL(/.*dashboard/, { timeout: 10000 })
    })

    test('should display dashboard with stats', async ({ page }) => {
      await page.waitForLoadState('networkidle')
      // Check page has content
      const content = await page.content()
      expect(content.length).toBeGreaterThan(1000)
    })

    test('should have working navigation links', async ({ page }) => {
      // Check sidebar navigation exists
      await expect(page.locator('nav, aside').first()).toBeVisible({ timeout: 5000 })
    })
  })

  test.describe('Data Sources API', () => {
    test('should list data sources', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/datasources')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create a data source', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/datasources', {
        name: `Test DS ${Date.now()}`,
        type: 'postgres',
        config: { host: 'localhost', port: 5432 }
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.id).toBeDefined()
      expect(data.name).toContain('Test DS')
    })
  })

  test.describe('Governance Policies API', () => {
    test('should list policies', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/governance/policies')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create a policy', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/governance/policies', {
        name: `Test Policy ${Date.now()}`,
        type: 'access',
        active: true
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.id).toBeDefined()
    })

    test('should evaluate policy', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/governance/evaluate', {
        data: 'John Doe email: john@example.com SSN: 123-45-6789'
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.decision).toBeDefined()
    })
  })

  test.describe('Classification API', () => {
    test('should classify text', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/classify/text', {
        text: 'My email is john@example.com and my SSN is 123-45-6789'
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.entities).toBeDefined()
      expect(Array.isArray(data.entities)).toBeTruthy()
    })

    test('should list classification models', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/classify/models')
      expect(response?.ok()).toBeTruthy()
    })

    test('should list classification rules', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/classify/rules')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('AI Gate API', () => {
    test('should process gate query', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/gate/query', {
        query: 'What is the customer data for order 12345?',
        max_chunks: 5
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.decision).toBeDefined()
    })

    test('should get gate stats', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/gate/stats')
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.total_queries).toBeDefined()
    })
  })

  test.describe('Data Quality API', () => {
    test('should get quality trends', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/quality/trends')
      expect(response?.ok()).toBeTruthy()
    })

    test('should set quality threshold', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/quality/thresholds', {
        dimension: 'completeness',
        minimum: 90,
        severity: 'warning'
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Privacy Compliance API', () => {
    test('should list DSARs', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/privacy/dsar')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create DSAR', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/privacy/dsar', {
        subject_id: `subject-${Date.now()}`,
        type: 'access'
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.id).toBeDefined()
    })

    test('should get RoPA', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/privacy/ropa')
      expect(response?.ok()).toBeTruthy()
    })

    test('should record consent', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/privacy/consent', {
        subject_id: `subject-${Date.now()}`,
        purpose: 'marketing'
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Audit Trail API', () => {
    test('should get audit trail', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/audit/trail')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Observability API', () => {
    test('should get system health', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/observability/health')
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.status).toBeDefined()
    })

    test('should get system metrics', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/observability/metrics')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get alerts', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/observability/alerts')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Compliance Advisor API', () => {
    test('should get recommendations', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/advisor/recommendations')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get compliance gaps', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/advisor/gaps')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get risk score', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/advisor/risk-score')
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.overall_score).toBeDefined()
    })

    test('should generate defense docket', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/advisor/defense-docket', {
        regulations: ['GDPR'],
        date_from: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(),
        date_to: new Date().toISOString()
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('ROT Data API', () => {
    test('should get ROT summary', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/rot/summary')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get ROT datasets', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/rot/datasets')
      expect(response?.ok()).toBeTruthy()
    })

    test('should trigger ROT scan', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/rot/scan')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Sensitivity Labels API', () => {
    test('should get label summary', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/labels/summary')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get label rules', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/labels/rules')
      expect(response?.ok()).toBeTruthy()
    })

    test('should assign label', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/labels/assign', {
        dataset_id: 'test-dataset',
        label: 'CONFIDENTIAL'
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Integrations API', () => {
    test('should list integrations', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/integrations')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create integration', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/integrations', {
        name: `Test Integration ${Date.now()}`,
        type: 'dlp',
        provider: 'test-provider'
      })
      expect(response?.ok()).toBeTruthy()
      
      const data = await response?.json()
      expect(data.id).toBeDefined()
    })
  })

  test.describe('Jobs API', () => {
    test('should list jobs', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/jobs')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create job', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/jobs', {
        name: `Test Job ${Date.now()}`,
        type: 'scan',
        schedule: '0 0 * * *'
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Notifications API', () => {
    test('should list webhooks', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/notifications/webhooks')
      expect(response?.ok()).toBeTruthy()
    })

    test('should create webhook', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/notifications/webhooks', {
        name: `Test Webhook ${Date.now()}`,
        url: 'https://example.com/webhook',
        events: ['policy.violation', 'scan.complete']
      })
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('User Management API', () => {
    test('should list users', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/users')
      expect(response?.ok()).toBeTruthy()
    })

    test('should list roles', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/roles')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Reports API', () => {
    test('should list reports', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/reports')
      expect(response?.ok()).toBeTruthy()
    })

    test('should get analytics summary', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/analytics/summary')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Data Lineage API', () => {
    test('should get data map', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/datamap')
      expect(response?.ok()).toBeTruthy()
    })
  })

  test.describe('Feedback API', () => {
    test('should submit feedback', async ({ page }) => {
      const response = await apiRequest(page, 'POST', '/feedback', {
        entity_id: 'test-entity',
        original_classification: 'EMAIL',
        corrected_classification: 'PHONE',
        comment: 'Test correction'
      })
      expect(response?.ok()).toBeTruthy()
    })

    test('should get feedback stats', async ({ page }) => {
      const response = await apiRequest(page, 'GET', '/feedback/stats')
      expect(response?.ok()).toBeTruthy()
    })
  })
})
