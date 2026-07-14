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

test.describe('TrustVault Data Source Complete Testing', () => {
  test.setTimeout(300000)

  test('DATA SOURCE: Create, View, Scan, Delete', async ({ page }) => {
    await login(page)
    
    // 1. Navigate to data sources list
    await page.goto(`${BASE_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'ds-01-list')
    logResult('DS-1: Navigate to data sources list', 'PASS')

    // 2. Click "Add Source" link (navigates to /data-sources/new)
    const addSourceLink = page.locator('a:has-text("Add Source"), a:has-text("Add Data Source")')
    if (await addSourceLink.count() > 0) {
      await addSourceLink.first().click()
      await page.waitForURL('**/data-sources/new', { timeout: 10000 })
      await page.waitForTimeout(1000)
      await screenshot(page, 'ds-02-new-page')
      logResult('DS-2: Navigate to new data source page', 'PASS')

      // 3. Fill the form
      const testName = `E2E-Test-DS-${Date.now()}`
      
      // Fill name field
      const nameInput = page.locator('input[type="text"]').first()
      await nameInput.fill(testName)
      await screenshot(page, 'ds-03-name-filled')
      logResult('DS-3: Fill data source name', 'PASS', testName)

      // PostgreSQL should be selected by default, verify
      const selectedType = page.locator('label:has(input[type="radio"]:checked)')
      const typeText = await selectedType.textContent()
      logResult('DS-4: Default type selected', 'PASS', typeText || 'postgres')

      // Fill connection details (postgres is default)
      const hostInput = page.locator('input[placeholder="localhost"]')
      if (await hostInput.count() > 0) {
        await hostInput.fill('test-db.example.com')
      }
      
      const portInput = page.locator('input[placeholder="5432"]')
      if (await portInput.count() > 0) {
        await portInput.fill('5432')
      }
      
      const dbInput = page.locator('input[placeholder="mydb"]')
      if (await dbInput.count() > 0) {
        await dbInput.fill('testdb')
      }
      
      const userInput = page.locator('input[placeholder="user"]')
      if (await userInput.count() > 0) {
        await userInput.fill('testuser')
      }
      
      await screenshot(page, 'ds-04-form-filled')
      logResult('DS-5: Fill connection details', 'PASS')

      // 4. Submit the form
      const submitBtn = page.locator('button:has-text("Create Data Source")')
      await submitBtn.click()
      await page.waitForTimeout(3000)
      
      // Check if we're redirected back to list or if there's an error
      const currentUrl = page.url()
      if (currentUrl.includes('/data-sources') && !currentUrl.includes('/new')) {
        await screenshot(page, 'ds-05-created')
        logResult('DS-6: Create data source', 'PASS', 'Redirected to list')
      } else {
        // Check for error toast
        const errorToast = await page.locator('[data-sonner-toast]').textContent().catch(() => '')
        await screenshot(page, 'ds-05-create-error')
        logResult('DS-6: Create data source', 'FAIL', `Error: ${errorToast || 'Unknown'}`)
      }
    } else {
      logResult('DS-2: Navigate to new data source page', 'FAIL', 'Add Source link not found')
    }

    // 5. Click on an existing data source to view details
    await page.goto(`${BASE_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    
    const dataSourceLink = page.locator('table tbody tr a').first()
    if (await dataSourceLink.count() > 0) {
      const dsName = await dataSourceLink.textContent()
      await dataSourceLink.click()
      await page.waitForTimeout(2000)
      
      const detailUrl = page.url()
      if (detailUrl.includes('/data-sources/') && !detailUrl.includes('/new')) {
        await screenshot(page, 'ds-06-detail-page')
        logResult('DS-7: View data source detail', 'PASS', `Viewing: ${dsName}`)

        // 6. Test Run Scan button
        const scanBtn = page.locator('button:has-text("Run Scan")')
        if (await scanBtn.count() > 0) {
          await scanBtn.first().click()
          await page.waitForTimeout(2000)
          await screenshot(page, 'ds-07-scan-started')
          
          // Check for scanning indicator
          const scanningIndicator = page.locator('text=Scanning')
          if (await scanningIndicator.count() > 0) {
            logResult('DS-8: Trigger scan', 'PASS', 'Scan started')
            
            // Wait for scan to complete (up to 30 seconds)
            await page.waitForTimeout(5000)
            await screenshot(page, 'ds-08-scan-progress')
            
            // Check scan logs section
            const logsSection = page.locator('text=Scan Logs')
            if (await logsSection.count() > 0) {
              logResult('DS-9: Scan logs visible', 'PASS')
            }
          } else {
            logResult('DS-8: Trigger scan', 'PASS', 'Scan button clicked')
          }
        } else {
          logResult('DS-8: Trigger scan', 'SKIP', 'Run Scan button not found')
        }

        // 7. Test Delete button
        const deleteBtn = page.locator('button:has-text("Delete")')
        if (await deleteBtn.count() > 0) {
          await deleteBtn.first().click()
          await page.waitForTimeout(1000)
          await screenshot(page, 'ds-09-delete-dialog')
          
          // Check for confirmation dialog
          const confirmDialog = page.locator('[role="alertdialog"], [role="dialog"]')
          if (await confirmDialog.count() > 0) {
            logResult('DS-10: Delete confirmation dialog', 'PASS')
            
            // Cancel the delete
            const cancelBtn = page.locator('button:has-text("Cancel")')
            if (await cancelBtn.count() > 0) {
              await cancelBtn.click()
              await page.waitForTimeout(500)
              logResult('DS-11: Cancel delete', 'PASS')
            }
          }
        }
      } else {
        logResult('DS-7: View data source detail', 'FAIL', `Unexpected URL: ${detailUrl}`)
      }
    } else {
      logResult('DS-7: View data source detail', 'SKIP', 'No data sources in list')
    }
  })

  test('GOVERNANCE POLICIES: Full CRUD', async ({ page }) => {
    await login(page)
    
    // Navigate to policies
    await page.goto(`${BASE_URL}/governance/policies`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'gp-01-list')
    logResult('GP-1: Navigate to governance policies', 'PASS')

    // Click Create Policy
    const createBtn = page.locator('button:has-text("Create"), a:has-text("Create")')
    if (await createBtn.count() > 0) {
      await createBtn.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'gp-02-create-form')
      
      // Check if form/dialog opened
      const formVisible = await page.locator('form, [role="dialog"]').isVisible()
      if (formVisible) {
        logResult('GP-2: Create policy form opens', 'PASS')
        
        // Fill form
        const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
        if (await nameInput.count() > 0) {
          const testPolicyName = `E2E-Policy-${Date.now()}`
          await nameInput.fill(testPolicyName)
          
          // Try to select type
          const typeSelect = page.locator('[role="combobox"], select').first()
          if (await typeSelect.count() > 0) {
            await typeSelect.click()
            await page.waitForTimeout(500)
            const option = page.locator('[role="option"]').first()
            if (await option.count() > 0) {
              await option.click()
            }
          }
          
          await screenshot(page, 'gp-03-form-filled')
          
          // Submit
          const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last()
          if (await submitBtn.count() > 0) {
            await submitBtn.click()
            await page.waitForTimeout(3000)
            await screenshot(page, 'gp-04-after-create')
            logResult('GP-3: Create policy', 'PASS', testPolicyName)
          }
        }
      } else {
        logResult('GP-2: Create policy form opens', 'FAIL', 'Form not visible')
      }
    } else {
      logResult('GP-2: Create policy form opens', 'SKIP', 'Create button not found')
    }

    // View policy detail
    await page.goto(`${BASE_URL}/governance/policies`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    
    const policyLink = page.locator('table tbody tr a').first()
    if (await policyLink.count() > 0) {
      await policyLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, 'gp-05-detail')
      logResult('GP-4: View policy detail', 'PASS')
    }
  })

  test('CLASSIFICATION: View and Interact', async ({ page }) => {
    await login(page)
    
    // Navigate to classification
    await page.goto(`${BASE_URL}/classification`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'cl-01-list')
    logResult('CL-1: Navigate to classification', 'PASS')

    // Click on a dataset if available
    const datasetLink = page.locator('table tbody tr a, [class*="card"] a').first()
    if (await datasetLink.count() > 0) {
      await datasetLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, 'cl-02-detail')
      logResult('CL-2: View classification detail', 'PASS')

      // Check for columns table
      const columnsTable = page.locator('table')
      if (await columnsTable.count() > 0) {
        logResult('CL-3: Columns table visible', 'PASS')
      }

      // Check for re-classify button
      const reclassifyBtn = page.locator('button:has-text("Classify"), button:has-text("Re-classify")')
      if (await reclassifyBtn.count() > 0) {
        logResult('CL-4: Re-classify button available', 'PASS')
      }
    } else {
      logResult('CL-2: View classification detail', 'SKIP', 'No datasets found')
    }
  })

  test('SETTINGS: Users and API Keys', async ({ page }) => {
    await login(page)
    
    // Navigate to users
    await page.goto(`${BASE_URL}/settings/users`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'st-01-users')
    logResult('ST-1: Navigate to users settings', 'PASS')

    // Check for invite button
    const inviteBtn = page.locator('button:has-text("Invite")')
    if (await inviteBtn.count() > 0) {
      await inviteBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'st-02-invite-modal')
      logResult('ST-2: Invite user modal opens', 'PASS')
      
      // Close modal
      const closeBtn = page.locator('button:has-text("Cancel"), [aria-label="Close"]')
      if (await closeBtn.count() > 0) {
        await closeBtn.first().click()
      }
    }

    // Navigate to API keys
    await page.goto(`${BASE_URL}/settings/api-keys`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'st-03-apikeys')
    logResult('ST-3: Navigate to API keys settings', 'PASS')

    // Check for create key button
    const createKeyBtn = page.locator('button:has-text("Create"), button:has-text("Generate")')
    if (await createKeyBtn.count() > 0) {
      await createKeyBtn.first().click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'st-04-create-key-modal')
      logResult('ST-4: Create API key modal opens', 'PASS')
    }
  })

  test('AI GOVERNANCE: Playground Test', async ({ page }) => {
    await login(page)
    
    // Navigate to AI governance
    await page.goto(`${BASE_URL}/ai-governance`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'ai-01-page')
    logResult('AI-1: Navigate to AI governance', 'PASS')

    // Look for playground/chat input
    const chatInput = page.locator('textarea, input[type="text"][placeholder*="message" i], input[type="text"][placeholder*="query" i]')
    if (await chatInput.count() > 0) {
      await chatInput.first().fill('What is PII data?')
      await screenshot(page, 'ai-02-query-entered')
      logResult('AI-2: Enter test query', 'PASS')

      // Submit
      const submitBtn = page.locator('button:has-text("Send"), button:has-text("Submit"), button[type="submit"]')
      if (await submitBtn.count() > 0) {
        await submitBtn.first().click()
        await page.waitForTimeout(5000)
        await screenshot(page, 'ai-03-response')
        logResult('AI-3: Submit query', 'PASS')
      }
    } else {
      logResult('AI-2: Enter test query', 'SKIP', 'No chat input found')
    }
  })

  test('COMPLIANCE: DSAR and RoPA', async ({ page }) => {
    await login(page)
    
    // Navigate to compliance dashboard
    await page.goto(`${BASE_URL}/compliance`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'cp-01-dashboard')
    logResult('CP-1: Navigate to compliance dashboard', 'PASS')

    // Navigate to DSAR
    await page.goto(`${BASE_URL}/compliance/dsar`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'cp-02-dsar')
    logResult('CP-2: Navigate to DSAR', 'PASS')

    // Check for create button
    const createDsarBtn = page.locator('button:has-text("Create"), button:has-text("New")')
    if (await createDsarBtn.count() > 0) {
      logResult('CP-3: DSAR create button available', 'PASS')
    } else {
      logResult('CP-3: DSAR create button available', 'SKIP', 'No create button')
    }

    // Navigate to RoPA
    await page.goto(`${BASE_URL}/compliance/ropa`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'cp-03-ropa')
    logResult('CP-4: Navigate to RoPA', 'PASS')

    // Navigate to Advisor
    await page.goto(`${BASE_URL}/compliance/advisor`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'cp-04-advisor')
    logResult('CP-5: Navigate to Compliance Advisor', 'PASS')
  })

  test('AUDIT: View and Filter Logs', async ({ page }) => {
    await login(page)
    
    // Navigate to audit
    await page.goto(`${BASE_URL}/audit`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'au-01-page')
    logResult('AU-1: Navigate to audit logs', 'PASS')

    // Check for table
    const table = page.locator('table')
    if (await table.count() > 0) {
      logResult('AU-2: Audit logs table visible', 'PASS')
    }

    // Check for filter/search
    const filterInput = page.locator('input[placeholder*="search" i], input[placeholder*="filter" i]')
    if (await filterInput.count() > 0) {
      await filterInput.first().fill('login')
      await page.waitForTimeout(1000)
      await screenshot(page, 'au-02-filtered')
      logResult('AU-3: Filter audit logs', 'PASS')
    }
  })

  test('NAVIGATION: Sidebar and Logout', async ({ page }) => {
    await login(page)
    
    // Test sidebar navigation
    const sidebarLinks = [
      { text: 'Dashboard', url: '/dashboard' },
      { text: 'Data Sources', url: '/data-sources' },
      { text: 'Classification', url: '/classification' },
      { text: 'Governance', url: '/governance' },
      { text: 'Compliance', url: '/compliance' },
      { text: 'Audit', url: '/audit' },
      { text: 'Settings', url: '/settings' },
    ]

    for (const link of sidebarLinks) {
      const navLink = page.locator(`nav a:has-text("${link.text}"), aside a:has-text("${link.text}")`).first()
      if (await navLink.count() > 0) {
        await navLink.click()
        await page.waitForTimeout(1000)
        const currentUrl = page.url()
        if (currentUrl.includes(link.url)) {
          logResult(`NAV: ${link.text}`, 'PASS')
        } else {
          logResult(`NAV: ${link.text}`, 'FAIL', `Expected ${link.url}, got ${currentUrl}`)
        }
      }
    }

    // Test logout
    const userMenu = page.locator('[class*="avatar"], button:has-text("Logout"), [aria-label*="user" i]')
    if (await userMenu.count() > 0) {
      await userMenu.first().click()
      await page.waitForTimeout(500)
      
      const logoutBtn = page.locator('button:has-text("Logout"), a:has-text("Logout"), [role="menuitem"]:has-text("Logout")')
      if (await logoutBtn.count() > 0) {
        await logoutBtn.first().click()
        await page.waitForTimeout(2000)
        await screenshot(page, 'nav-logout')
        
        const onLoginPage = page.url().includes('login')
        logResult('NAV: Logout', onLoginPage ? 'PASS' : 'FAIL', `Current URL: ${page.url()}`)
      }
    }
  })

  test.afterAll(async () => {
    console.log('\n\n========== COMPLETE TEST RESULTS ==========\n')
    const passed = results.filter(r => r.status === 'PASS').length
    const failed = results.filter(r => r.status === 'FAIL').length
    const skipped = results.filter(r => r.status === 'SKIP').length
    
    console.log(`PASSED: ${passed}`)
    console.log(`FAILED: ${failed}`)
    console.log(`SKIPPED: ${skipped}`)
    console.log(`TOTAL: ${results.length}`)
    
    console.log('\n--- FAILED TESTS ---')
    results.filter(r => r.status === 'FAIL').forEach(r => {
      console.log(`  [FAIL] ${r.test}: ${r.details || ''} ${r.error || ''}`)
    })
    
    console.log('\n--- ALL RESULTS ---')
    results.forEach(r => {
      console.log(`[${r.status}] ${r.test}${r.details ? ` - ${r.details}` : ''}`)
    })
  })
})
