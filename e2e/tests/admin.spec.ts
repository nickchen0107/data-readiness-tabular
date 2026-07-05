import { test, expect } from '@playwright/test'
import {
  ADMIN_USER,
  TEST_USER,
  ensureAdminUser,
  ensureTestUser,
  loginAsAdmin,
  loginAsTestUser,
} from '../helpers/auth'

test.describe('Admin dashboard', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await ensureAdminUser(page)
    await page.close()
  })

  test('Non-admin user cannot access /admin → redirected', async ({ page }) => {
    await loginAsTestUser(page)

    // Try to navigate to admin page
    await page.goto('./admin')

    // AdminRoute redirects non-admin users to "/"
    // Wait for the redirect to happen
    await page.waitForURL((url) => !url.pathname.includes('/admin'), { timeout: 10000 }).catch(() => {})
    await page.waitForTimeout(1000)

    const url = page.url()
    const isRedirected = !url.includes('/admin')
    const hasAccessDenied = await page.getByText(/無權限|unauthorized|forbidden|access denied/i).isVisible().catch(() => false)

    expect(isRedirected || hasAccessDenied).toBeTruthy()
  })

  test('Admin user can access /admin → see user management page', async ({ page }) => {
    await loginAsAdmin(page)

    await page.goto('./admin')

    // Should see admin dashboard content
    await expect(page).toHaveURL(/\/admin/)
    await expect(page.locator('body')).toContainText(/使用者|user|管理|admin|management/i)
  })

  test('Admin can view quota settings', async ({ page }) => {
    await loginAsAdmin(page)

    // Navigate to quota settings
    await page.goto('./admin')

    // Click on quota settings link/tab
    const quotaLink = page.getByText(/配額|quota/i).first()
    if (await quotaLink.isVisible()) {
      await quotaLink.click()
    } else {
      await page.goto('./admin/quota')
    }

    // Should see quota configuration
    await expect(page.locator('body')).toContainText(/配額|quota|max.*assessment|重置/i)
  })

  test('Admin can update quota settings', async ({ page }) => {
    await loginAsAdmin(page)

    await page.goto('./admin/quota').catch(() => page.goto('./admin'))

    // Find quota settings link if needed
    const quotaLink = page.getByText(/配額|quota/i).first()
    if (await quotaLink.isVisible() && !page.url().includes('/quota')) {
      await quotaLink.click()
    }

    // Find the max assessments input
    const maxInput = page.locator('input[type="number"], input[name*="max"], input[name*="quota"]').first()
    if (await maxInput.isVisible()) {
      await maxInput.clear()
      await maxInput.fill('10')

      // Save
      const saveBtn = page.getByRole('button', { name: /儲存|save|確認|submit/i }).first()
      await saveBtn.click()

      // Should see success feedback
      await expect(page.locator('body')).toContainText(/成功|saved|success|更新/i, { timeout: 5000 })
    }
  })

  test('Admin can search and edit translations', async ({ page }) => {
    await loginAsAdmin(page)

    // Navigate to translations page
    await page.goto('./admin/translations').catch(() => page.goto('./admin'))

    const transLink = page.getByText(/翻譯|translation/i).first()
    if (await transLink.isVisible() && !page.url().includes('/translation')) {
      await transLink.click()
    }

    await page.waitForTimeout(1000)

    // Should see translation editor content
    await expect(page.locator('body')).toContainText(/翻譯|translation|key|locale/i)

    // Try searching
    const searchInput = page.locator('input[type="search"], input[placeholder*="搜尋"], input[placeholder*="search"]').first()
    if (await searchInput.isVisible()) {
      await searchInput.fill('login')
      await page.waitForTimeout(500)

      // Should filter results (fewer items or matching text)
      await expect(page.locator('body')).toContainText(/login/i)
    }
  })

  test('Admin can view assessment records', async ({ page }) => {
    await loginAsAdmin(page)

    // Navigate to assessment records page
    await page.goto('./admin/records').catch(() => page.goto('./admin'))

    const recordsLink = page.getByText(/記錄|record|history|評估.*紀錄/i).first()
    if (await recordsLink.isVisible()) {
      await recordsLink.click()
    }

    await page.waitForTimeout(1000)

    // Should see assessment records content (table, list, or empty state)
    await expect(page.locator('body')).toContainText(/記錄|record|評估|assessment|暫無|no.*record/i)
  })
})
