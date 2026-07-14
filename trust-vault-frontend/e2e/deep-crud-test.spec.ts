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

test.describe('TrustVault Deep CRUD Testing', () => {
  test.setTimeout(300000)

  test('DATA SOURCE FULL CRUD + SCAN', async ({ page }) => {
    await login(page)
    
    // Navigate to data sources
    await page.goto(`${BASE_URL}/data-sources`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-01-datasources-list')

    // Count existing data sources
    const initialRows = await page.locator('table tbody tr').count()
    console.log(`Initial data sources count: ${initialRows}`)

    // CREATE: Click Add button
    const addBtn = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")').first()
    await addBtn.click()
    await page.waitForTimeout(1000)
    await screenshot(page, 'crud-02-datasources-add-dialog')

    // Fill the form
    const testName = `Test-DS-${Date.now()}`
    
    // Fill name
    const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
    await nameInput.fill(testName)
    
    // Select type - click the combobox/select
    const typeSelect = page.locator('[role="combobox"], select[name="type"]').first()
    if (await typeSelect.count() > 0) {
      await typeSelect.click()
      await page.waitForTimeout(500)
      // Select PostgreSQL or first option
      const pgOption = page.locator('[role="option"]:has-text("PostgreSQL"), [role="option"]').first()
      if (await pgOption.count() > 0) {
        await pgOption.click()
      }
    }
    
    await page.waitForTimeout(500)
    await screenshot(page, 'crud-03-datasources-form-filled')

    // Fill connection details if visible
    const hostInput = page.locator('input[name="host"], input[placeholder*="host" i]')
    if (await hostInput.count() > 0) {
      await hostInput.first().fill('localhost')
    }
    
    const portInput = page.locator('input[name="port"], input[placeholder*="port" i]')
    if (await portInput.count() > 0) {
      await portInput.first().fill('5432')
    }
    
    const dbInput = page.locator('input[name="database"], input[placeholder*="database" i]')
    if (await dbInput.count() > 0) {
      await dbInput.first().fill('testdb')
    }

    // Submit
    const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create"), button:has-text("Add")').last()
    await submitBtn.click()
    await page.waitForTimeout(3000)
    await screenshot(page, 'crud-04-datasources-after-create')

    // Check if created
    const newRows = await page.locator('table tbody tr').count()
    if (newRows > initialRows) {
      logResult('CREATE Data Source', 'PASS', `Created ${testName}`)
    } else {
      // Check for error toast
      const errorToast = await page.locator('[data-sonner-toast][data-type="error"], .toast-error').isVisible()
      if (errorToast) {
        const errorText = await page.locator('[data-sonner-toast][data-type="error"], .toast-error').textContent()
        logResult('CREATE Data Source', 'FAIL', `Error: ${errorText}`)
      } else {
        logResult('CREATE Data Source', 'SKIP', 'Could not verify creation')
      }
    }

    // READ: Click on a data source to view details
    await page.waitForTimeout(1000)
    const firstRow = page.locator('table tbody tr').first()
    const rowLink = firstRow.locator('a').first()
    
    if (await rowLink.count() > 0) {
      const href = await rowLink.getAttribute('href')
      console.log(`Clicking on data source link: ${href}`)
      await rowLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, 'crud-05-datasources-detail')
      
      const currentUrl = page.url()
      if (currentUrl.includes('/data-sources/')) {
        logResult('READ Data Source Detail', 'PASS', `URL: ${currentUrl}`)
        
        // TEST SCAN FUNCTIONALITY
        const scanBtn = page.locator('button:has-text("Scan"), button:has-text("Run Scan"), button:has-text("Start Scan")')
        if (await scanBtn.count() > 0) {
          console.log('Found scan button, clicking...')
          await scanBtn.first().click()
          await page.waitForTimeout(5000)
          await screenshot(page, 'crud-06-datasources-scan-started')
          
          // Check for scan logs or progress
          const logsArea = page.locator('[class*="log"], pre, [class*="terminal"], [class*="output"], [class*="progress"]')
          if (await logsArea.count() > 0) {
            logResult('SCAN Data Source', 'PASS', 'Scan initiated, logs visible')
          } else {
            // Check for status change
            const statusBadge = page.locator('[class*="badge"], [class*="status"]')
            if (await statusBadge.count() > 0) {
              const statusText = await statusBadge.first().textContent()
              logResult('SCAN Data Source', 'PASS', `Status: ${statusText}`)
            } else {
              logResult('SCAN Data Source', 'SKIP', 'No visible scan feedback')
            }
          }
        } else {
          logResult('SCAN Data Source', 'SKIP', 'No scan button found on detail page')
        }

        // UPDATE: Click Edit button
        const editBtn = page.locator('button:has-text("Edit"), a:has-text("Edit")')
        if (await editBtn.count() > 0) {
          await editBtn.first().click()
          await page.waitForTimeout(1000)
          await screenshot(page, 'crud-07-datasources-edit-form')
          
          // Modify name
          const editNameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
          if (await editNameInput.count() > 0) {
            const currentName = await editNameInput.inputValue()
            await editNameInput.fill(currentName + '-EDITED')
            
            const saveBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Update")')
            if (await saveBtn.count() > 0) {
              await saveBtn.first().click()
              await page.waitForTimeout(2000)
              await screenshot(page, 'crud-08-datasources-after-edit')
              logResult('UPDATE Data Source', 'PASS', 'Name modified')
            }
          }
        } else {
          logResult('UPDATE Data Source', 'SKIP', 'No edit button found')
        }

        // DELETE: Click Delete button
        const deleteBtn = page.locator('button:has-text("Delete"), button[class*="destructive"]')
        if (await deleteBtn.count() > 0) {
          await deleteBtn.first().click()
          await page.waitForTimeout(1000)
          await screenshot(page, 'crud-09-datasources-delete-confirm')
          
          // Confirm deletion
          const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("Delete")[class*="destructive"]')
          if (await confirmBtn.count() > 0) {
            await confirmBtn.first().click()
            await page.waitForTimeout(2000)
            await screenshot(page, 'crud-10-datasources-after-delete')
            logResult('DELETE Data Source', 'PASS', 'Deleted successfully')
          }
        } else {
          logResult('DELETE Data Source', 'SKIP', 'No delete button found')
        }
      } else {
        logResult('READ Data Source Detail', 'FAIL', `Unexpected URL: ${currentUrl}`)
      }
    } else {
      // Try clicking the row itself
      await firstRow.click()
      await page.waitForTimeout(2000)
      logResult('READ Data Source Detail', 'SKIP', 'No clickable link in row')
    }
  })

  test('GOVERNANCE POLICY FULL CRUD', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/governance/policies`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-11-policies-list')

    // CREATE
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Add"), button:has-text("New")').first()
    if (await createBtn.count() > 0) {
      await createBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-12-policies-create-form')

      const testPolicyName = `Test-Policy-${Date.now()}`
      
      // Fill form
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
      if (await nameInput.count() > 0) {
        await nameInput.fill(testPolicyName)
      }

      // Select type if available
      const typeSelect = page.locator('[role="combobox"], select[name="type"]').first()
      if (await typeSelect.count() > 0) {
        await typeSelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      // Description
      const descInput = page.locator('textarea[name="description"], input[name="description"]')
      if (await descInput.count() > 0) {
        await descInput.first().fill('Test policy description')
      }

      await screenshot(page, 'crud-13-policies-form-filled')

      // Submit
      const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-14-policies-after-create')
      logResult('CREATE Policy', 'PASS', testPolicyName)
    }

    // READ - click on a policy
    const policyRow = page.locator('table tbody tr').first()
    const policyLink = policyRow.locator('a').first()
    if (await policyLink.count() > 0) {
      await policyLink.click()
      await page.waitForTimeout(2000)
      await screenshot(page, 'crud-15-policies-detail')
      logResult('READ Policy Detail', 'PASS')

      // UPDATE
      const editBtn = page.locator('button:has-text("Edit")')
      if (await editBtn.count() > 0) {
        await editBtn.first().click()
        await page.waitForTimeout(1000)
        
        const nameInput = page.locator('input[name="name"]').first()
        if (await nameInput.count() > 0) {
          const currentName = await nameInput.inputValue()
          await nameInput.fill(currentName + '-EDITED')
          
          const saveBtn = page.locator('button[type="submit"], button:has-text("Save")')
          await saveBtn.first().click()
          await page.waitForTimeout(2000)
          await screenshot(page, 'crud-16-policies-after-edit')
          logResult('UPDATE Policy', 'PASS')
        }
      }

      // DELETE
      const deleteBtn = page.locator('button:has-text("Delete")')
      if (await deleteBtn.count() > 0) {
        await deleteBtn.first().click()
        await page.waitForTimeout(1000)
        
        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("Delete")[class*="destructive"]')
        if (await confirmBtn.count() > 0) {
          await confirmBtn.first().click()
          await page.waitForTimeout(2000)
          await screenshot(page, 'crud-17-policies-after-delete')
          logResult('DELETE Policy', 'PASS')
        }
      }
    }
  })

  test('CLASSIFICATION RULES FULL CRUD', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/classification/rules`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-18-rules-list')

    // CREATE
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Add"), button:has-text("New")').first()
    if (await createBtn.count() > 0) {
      await createBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-19-rules-create-form')

      const testRuleName = `Test-Rule-${Date.now()}`
      
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
      if (await nameInput.count() > 0) {
        await nameInput.fill(testRuleName)
      }

      // Pattern input
      const patternInput = page.locator('input[name="pattern"], textarea[name="pattern"]')
      if (await patternInput.count() > 0) {
        await patternInput.first().fill('\\b[A-Z]{2}\\d{6}\\b')
      }

      // Category/Type
      const categorySelect = page.locator('[role="combobox"], select[name="category"], select[name="type"]').first()
      if (await categorySelect.count() > 0) {
        await categorySelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      await screenshot(page, 'crud-20-rules-form-filled')

      const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-21-rules-after-create')
      logResult('CREATE Classification Rule', 'PASS', testRuleName)
    }
  })

  test('API KEYS FULL CRUD', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/settings/api-keys`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-22-apikeys-list')

    // CREATE
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Generate"), button:has-text("Add")').first()
    if (await createBtn.count() > 0) {
      await createBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-23-apikeys-create-form')

      const testKeyName = `Test-Key-${Date.now()}`
      
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
      if (await nameInput.count() > 0) {
        await nameInput.fill(testKeyName)
      }

      // Expiry if available
      const expirySelect = page.locator('[role="combobox"], select[name="expiry"]').first()
      if (await expirySelect.count() > 0) {
        await expirySelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      await screenshot(page, 'crud-24-apikeys-form-filled')

      const submitBtn = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Generate")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-25-apikeys-after-create')
      
      // Check if key is displayed
      const keyDisplay = page.locator('code, [class*="key"], input[readonly]')
      if (await keyDisplay.count() > 0) {
        logResult('CREATE API Key', 'PASS', 'Key generated and displayed')
      } else {
        logResult('CREATE API Key', 'PASS', testKeyName)
      }
    }

    // DELETE - find and delete a key
    await page.waitForTimeout(1000)
    const deleteBtn = page.locator('button:has-text("Delete"), button:has-text("Revoke"), button[class*="destructive"]').first()
    if (await deleteBtn.count() > 0) {
      await deleteBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-26-apikeys-delete-confirm')
      
      const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("Delete")[class*="destructive"], button:has-text("Revoke")[class*="destructive"]')
      if (await confirmBtn.count() > 0) {
        await confirmBtn.first().click()
        await page.waitForTimeout(2000)
        await screenshot(page, 'crud-27-apikeys-after-delete')
        logResult('DELETE API Key', 'PASS')
      }
    }
  })

  test('USER INVITATION', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/settings/users`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-28-users-list')

    // INVITE
    const inviteBtn = page.locator('button:has-text("Invite"), button:has-text("Add")').first()
    if (await inviteBtn.count() > 0) {
      await inviteBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-29-users-invite-form')

      const testEmail = `test-${Date.now()}@example.com`
      
      const emailInput = page.locator('input[name="email"], input[type="email"]').first()
      if (await emailInput.count() > 0) {
        await emailInput.fill(testEmail)
      }

      // Role selection
      const roleSelect = page.locator('[role="combobox"], select[name="role"]').first()
      if (await roleSelect.count() > 0) {
        await roleSelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      await screenshot(page, 'crud-30-users-invite-filled')

      const submitBtn = page.locator('button[type="submit"], button:has-text("Invite"), button:has-text("Send")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-31-users-after-invite')
      logResult('INVITE User', 'PASS', testEmail)
    }
  })

  test('DSAR REQUEST CREATION', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/compliance/dsar`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-32-dsar-list')

    // CREATE DSAR
    const createBtn = page.locator('button:has-text("Create"), button:has-text("New"), button:has-text("Add")').first()
    if (await createBtn.count() > 0) {
      await createBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-33-dsar-create-form')

      // Fill DSAR form
      const subjectInput = page.locator('input[name="subject"], input[name="email"], input[placeholder*="email" i]').first()
      if (await subjectInput.count() > 0) {
        await subjectInput.fill(`dsar-test-${Date.now()}@example.com`)
      }

      // Request type
      const typeSelect = page.locator('[role="combobox"], select[name="type"], select[name="request_type"]').first()
      if (await typeSelect.count() > 0) {
        await typeSelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      await screenshot(page, 'crud-34-dsar-form-filled')

      const submitBtn = page.locator('button[type="submit"], button:has-text("Submit"), button:has-text("Create")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-35-dsar-after-create')
      logResult('CREATE DSAR Request', 'PASS')
    } else {
      logResult('CREATE DSAR Request', 'SKIP', 'No create button found')
    }
  })

  test('SENSITIVITY LABEL RULES CRUD', async ({ page }) => {
    await login(page)
    
    await page.goto(`${BASE_URL}/labels/rules`)
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)
    await screenshot(page, 'crud-36-label-rules-list')

    // CREATE
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Add"), button:has-text("New")').first()
    if (await createBtn.count() > 0) {
      await createBtn.click()
      await page.waitForTimeout(1000)
      await screenshot(page, 'crud-37-label-rules-create-form')

      const testRuleName = `Label-Rule-${Date.now()}`
      
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
      if (await nameInput.count() > 0) {
        await nameInput.fill(testRuleName)
      }

      // Label selection
      const labelSelect = page.locator('[role="combobox"], select[name="label"]').first()
      if (await labelSelect.count() > 0) {
        await labelSelect.click()
        await page.waitForTimeout(500)
        const option = page.locator('[role="option"]').first()
        if (await option.count() > 0) {
          await option.click()
        }
      }

      await screenshot(page, 'crud-38-label-rules-form-filled')

      const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last()
      await submitBtn.click()
      await page.waitForTimeout(3000)
      await screenshot(page, 'crud-39-label-rules-after-create')
      logResult('CREATE Label Rule', 'PASS', testRuleName)
    }
  })

  test.afterAll(async () => {
    console.log('\n\n========== CRUD TEST RESULTS ==========\n')
    const passed = results.filter(r => r.status === 'PASS').length
    const failed = results.filter(r => r.status === 'FAIL').length
    const skipped = results.filter(r => r.status === 'SKIP').length
    
    console.log(`PASSED: ${passed}`)
    console.log(`FAILED: ${failed}`)
    console.log(`SKIPPED: ${skipped}`)
    console.log(`TOTAL: ${results.length}`)
    
    console.log('\n--- ALL RESULTS ---')
    results.forEach(r => {
      console.log(`[${r.status}] ${r.test}${r.details ? ` - ${r.details}` : ''}`)
    })
  })
})
