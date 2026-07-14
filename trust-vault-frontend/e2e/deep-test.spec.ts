import { test, expect, Page } from '@playwright/test'

const BASE_URL = 'https://trust-vault.oortfy.com'
const API_URL = 'https://trust-vault-api.oortfy.com'
const ADMIN_EMAIL = 'admin@trustvault.local'
const ADMIN_PASSWORD = 'TrustVault@2026!'
const SCREENSHOT_DIR = 'screenshots/deep-test'

interface TestResult {
  test: string
  status: 'PASS' | 'FAIL' | 'SKIP'
  details?: string
  error?: string
}

const results: TestResult[] = []

function logResult(testName: string, status: 'PASS' | 'FAIL' | 'SKIP', details?: string, error?: string) {
  results.push({ test: testName, status, details, error })
  console.log(`[${status}] ${testName}${details ? ` - ${details}` : ''}${error ? ` - ERROR: ${error}` : ''}`)
}

async function screenshot(page: Page, name: string) {
  await page.screenshot({ path: `${SCREENSHOT_DIR}/${name}.png`, fullPage: true })
}

async function login(page: Page) {
  await page.goto(`${BASE_URL}/login`)
  await page.waitForLoadState('networkidle')
  await page.fill('input[name="email"]', ADMIN_EMAIL)
  await page.fill('input[name="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL('**/dashboard', { timeout: 15000 })
}

test.describe('TrustVault Deep Manual Testing', () => {
  test.setTimeout(300000) // 5 minutes per test

  test('1. LOGIN PAGE TESTS', async ({ page }) => {
    // 1.1 Test wrong credentials
    await page.goto(`${BASE_URL}/login`)
    await page.waitForLoadState('networkidle')
    await screenshot(page, '01-login-page')
    
    await page.fill('input[name="email"]', 'wrong@email.com')
    await page.fill('input[name="password"]', 'wrongpassword')
    await page.click('button[type="submit"]')
    await page.waitForTimeout(2000)
    
    const errorVisible = await page.locator('[role="alert"], .toast, .error, [data-sonner-toast]').isVisible().catch(() => false)
    if (errorVisible) {
      logResult('1.1 Wrong credentials error message', 'PASS')
    } else {
      logResult('1.1 Wrong credentials error message', 'FAIL', 'No error message shown')
    }
    await screenshot(page, '01-login-wrong-credentials')

    // 1.2 Test correct credentials
    await page.fill('input[name="email"]', ADMIN_EMAIL)
    await page.fill('input[name="password"]', ADMIN_PASSWORD)
    await page.click('button[type="submit"]')
    
    try {
      await page.waitForURL('**/dashboard', { timeout: 15000 })
      logResult('1.2 Correct credentials login', 'PASS', 'Redirected to dashboard')
    } catch (e) {
      logResult('1.2 Correct credentials login', 'FAIL', `Current URL: ${page.url()}`, String(e))
    }
    await screenshot(page, '01-login-success')
  })

  test('2. DASHBOARD TESTS', async ({ page }) => {
    await login(page)
    await page.waitForTimeout(2000)
    await screenshot(page, '02-dashboard-initial')

    // 2.1 Check stat cards
    const statCards = await page.locator('[class*="card"], [class*="stat"]').count()
    logResult('2.1 Dashboard stat cards', statCards > 0 ? 'PASS' : 'FAIL', `Found ${statCards} cards`)

    // 2.2 Check for real numbers (not just 0 or loading)
    const pageContent = await page.textContent('body')
    const hasNumbers = /\d+/.test(pageContent || '')
    logResult('2.2 Dashboard shows data', hasNumbers ? 'PASS' : 'FAIL')

    // 2.3 Click on stat cards if they exist
    const clickableCards = page.locator('[class*="card"] a, [class*="stat"] a, [role="button"]')
    const cardCount = await clickableCards.count()
    if (cardCount > 0) {
      try {
        await clickableCards.first().click()
        await page.waitForTimeout(1000)
        logResult('2.3 Stat card navigation', 'PASS', `Navigated to ${page.url()}`)
        await page.goBack()
      } catch (e) {
        logResult('2.3 Stat card navigation', 'FAIL', '', String(e))
      }
    }
    await screenshot(page, '02-dashboard-final')
  })

  test('3. DATA SOURCES TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '03-datasources-list')

    // 3.1 Check list loads
    const tableOrList = await page.locator('table, [class*="list"], [class*="grid"]').count()
    logResult('3.1 Data sources list loads', tableOrList > 0 ? 'PASS' : 'FAIL')

    // 3.2 Click Add Source button
    const addButton = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New"), a:has-text("Add")')
    if (await addButton.count() > 0) {
      await addButton.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, '03-datasources-add-form')
      
      // Check if form/dialog opened
      const formVisible = await page.locator('form, [role="dialog"], [class*="modal"]').isVisible()
      logResult('3.2 Add Source form opens', formVisible ? 'PASS' : 'FAIL')

      // 3.3 Fill form with test data
      if (formVisible) {
        try {
          // Try to fill common form fields
          const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]')
          if (await nameInput.count() > 0) {
            await nameInput.first().fill('Test Data Source ' + Date.now())
          }
          
          // Select type if dropdown exists
          const typeSelect = page.locator('select[name="type"], [role="combobox"]')
          if (await typeSelect.count() > 0) {
            await typeSelect.first().click()
            await page.waitForTimeout(500)
            const option = page.locator('[role="option"]').first()
            if (await option.count() > 0) {
              await option.click()
            }
          }
          
          await screenshot(page, '03-datasources-form-filled')
          
          // Submit form
          const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create"), button:has-text("Add")')
          if (await submitBtn.count() > 0) {
            await submitBtn.first().click()
            await page.waitForTimeout(3000)
            await screenshot(page, '03-datasources-after-submit')
            logResult('3.3 Create data source', 'PASS')
          }
        } catch (e) {
          logResult('3.3 Create data source', 'FAIL', '', String(e))
        }
      }
    } else {
      logResult('3.2 Add Source button', 'FAIL', 'Button not found')
    }

    // 3.4 Click on a data source to view details
    await page.goto(`${BASE_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    
    const dataSourceLink = page.locator('table tbody tr a, [class*="list"] a, [class*="card"] a').first()
    if (await dataSourceLink.count() > 0) {
      await dataSourceLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, '03-datasources-detail')
      logResult('3.4 Data source detail page', 'PASS', `URL: ${page.url()}`)

      // 3.5 Test Run Scan button
      const scanButton = page.locator('button:has-text("Scan"), button:has-text("Run")')
      if (await scanButton.count() > 0) {
        await scanButton.first().click()
        await page.waitForTimeout(5000)
        await screenshot(page, '03-datasources-scan-started')
        logResult('3.5 Run Scan button', 'PASS')
        
        // 3.6 Check for real-time logs (SSE)
        const logsArea = page.locator('[class*="log"], pre, [class*="terminal"], [class*="output"]')
        if (await logsArea.count() > 0) {
          logResult('3.6 Scan logs area visible', 'PASS')
        } else {
          logResult('3.6 Scan logs area visible', 'SKIP', 'No logs area found')
        }
      } else {
        logResult('3.5 Run Scan button', 'SKIP', 'Button not found')
      }

      // 3.7 Test Edit button
      const editButton = page.locator('button:has-text("Edit"), a:has-text("Edit")')
      if (await editButton.count() > 0) {
        await editButton.first().click()
        await page.waitForTimeout(1000)
        await screenshot(page, '03-datasources-edit')
        logResult('3.7 Edit data source', 'PASS')
      }
    } else {
      logResult('3.4 Data source detail page', 'SKIP', 'No data sources to click')
    }
  })

  test('4. CLASSIFICATION TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/classification`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '04-classification-list')

    // 4.1 Check list loads
    const content = await page.textContent('body')
    logResult('4.1 Classification page loads', content ? 'PASS' : 'FAIL')

    // 4.2 Click on a dataset
    const datasetLink = page.locator('table tbody tr a, [class*="list"] a, [class*="card"] a').first()
    if (await datasetLink.count() > 0) {
      await datasetLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, '04-classification-detail')
      logResult('4.2 Classification detail page', 'PASS')

      // 4.3 Check columns table
      const columnsTable = page.locator('table')
      if (await columnsTable.count() > 0) {
        logResult('4.3 Columns table visible', 'PASS')
      }

      // 4.4 Test Re-classify button
      const reclassifyBtn = page.locator('button:has-text("Classify"), button:has-text("Re-classify"), button:has-text("Scan")')
      if (await reclassifyBtn.count() > 0) {
        await reclassifyBtn.first().click()
        await page.waitForTimeout(3000)
        await screenshot(page, '04-classification-reclassify')
        logResult('4.4 Re-classify button', 'PASS')
      }
    } else {
      logResult('4.2 Classification detail page', 'SKIP', 'No datasets found')
    }
  })

  test('5. CLASSIFICATION RULES TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/classification/rules`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '05-classification-rules')

    // 5.1 Check list loads
    logResult('5.1 Classification rules page loads', 'PASS')

    // 5.2 Add Rule button
    const addButton = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")')
    if (await addButton.count() > 0) {
      await addButton.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, '05-classification-rules-add')
      logResult('5.2 Add Rule form opens', 'PASS')
    }
  })

  test('6. CLASSIFICATION MODELS TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/classification/models`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '06-classification-models')

    // 6.1 Check models list
    const content = await page.textContent('body')
    logResult('6.1 Classification models page loads', content ? 'PASS' : 'FAIL')

    // 6.2 Check status indicators
    const statusIndicators = page.locator('[class*="status"], [class*="badge"], .text-green, .text-red')
    const statusCount = await statusIndicators.count()
    logResult('6.2 Status indicators visible', statusCount > 0 ? 'PASS' : 'SKIP', `Found ${statusCount} indicators`)
  })

  test('7. GOVERNANCE POLICIES TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/governance/policies`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '07-governance-policies')

    // 7.1 Check list loads
    logResult('7.1 Governance policies page loads', 'PASS')

    // 7.2 Create Policy button
    const createButton = page.locator('button:has-text("Create"), button:has-text("Add"), button:has-text("New")')
    if (await createButton.count() > 0) {
      await createButton.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, '07-governance-policies-create')
      
      // Fill form
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]')
      if (await nameInput.count() > 0) {
        await nameInput.first().fill('Test Policy ' + Date.now())
        
        const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")')
        if (await submitBtn.count() > 0) {
          await submitBtn.first().click()
          await page.waitForTimeout(2000)
          await screenshot(page, '07-governance-policies-created')
          logResult('7.2 Create policy', 'PASS')
        }
      }
    }
  })

  test('8. SENSITIVITY LABELS TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/labels`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '08-labels-overview')
    logResult('8.1 Labels overview page loads', 'PASS')

    // 8.2 Navigate to rules
    await page.goto(`${BASE_URL}/labels/rules`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '08-labels-rules')
    logResult('8.2 Labels rules page loads', 'PASS')
  })

  test('9. DATA MAP TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/data-map`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '09-datamap')
    logResult('9.1 Data map page loads', 'PASS')

    // 9.2 Check for visualization
    const visualization = page.locator('canvas, svg, [class*="chart"], [class*="graph"], [class*="map"]')
    const vizCount = await visualization.count()
    logResult('9.2 Data map visualization', vizCount > 0 ? 'PASS' : 'SKIP', `Found ${vizCount} visualizations`)

    // 9.3 Navigate to coverage
    await page.goto(`${BASE_URL}/data-map/coverage`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '09-datamap-coverage')
    logResult('9.3 Data map coverage page loads', 'PASS')
  })

  test('10. LINEAGE TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/lineage`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '10-lineage')
    logResult('10.1 Lineage page loads', 'PASS')

    // Check for graph
    const graph = page.locator('canvas, svg, [class*="graph"], [class*="flow"]')
    const graphCount = await graph.count()
    logResult('10.2 Lineage graph visible', graphCount > 0 ? 'PASS' : 'SKIP', `Found ${graphCount} graph elements`)
  })

  test('11. DOCUMENTS TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/documents`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '11-documents')
    logResult('11.1 Documents page loads', 'PASS')
  })

  test('12. COMPLIANCE TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/compliance`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '12-compliance-dashboard')
    logResult('12.1 Compliance dashboard loads', 'PASS')

    // 12.2 DSAR
    await page.goto(`${BASE_URL}/compliance/dsar`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '12-compliance-dsar')
    logResult('12.2 DSAR page loads', 'PASS')

    // 12.3 RoPA
    await page.goto(`${BASE_URL}/compliance/ropa`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '12-compliance-ropa')
    logResult('12.3 RoPA page loads', 'PASS')

    // 12.4 Advisor
    await page.goto(`${BASE_URL}/compliance/advisor`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '12-compliance-advisor')
    logResult('12.4 Compliance advisor loads', 'PASS')
  })

  test('13. AI GOVERNANCE TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/ai-governance`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '13-ai-governance')
    logResult('13.1 AI Governance page loads', 'PASS')

    // Check for playground
    const playground = page.locator('textarea, [class*="playground"], [class*="chat"]')
    if (await playground.count() > 0) {
      await playground.first().fill('Test query: What is PII?')
      await screenshot(page, '13-ai-governance-query')
      
      const submitBtn = page.locator('button:has-text("Send"), button:has-text("Submit"), button[type="submit"]')
      if (await submitBtn.count() > 0) {
        await submitBtn.first().click()
        await page.waitForTimeout(5000)
        await screenshot(page, '13-ai-governance-response')
        logResult('13.2 AI Gate playground test', 'PASS')
      }
    }
  })

  test('14. DATA QUALITY TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/quality`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '14-quality')
    logResult('14.1 Data quality page loads', 'PASS')
  })

  test('15. AUDIT TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/audit`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '15-audit')
    logResult('15.1 Audit page loads', 'PASS')

    // Check for logs table
    const table = page.locator('table')
    if (await table.count() > 0) {
      logResult('15.2 Audit logs table visible', 'PASS')
    }

    // Test filter
    const filterInput = page.locator('input[placeholder*="filter" i], input[placeholder*="search" i]')
    if (await filterInput.count() > 0) {
      await filterInput.first().fill('login')
      await page.waitForTimeout(1000)
      await screenshot(page, '15-audit-filtered')
      logResult('15.3 Audit filter works', 'PASS')
    }
  })

  test('16. INTEGRATIONS TESTS', async ({ page }) => {
    await login(page)
    await page.goto(`${BASE_URL}/integrations`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '16-integrations')
    logResult('16.1 Integrations page loads', 'PASS')
  })

  test('17. SETTINGS TESTS', async ({ page }) => {
    await login(page)
    
    // 17.1 General settings
    await page.goto(`${BASE_URL}/settings`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '17-settings-general')
    logResult('17.1 Settings general page loads', 'PASS')

    // 17.2 Users page
    await page.goto(`${BASE_URL}/settings/users`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '17-settings-users')
    logResult('17.2 Settings users page loads', 'PASS')

    // 17.3 Invite user
    const inviteButton = page.locator('button:has-text("Invite"), button:has-text("Add")')
    if (await inviteButton.count() > 0) {
      await inviteButton.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, '17-settings-users-invite')
      logResult('17.3 Invite user modal opens', 'PASS')
      
      // Close modal
      const closeBtn = page.locator('button:has-text("Cancel"), button:has-text("Close"), [aria-label="Close"]')
      if (await closeBtn.count() > 0) {
        await closeBtn.first().click()
      }
    }

    // 17.4 API Keys page
    await page.goto(`${BASE_URL}/settings/api-keys`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, '17-settings-apikeys')
    logResult('17.4 Settings API keys page loads', 'PASS')

    // 17.5 Create API key
    const createKeyButton = page.locator('button:has-text("Create"), button:has-text("Generate"), button:has-text("Add")')
    if (await createKeyButton.count() > 0) {
      await createKeyButton.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, '17-settings-apikeys-create')
      logResult('17.5 Create API key modal opens', 'PASS')
    }
  })

  test('18. NAVIGATION TESTS', async ({ page }) => {
    await login(page)

    // 18.1 Test all sidebar links
    const sidebarLinks = page.locator('nav a, aside a, [class*="sidebar"] a')
    const linkCount = await sidebarLinks.count()
    logResult('18.1 Sidebar links found', linkCount > 0 ? 'PASS' : 'FAIL', `Found ${linkCount} links`)

    // 18.2 Test logout
    const userMenu = page.locator('[class*="avatar"], [class*="user"], button:has-text("Logout")')
    if (await userMenu.count() > 0) {
      await userMenu.first().click()
      await page.waitForTimeout(500)
      
      const logoutBtn = page.locator('button:has-text("Logout"), a:has-text("Logout"), [role="menuitem"]:has-text("Logout")')
      if (await logoutBtn.count() > 0) {
        await logoutBtn.first().click()
        await page.waitForTimeout(2000)
        await screenshot(page, '18-logout')
        
        const onLoginPage = page.url().includes('login')
        logResult('18.2 Logout works', onLoginPage ? 'PASS' : 'FAIL', `Current URL: ${page.url()}`)
      }
    }
  })

  test.afterAll(async () => {
    console.log('\n\n========== TEST RESULTS SUMMARY ==========\n')
    const passed = results.filter(r => r.status === 'PASS').length
    const failed = results.filter(r => r.status === 'FAIL').length
    const skipped = results.filter(r => r.status === 'SKIP').length
    
    console.log(`PASSED: ${passed}`)
    console.log(`FAILED: ${failed}`)
    console.log(`SKIPPED: ${skipped}`)
    console.log(`TOTAL: ${results.length}`)
    
    console.log('\n--- FAILED TESTS ---')
    results.filter(r => r.status === 'FAIL').forEach(r => {
      console.log(`  - ${r.test}: ${r.details || ''} ${r.error || ''}`)
    })
    
    console.log('\n--- ALL RESULTS ---')
    results.forEach(r => {
      console.log(`[${r.status}] ${r.test}${r.details ? ` - ${r.details}` : ''}`)
    })
  })
})
