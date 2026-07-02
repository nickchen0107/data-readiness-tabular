import { type Page, expect } from '@playwright/test'
import path from 'path'

export const TEST_FILE_PATH = path.resolve(__dirname, '../fixtures/test-data.xlsx')

/**
 * Upload the test Excel file on the Upload page.
 * Assumes user is already logged in. Navigates to /upload if needed.
 */
export async function uploadTestFile(page: Page): Promise<void> {
  const url = page.url()
  if (!url.includes('/upload')) {
    await page.goto('/upload')
  }

  const fileInput = page.locator('input[type="file"]')
  await fileInput.setInputFiles(TEST_FILE_PATH)

  // Wait for file info to appear (filename visible on page)
  await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })
}

/**
 * Select a sheet (the first available) after file upload.
 * If there's only one sheet it's auto-selected, so this is a no-op.
 */
export async function selectSheet(page: Page): Promise<void> {
  // Wait a moment for sheet buttons to render
  await page.waitForTimeout(500)

  // If sheet buttons are visible (multiple sheets), click the first one
  const sheetBtn = page.locator('button:has-text("Sheet")').first()
  const isVisible = await sheetBtn.isVisible({ timeout: 2000 }).catch(() => false)
  if (isVisible) {
    await sheetBtn.click()
    await page.waitForTimeout(300)
  }
}

/**
 * Combined helper: upload file, select sheet, start assessment, wait for redirect.
 * Call this when you need to go through the full upload→assess flow.
 */
export async function selectSheetAndStartAssessment(page: Page): Promise<void> {
  // Wait for file info to be visible (assumes file is already uploaded)
  await page.waitForSelector('text=test-data.xlsx', { timeout: 15000 })

  // Select sheet if multiple sheets are shown
  await selectSheet(page)

  // Wait for button to be enabled
  const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
  await expect(startBtn).toBeEnabled({ timeout: 5000 })

  // Click start assessment
  await startBtn.click()

  // Wait for redirect to assessment page
  await page.waitForURL(/\/assessment/, { timeout: 30000 })
}

/**
 * Full flow: navigate to upload, upload file, select sheet, start assessment.
 */
export async function uploadAndStartAssessment(page: Page): Promise<void> {
  await page.goto('/upload')
  await uploadTestFile(page)
  await selectSheetAndStartAssessment(page)
}

/**
 * Start assessment after uploading a file (legacy helper).
 */
export async function startAssessment(page: Page): Promise<void> {
  await selectSheet(page)
  const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
  await expect(startBtn).toBeEnabled({ timeout: 5000 })
  await startBtn.click()
  await page.waitForURL('**/assessment**', { timeout: 30000 })
}
