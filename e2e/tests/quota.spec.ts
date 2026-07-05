import { test, expect } from '@playwright/test'
import { loginAsTestUser, ensureTestUser } from '../helpers/auth'
import { TEST_FILE_PATH, selectSheet } from '../helpers/upload'

test.describe('Quota enforcement', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await page.close()
  })

  test('User with remaining quota can start assessment', async ({ page }) => {
    await loginAsTestUser(page)

    // Mock the quota API to return remaining quota
    await page.route('**/api/quota/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          remaining: 10,
          max_assessments: 100,
          used_count: 90,
        }),
      })
    })

    await page.goto('./upload')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Select sheet
    await selectSheet(page)

    // The start assessment button should be enabled (not disabled)
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeVisible()
    await expect(startBtn).toBeEnabled({ timeout: 5000 })
  })

  test('User with exhausted quota sees disabled button + tooltip', async ({ page }) => {
    await loginAsTestUser(page)

    // Mock the quota API to return 0 remaining
    await page.route('**/api/quota/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          remaining: 0,
          max_assessments: 5,
          used_count: 5,
        }),
      })
    })

    await page.goto('./upload')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Select sheet
    await selectSheet(page)

    // The start assessment button should be disabled (quota exhausted)
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeVisible()
    await expect(startBtn).toBeDisabled({ timeout: 5000 })

    // Check for tooltip/title attribute on the button
    const hasTitle = await startBtn.getAttribute('title')
    expect(hasTitle).toBeTruthy()
  })
})
