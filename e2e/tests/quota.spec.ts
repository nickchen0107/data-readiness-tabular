import { test, expect } from '@playwright/test'
import { loginAsTestUser, ensureTestUser } from '../helpers/auth'
import { TEST_FILE_PATH } from '../helpers/upload'

test.describe('Quota enforcement', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await page.close()
  })

  test('User with remaining quota can start assessment', async ({ page }) => {
    await loginAsTestUser(page)
    await page.goto('/landing')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // The start assessment button should be enabled (not disabled)
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeVisible()
    await expect(startBtn).toBeEnabled()

    // Click and verify assessment starts
    await startBtn.click()
    await expect(page).toHaveURL(/\/assessment/, { timeout: 30000 })
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

    await page.goto('/landing')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // The start assessment button should be disabled
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeVisible()
    await expect(startBtn).toBeDisabled()

    // Hover over the button to check for tooltip
    await startBtn.hover()
    await page.waitForTimeout(500)

    // Should show quota exhausted tooltip or title
    const tooltip = page.locator('[role="tooltip"], [class*="tooltip"]').first()
    const hasTooltip = await tooltip.isVisible().catch(() => false)
    const hasTitle = await startBtn.getAttribute('title')

    expect(hasTooltip || (hasTitle && hasTitle.includes('用盡'))).toBeTruthy()
  })
})
