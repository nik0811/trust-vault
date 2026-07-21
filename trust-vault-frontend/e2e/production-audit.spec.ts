import { test, expect, Page, ConsoleMessage, Request, Response } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'

const FRONTEND_URL = 'https://app.securelens.ai'
const API_BASE = 'https://api.securelens.ai/api/v1'
const ADMIN_EMAIL = 'admin@securelens.local'
const ADMIN_PASSWORD = 'SecureLens@2026!'

const SCREENSHOTS_DIR = path.join(__dirname, '..', 'audit-screenshots')
const REPORT_FILE = path.join(__dirname, '..', 'audit-report.json')

interface Issue {
  severity: 'critical' | 'major' | 'minor'
  page: string
  url: string
  description: string
  details?: string
  screenshot?: string
  apiResponse?: any
}

interface PageAudit {
  page: string
  url: string
  status: 'pass' | 'fail' | 'partial'
  issues: Issue[]
  consoleErrors: string[]
  failedApiCalls: { url: string; status: number; body?: string }[]
  dataChecks: { check: string; passed: boolean; value?: string }[]
}

const auditResults: PageAudit[] = []
let authToken: string

if (!fs.existsSync(SCREENSHOTS_DIR)) {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true })
}

async function takeScreenshot(page: Page, name: string): Promise<string> {
  const screenshotPath = path.join(SCREENSHOTS_DIR, `${name.replace(/[^a-z0-9]/gi, '-')}.png`)
  await page.screenshot({ path: screenshotPath, fullPage: true })
  return screenshotPath
}

async function loginViaAPI(page: Page): Promise<string> {
  if (authToken) return authToken
  
  const response = await page.request.post(`${API_BASE}/auth/login`, {
    data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
  })
  
  if (response.ok()) {
    const data = await response.json()
    authToken = data.access_token
    return authToken
  }
  
  throw new Error(`Failed to login via API: ${response.status()} - ${await response.text()}`)
}

async function loginViaUI(page: Page) {
  await page.goto(`${FRONTEND_URL}/login`)
  await page.waitForLoadState('networkidle')
  
  await page.fill('input[name="email"]', ADMIN_EMAIL)
  await page.fill('input[name="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  
  await expect(page).toHaveURL(/.*dashboard/, { timeout: 30000 })
}

async function checkForEmptyData(page: Page): Promise<{ check: string; passed: boolean; value?: string }[]> {
  const checks: { check: string; passed: boolean; value?: string }[] = []
  
  const emptyIndicators = [
    { selector: ':text("No data")', name: 'No data message' },
    { selector: ':text("No results")', name: 'No results message' },
    { selector: ':text("Nothing to show")', name: 'Nothing to show message' },
    { selector: ':text("Empty")', name: 'Empty message' },
    { selector: ':text("N/A")', name: 'N/A values' },
    { selector: ':text("--")', name: 'Dash placeholders' },
  ]
  
  for (const indicator of emptyIndicators) {
    const count = await page.locator(indicator.selector).count()
    checks.push({
      check: indicator.name,
      passed: count === 0,
      value: count > 0 ? `Found ${count} instances` : 'None found'
    })
  }
  
  const statCards = await page.locator('[class*="stat"], [class*="card"], [class*="metric"]').all()
  let zeroCount = 0
  for (const card of statCards) {
    const text = await card.textContent()
    if (text && /\b0\b/.test(text) && !/0\./.test(text)) {
      zeroCount++
    }
  }
  checks.push({
    check: 'Zero values in stat cards',
    passed: zeroCount < 3,
    value: `Found ${zeroCount} cards with zero values`
  })
  
  const tables = await page.locator('table tbody tr').count()
  checks.push({
    check: 'Table has data rows',
    passed: tables > 0 || await page.locator('table').count() === 0,
    value: `${tables} rows found`
  })
  
  return checks
}

async function auditPage(
  page: Page,
  pageName: string,
  pageUrl: string,
  additionalChecks?: (page: Page) => Promise<{ check: string; passed: boolean; value?: string }[]>
): Promise<PageAudit> {
  const audit: PageAudit = {
    page: pageName,
    url: pageUrl,
    status: 'pass',
    issues: [],
    consoleErrors: [],
    failedApiCalls: [],
    dataChecks: []
  }
  
  const consoleMessages: string[] = []
  const failedRequests: { url: string; status: number; body?: string }[] = []
  
  page.on('console', (msg: ConsoleMessage) => {
    if (msg.type() === 'error') {
      consoleMessages.push(msg.text())
    }
  })
  
  page.on('response', async (response: Response) => {
    const url = response.url()
    if (url.includes('/api/') && !response.ok() && response.status() !== 304) {
      let body = ''
      try {
        body = await response.text()
      } catch {}
      failedRequests.push({ url, status: response.status(), body })
    }
  })
  
  try {
    await page.goto(`${FRONTEND_URL}${pageUrl}`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    
    audit.consoleErrors = consoleMessages
    audit.failedApiCalls = failedRequests
    
    audit.dataChecks = await checkForEmptyData(page)
    
    if (additionalChecks) {
      const extraChecks = await additionalChecks(page)
      audit.dataChecks.push(...extraChecks)
    }
    
    const failedChecks = audit.dataChecks.filter(c => !c.passed)
    if (failedChecks.length > 0) {
      audit.status = 'partial'
      for (const check of failedChecks) {
        audit.issues.push({
          severity: 'major',
          page: pageName,
          url: pageUrl,
          description: `Data check failed: ${check.check}`,
          details: check.value
        })
      }
    }
    
    if (consoleMessages.length > 0) {
      audit.status = audit.status === 'pass' ? 'partial' : audit.status
      audit.issues.push({
        severity: 'minor',
        page: pageName,
        url: pageUrl,
        description: `Console errors found: ${consoleMessages.length}`,
        details: consoleMessages.join('\n')
      })
    }
    
    if (failedRequests.length > 0) {
      audit.status = 'fail'
      for (const req of failedRequests) {
        audit.issues.push({
          severity: 'critical',
          page: pageName,
          url: pageUrl,
          description: `API call failed: ${req.url}`,
          details: `Status: ${req.status}, Body: ${req.body?.substring(0, 200)}`
        })
      }
    }
    
    if (audit.issues.length > 0) {
      const screenshotPath = await takeScreenshot(page, `${pageName}-issues`)
      audit.issues.forEach(issue => issue.screenshot = screenshotPath)
    }
    
  } catch (error: any) {
    audit.status = 'fail'
    audit.issues.push({
      severity: 'critical',
      page: pageName,
      url: pageUrl,
      description: `Page failed to load`,
      details: error.message
    })
    try {
      await takeScreenshot(page, `${pageName}-error`)
    } catch {}
  }
  
  return audit
}

test.describe('SecureLens Production Audit', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    try {
      await loginViaAPI(page)
    } catch (e) {
      console.log('API login failed, will use UI login')
    }
    await page.close()
  })
  
  test.beforeEach(async ({ page }) => {
    await loginViaUI(page)
  })
  
  test.afterAll(async () => {
    fs.writeFileSync(REPORT_FILE, JSON.stringify(auditResults, null, 2))
    console.log(`\n${'='.repeat(80)}`)
    console.log('AUDIT REPORT SUMMARY')
    console.log('='.repeat(80))
    
    const critical = auditResults.flatMap(r => r.issues.filter(i => i.severity === 'critical'))
    const major = auditResults.flatMap(r => r.issues.filter(i => i.severity === 'major'))
    const minor = auditResults.flatMap(r => r.issues.filter(i => i.severity === 'minor'))
    
    console.log(`\nCRITICAL ISSUES (${critical.length}):`)
    critical.forEach(i => console.log(`  - [${i.page}] ${i.description}`))
    
    console.log(`\nMAJOR ISSUES (${major.length}):`)
    major.forEach(i => console.log(`  - [${i.page}] ${i.description}`))
    
    console.log(`\nMINOR ISSUES (${minor.length}):`)
    minor.forEach(i => console.log(`  - [${i.page}] ${i.description}`))
    
    console.log(`\nPASSING PAGES:`)
    auditResults.filter(r => r.status === 'pass').forEach(r => console.log(`  - ${r.page}`))
    
    console.log(`\nFull report saved to: ${REPORT_FILE}`)
    console.log(`Screenshots saved to: ${SCREENSHOTS_DIR}`)
  })

  test('1. Dashboard', async ({ page }) => {
    const audit = await auditPage(page, 'Dashboard', '/dashboard', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      
      const statValues = await p.locator('[class*="stat"] [class*="value"], [class*="metric"] [class*="number"]').allTextContents()
      const hasRealData = statValues.some(v => {
        const num = parseInt(v.replace(/[^0-9]/g, ''))
        return !isNaN(num) && num > 0
      })
      checks.push({ check: 'Dashboard has non-zero stats', passed: hasRealData, value: statValues.join(', ') })
      
      const chartExists = await p.locator('[class*="chart"], [class*="recharts"], svg').count() > 0
      checks.push({ check: 'Charts are rendered', passed: chartExists })
      
      return checks
    })
    auditResults.push(audit)
    expect(audit.status).not.toBe('fail')
  })

  test('2. Data Sources - List', async ({ page }) => {
    const audit = await auditPage(page, 'Data Sources List', '/data-sources', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      
      const rows = await p.locator('table tbody tr, [class*="list-item"], [class*="card"]').count()
      checks.push({ check: 'Data sources list has items', passed: rows > 0, value: `${rows} items` })
      
      return checks
    })
    auditResults.push(audit)
  })

  test('3. Data Sources - Detail Page', async ({ page }) => {
    await page.goto(`${FRONTEND_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    
    const firstLink = page.locator('table tbody tr a, [class*="list-item"] a').first()
    if (await firstLink.count() > 0) {
      await firstLink.click()
      await page.waitForLoadState('networkidle')
      
      const audit = await auditPage(page, 'Data Source Detail', page.url().replace(FRONTEND_URL, ''), async (p) => {
        const checks: { check: string; passed: boolean; value?: string }[] = []
        
        const hasName = await p.locator('h1, h2, [class*="title"]').first().textContent()
        checks.push({ check: 'Data source name displayed', passed: !!hasName && hasName.length > 0, value: hasName || '' })
        
        return checks
      })
      auditResults.push(audit)
    } else {
      auditResults.push({
        page: 'Data Source Detail',
        url: '/data-sources/[id]',
        status: 'fail',
        issues: [{ severity: 'major', page: 'Data Source Detail', url: '/data-sources', description: 'No data sources to view detail' }],
        consoleErrors: [],
        failedApiCalls: [],
        dataChecks: []
      })
    }
  })

  test('4. Classification - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Classification', '/classification')
    auditResults.push(audit)
  })

  test('5. Classification - Rules', async ({ page }) => {
    const audit = await auditPage(page, 'Classification Rules', '/classification/rules', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const rows = await p.locator('table tbody tr').count()
      checks.push({ check: 'Rules table has data', passed: rows > 0, value: `${rows} rules` })
      return checks
    })
    auditResults.push(audit)
  })

  test('6. Classification - Models', async ({ page }) => {
    const audit = await auditPage(page, 'Classification Models', '/classification/models')
    auditResults.push(audit)
  })

  test('7. Data Map - Geographic', async ({ page }) => {
    const audit = await auditPage(page, 'Data Map Geographic', '/data-map', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const mapExists = await p.locator('[class*="map"], svg, canvas').count() > 0
      checks.push({ check: 'Map visualization rendered', passed: mapExists })
      return checks
    })
    auditResults.push(audit)
  })

  test('8. Data Map - Residency', async ({ page }) => {
    const audit = await auditPage(page, 'Data Map Residency', '/data-map/residency')
    auditResults.push(audit)
  })

  test('9. Data Map - Lineage', async ({ page }) => {
    const audit = await auditPage(page, 'Data Lineage', '/lineage')
    auditResults.push(audit)
  })

  test('10. ROT Data', async ({ page }) => {
    const audit = await auditPage(page, 'ROT Data', '/rot', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const hasCategories = await p.locator('[class*="category"], [class*="card"]').count() > 0
      checks.push({ check: 'ROT categories displayed', passed: hasCategories })
      return checks
    })
    auditResults.push(audit)
  })

  test('11. Sensitivity Labels - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Sensitivity Labels', '/labels')
    auditResults.push(audit)
  })

  test('12. Sensitivity Labels - Rules', async ({ page }) => {
    const audit = await auditPage(page, 'Label Rules', '/labels/rules')
    auditResults.push(audit)
  })

  test('13. Data Quality - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Data Quality', '/quality', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const scoreExists = await p.locator('[class*="score"], [class*="percentage"]').count() > 0
      checks.push({ check: 'Quality scores displayed', passed: scoreExists })
      return checks
    })
    auditResults.push(audit)
  })

  test('14. Data Quality - Dimensions', async ({ page }) => {
    const audit = await auditPage(page, 'Quality Dimensions', '/quality/dimensions')
    auditResults.push(audit)
  })

  test('15. Data Quality - Profile', async ({ page }) => {
    const audit = await auditPage(page, 'Quality Profile', '/quality/profile')
    auditResults.push(audit)
  })

  test('16. Governance - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Governance', '/governance')
    auditResults.push(audit)
  })

  test('17. Governance - Policies', async ({ page }) => {
    const audit = await auditPage(page, 'Governance Policies', '/governance/policies', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const rows = await p.locator('table tbody tr').count()
      checks.push({ check: 'Policies table has data', passed: rows > 0, value: `${rows} policies` })
      return checks
    })
    auditResults.push(audit)
  })

  test('18. AI Gate - Main', async ({ page }) => {
    const audit = await auditPage(page, 'AI Gate', '/ai-gate')
    auditResults.push(audit)
  })

  test('19. AI Gate - Playground', async ({ page }) => {
    const audit = await auditPage(page, 'AI Gate Playground', '/ai-gate/playground')
    auditResults.push(audit)
  })

  test('20. AI Gate - Query History', async ({ page }) => {
    const audit = await auditPage(page, 'AI Gate History', '/ai-gate/history')
    auditResults.push(audit)
  })

  test('21. Privacy - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Privacy', '/privacy')
    auditResults.push(audit)
  })

  test('22. Privacy - DSAR', async ({ page }) => {
    const audit = await auditPage(page, 'DSAR', '/privacy/dsar')
    auditResults.push(audit)
  })

  test('23. Privacy - Consent', async ({ page }) => {
    const audit = await auditPage(page, 'Consent Management', '/privacy/consent')
    auditResults.push(audit)
  })

  test('24. Privacy - DPIA', async ({ page }) => {
    const audit = await auditPage(page, 'DPIA', '/privacy/dpia')
    auditResults.push(audit)
  })

  test('25. Compliance Advisor - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Compliance Advisor', '/advisor')
    auditResults.push(audit)
  })

  test('26. Compliance Advisor - Assessment', async ({ page }) => {
    const audit = await auditPage(page, 'Compliance Assessment', '/advisor/assessment')
    auditResults.push(audit)
  })

  test('27. Compliance Advisor - Gaps', async ({ page }) => {
    const audit = await auditPage(page, 'Compliance Gaps', '/advisor/gaps')
    auditResults.push(audit)
  })

  test('28. Compliance Advisor - Playbooks', async ({ page }) => {
    const audit = await auditPage(page, 'Compliance Playbooks', '/advisor/playbooks')
    auditResults.push(audit)
  })

  test('29. Audit Trail', async ({ page }) => {
    const audit = await auditPage(page, 'Audit Trail', '/audit', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const rows = await p.locator('table tbody tr').count()
      checks.push({ check: 'Audit logs displayed', passed: rows > 0, value: `${rows} entries` })
      return checks
    })
    auditResults.push(audit)
  })

  test('30. Observability - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Observability', '/observability')
    auditResults.push(audit)
  })

  test('31. Observability - Health', async ({ page }) => {
    const audit = await auditPage(page, 'System Health', '/observability/health')
    auditResults.push(audit)
  })

  test('32. Observability - Metrics', async ({ page }) => {
    const audit = await auditPage(page, 'Metrics', '/observability/metrics')
    auditResults.push(audit)
  })

  test('33. Observability - Alerts', async ({ page }) => {
    const audit = await auditPage(page, 'Alerts', '/observability/alerts')
    auditResults.push(audit)
  })

  test('34. Integrations', async ({ page }) => {
    const audit = await auditPage(page, 'Integrations', '/integrations')
    auditResults.push(audit)
  })

  test('35. Scheduled Jobs - List', async ({ page }) => {
    const audit = await auditPage(page, 'Scheduled Jobs', '/jobs')
    auditResults.push(audit)
  })

  test('36. Scheduled Jobs - History', async ({ page }) => {
    const audit = await auditPage(page, 'Job History', '/jobs/history')
    auditResults.push(audit)
  })

  test('37. Remediation - Actions', async ({ page }) => {
    const audit = await auditPage(page, 'Remediation Actions', '/remediation')
    auditResults.push(audit)
  })

  test('38. Remediation - Logs', async ({ page }) => {
    const audit = await auditPage(page, 'Remediation Logs', '/remediation/logs')
    auditResults.push(audit)
  })

  test('39. Reports - Generate', async ({ page }) => {
    const audit = await auditPage(page, 'Reports', '/reports')
    auditResults.push(audit)
  })

  test('40. Feedback - Corrections', async ({ page }) => {
    const audit = await auditPage(page, 'Feedback Corrections', '/feedback')
    auditResults.push(audit)
  })

  test('41. Feedback - Entities', async ({ page }) => {
    const audit = await auditPage(page, 'Feedback Entities', '/feedback/entities')
    auditResults.push(audit)
  })

  test('42. Documents - Upload', async ({ page }) => {
    const audit = await auditPage(page, 'Documents', '/documents')
    auditResults.push(audit)
  })

  test('43. Settings - Main', async ({ page }) => {
    const audit = await auditPage(page, 'Settings', '/settings')
    auditResults.push(audit)
  })

  test('44. Settings - Users', async ({ page }) => {
    const audit = await auditPage(page, 'Users', '/settings/users', async (p) => {
      const checks: { check: string; passed: boolean; value?: string }[] = []
      const rows = await p.locator('table tbody tr').count()
      checks.push({ check: 'Users table has data', passed: rows > 0, value: `${rows} users` })
      return checks
    })
    auditResults.push(audit)
  })

  test('45. Settings - API Keys', async ({ page }) => {
    const audit = await auditPage(page, 'API Keys', '/settings/api-keys')
    auditResults.push(audit)
  })
})
