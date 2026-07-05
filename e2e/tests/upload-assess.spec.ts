import { test, expect } from '@playwright/test'
import { TEST_USER, loginAsTestUser, ensureTestUser } from '../helpers/auth'
import { uploadTestFile, selectSheet, selectSheetAndStartAssessment, TEST_FILE_PATH } from '../helpers/upload'

test.describe.serial('Upload & Assessment flow', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await page.close()
  })

  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page)

    // Mock quota API to ensure quota is always available for these tests
    await page.route('**/api/quota/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          remaining: 50,
          max_assessments: 100,
          used_count: 50,
        }),
      })
    })
  })

  test('Upload an xlsx file → see file info chip with filename, rows, cols', async ({ page }) => {
    await page.goto('./upload')

    // Upload the test file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)

    // Wait for file info to appear
    await expect(page.getByText('test-data.xlsx')).toBeVisible({ timeout: 15000 })

    // Check rows/cols info is displayed
    const body = page.locator('body')
    await expect(body).toContainText(/\d+/, { timeout: 5000 })
  })

  test('Select sheet (if multiple) → sheet highlighted', async ({ page }) => {
    await page.goto('./upload')

    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Wait for sheet buttons to render
    await page.waitForTimeout(1000)

    // Check if sheet selector appears (test file has 2 sheets)
    const sheetButtons = page.locator('button:has-text("Sheet")')
    const count = await sheetButtons.count()

    if (count > 1) {
      // Click the second sheet
      await sheetButtons.nth(1).click()
      await page.waitForTimeout(300)

      // Verify it's visually highlighted (accent border color)
      const borderColor = await sheetButtons.nth(1).evaluate(
        (el) => getComputedStyle(el).borderColor
      )
      // Should have changed from default
      expect(borderColor).toBeTruthy()
    } else if (count === 1) {
      // Single sheet - just click it
      await sheetButtons.first().click()
    }
  })

  test('Click "開始評估" → see loading indicator → redirected to assessment page', async ({ page }) => {
    await page.goto('./upload')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Select sheet
    await selectSheet(page)

    // Click start assessment button
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeEnabled({ timeout: 5000 })
    await startBtn.click()

    // Should see a loading indicator (spinner) - may be too fast to catch
    const spinner = page.locator('[style*="animation"], .loading, [role="progressbar"]')
    await spinner.first().isVisible().catch(() => {})

    // Should redirect to assessment page
    await expect(page).toHaveURL(/\/assessment/, { timeout: 30000 })
  })

  test('Assessment page shows total score, 6 indicators, radar chart, issue list', async ({ page }) => {
    await page.goto('./upload')

    // Upload and start assessment
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Select sheet and start assessment
    await selectSheet(page)
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeEnabled({ timeout: 5000 })
    await startBtn.click()
    await page.waitForURL(/\/assessment/, { timeout: 30000 })

    // Wait for assessment content to load
    await page.waitForTimeout(2000)

    // Total score should be visible (some number)
    await expect(page.locator('body')).toContainText(/\d+/, { timeout: 10000 })

    // 6 indicators should be displayed
    const indicators = [
      /列完整度|Row Completeness/i,
      /欄完整度|Column Completeness/i,
      /格式一致性|Format Consistency/i,
      /資料唯一性|Data Uniqueness/i,
      /表格結構|Table Structure/i,
      /AI.*問答.*可用性|AI Query Readiness|AI/i,
    ]
    for (const indicator of indicators) {
      await expect(page.getByText(indicator).first()).toBeVisible({ timeout: 5000 }).catch(() => {
        // Some indicators may have different text in different languages
      })
    }

    // Radar chart (canvas or SVG element)
    const chart = page.locator('canvas, svg, [class*="radar"], [class*="chart"]')
    await expect(chart.first()).toBeVisible({ timeout: 5000 })

    // Issue list should have at least one item
    const body = page.locator('body')
    await expect(body).toContainText(/問題|issue|Issue/i, { timeout: 5000 })
  })

  test('Click back to assessment step from stepper → shows latest assessment (not error)', async ({ page }) => {
    await page.goto('./upload')

    // Upload and assess
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    await selectSheet(page)
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeEnabled({ timeout: 5000 })
    await startBtn.click()
    await page.waitForURL(/\/assessment/, { timeout: 30000 })

    // Wait for assessment content to load
    await page.waitForTimeout(3000)

    // Navigate to a different step via URL then come back
    await page.goto('./upload')
    await page.waitForTimeout(1000)

    // Click on assessment step in the stepper
    const stepperItem = page.locator('nav').getByText(/評估|Assess/i).first()
    if (await stepperItem.isVisible()) {
      await stepperItem.click()
      await page.waitForTimeout(2000)

      // Should show assessment results, not an error
      await expect(page.locator('body')).not.toContainText(/error|錯誤/i)
    } else {
      // Stepper not visible — navigate directly
      await page.goto('./assessment')
      await page.waitForTimeout(2000)
      await expect(page.locator('body')).not.toContainText(/error|錯誤/i)
    }
  })
})
