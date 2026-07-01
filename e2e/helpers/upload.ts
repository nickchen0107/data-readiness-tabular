import { type Page } from '@playwright/test'
import path from 'path'

export const TEST_FILE_PATH = path.resolve(__dirname, '../fixtures/test-data.xlsx')

/**
 * Upload the test Excel file on the Upload page.
 * Assumes user is already logged in and on the landing/upload page.
 */
export async function uploadTestFile(page: Page): Promise<void> {
  // Navigate to the upload step if not already there
  const url = page.url()
  if (!url.includes('/upload') && !url.includes('/landing')) {
    await page.goto('/landing')
  }

  // Click the upload area or find the file input
  const fileInput = page.locator('input[type="file"]')
  await fileInput.setInputFiles(TEST_FILE_PATH)

  // Wait for upload to complete - file info chip appears
  await page.waitForSelector('[class*="pill"], .pill', { timeout: 15000 })
}

/**
 * Start assessment after uploading a file.
 */
export async function startAssessment(page: Page): Promise<void> {
  const startBtn = page.getByRole('button', { name: /開始評估|start.*assess/i })
  await startBtn.click()

  // Wait for navigation to assessment page
  await page.waitForURL('**/assessment**', { timeout: 30000 })
}
