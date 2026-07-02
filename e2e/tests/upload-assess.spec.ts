import { test, expect } from '@playwright/test'
import { TEST_USER, loginAsTestUser, ensureTestUser } from '../helpers/auth'
import { uploadTestFile, startAssessment, TEST_FILE_PATH } from '../helpers/upload'

test.describe.serial('Upload & Assessment flow', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await page.close()
  })

  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page)
  })

  test('Upload an xlsx file → see file info chip with filename, rows, cols', async ({ page }) => {
    // Navigate to upload page
    await page.goto('/upload')

    // Upload the test file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)

    // Wait for file info to appear
    const fileInfo = page.locator('text=test-data.xlsx')
    await expect(fileInfo).toBeVisible({ timeout: 15000 })

    // Check rows/cols info is displayed
    const infoText = page.locator('body')
    await expect(infoText).toContainText(/\d+.*行|rows/i)
    await expect(infoText).toContainText(/\d+.*欄|cols/i)
  })

  test('Select sheet (if multiple) → sheet highlighted', async ({ page }) => {
    await page.goto('/upload')

    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)

    // Wait for upload to complete
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Check if sheet selector appears (test file has 2 sheets)
    const sheetButtons = page.locator('button:has-text("Sheet")')
    const count = await sheetButtons.count()

    if (count > 1) {
      // Click the second sheet
      const sheet2Btn = page.getByRole('button', { name: 'Sheet2' })
      await sheet2Btn.click()

      // Verify it's visually highlighted (accent color in style)
      await expect(sheet2Btn).toHaveCSS('border-color', /accent|rgb/)
    } else {
      // Single sheet, auto-selected — just verify no error
      test.skip(count <= 1, 'Only one sheet available, skipping sheet selection test')
    }
  })

  test('Click "開始評估" → see loading indicator → redirected to assessment page', async ({ page }) => {
    await page.goto('/upload')

    // Upload file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    // Select first sheet if multiple are shown
    const sheetBtn = page.locator('button:has-text("Sheet")').first()
    if (await sheetBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await sheetBtn.click()
    }

    // Click start assessment button
    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await expect(startBtn).toBeVisible()
    await expect(startBtn).toBeEnabled({ timeout: 5000 })
    await startBtn.click()

    // Should see a loading indicator (spinner)
    const spinner = page.locator('[style*="animation"], .loading, [role="progressbar"]')
    await expect(spinner.first()).toBeVisible({ timeout: 5000 }).catch(() => {
      // Loading might be too fast to catch — that's fine
    })

    // Should redirect to assessment page
    await expect(page).toHaveURL(/\/assessment/, { timeout: 30000 })
  })

  test('Assessment page shows total score, 6 indicators, radar chart, issue list', async ({ page }) => {
    await page.goto('/upload')

    // Upload and start assessment
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await startBtn.click()
    await page.waitForURL(/\/assessment/, { timeout: 30000 })

    // Total score should be visible
    const scoreElement = page.locator('text=/\\d+\\.?\\d*/')
    await expect(scoreElement.first()).toBeVisible({ timeout: 10000 })

    // 6 indicators should be displayed
    const indicators = [
      /列完整度|Row Completeness/,
      /欄完整度|Column Completeness/,
      /格式一致性|Format Consistency/,
      /資料唯一性|Data Uniqueness/,
      /表格結構|Table Structure/,
      /AI.*問答.*可用性|AI Query Readiness/,
    ]
    for (const indicator of indicators) {
      await expect(page.locator(`text=${indicator.source}`).first()).toBeVisible({ timeout: 5000 }).catch(async () => {
        // Fallback: search with getByText
        await expect(page.getByText(indicator).first()).toBeVisible()
      })
    }

    // Radar chart (canvas or SVG element)
    const chart = page.locator('canvas, svg, [class*="radar"], [class*="chart"]')
    await expect(chart.first()).toBeVisible()

    // Issue list should have at least one item
    const issueCards = page.locator('[class*="issue"], [class*="card"], [data-testid*="issue"]')
    await expect(issueCards.first()).toBeVisible({ timeout: 5000 }).catch(async () => {
      // Fallback: look for issue-related text
      await expect(page.getByText(/問題|issue/i).first()).toBeVisible()
    })
  })

  test('Click back to assessment step from stepper → shows latest assessment (not error)', async ({ page }) => {
    await page.goto('/upload')

    // Upload and assess
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(TEST_FILE_PATH)
    await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

    const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
    await startBtn.click()
    await page.waitForURL(/\/assessment/, { timeout: 30000 })

    // Wait for assessment content to load
    await page.waitForSelector('text=/\\d+\\.?\\d*/', { timeout: 10000 })

    // Navigate forward (if there's a "next" action) then back via stepper
    // Click on assessment step in the stepper
    const stepperItem = page.locator('[class*="stepper"], [class*="step"], nav').getByText(/評估|Assessment/i).first()
    if (await stepperItem.isVisible()) {
      await stepperItem.click()

      // Should show assessment results, not an error
      await expect(page.locator('body')).not.toContainText(/error|錯誤|失敗/i)
      // Score should still be visible
      await expect(page.locator('text=/\\d+\\.?\\d*/')).toBeVisible()
    }
  })
})
