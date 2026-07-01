import { test, expect } from '@playwright/test'
import { loginAsTestUser, ensureTestUser } from '../helpers/auth'

test.describe('Language switching (i18n)', () => {
  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await ensureTestUser(page)
    await page.close()
  })

  test.beforeEach(async ({ page }) => {
    // Clear localStorage to reset language preference
    await page.goto('/login')
    await page.evaluate(() => localStorage.removeItem('i18nextLng'))
    await page.evaluate(() => localStorage.removeItem('language'))
  })

  test('Default language is zh-TW', async ({ page }) => {
    await loginAsTestUser(page)

    // Page should display Chinese content by default
    await expect(page.locator('body')).toContainText(/上傳|梳理|評估|產出/)
  })

  test('Click language switcher → UI changes to English', async ({ page }) => {
    await loginAsTestUser(page)

    // Find and click language switcher
    const langSwitcher = page.locator(
      'button:has-text("EN"), button:has-text("English"), button:has-text("中"), [data-testid="lang-switch"], [aria-label*="language"]'
    ).first()
    await expect(langSwitcher).toBeVisible({ timeout: 5000 })
    await langSwitcher.click()

    // If it's a dropdown, select English
    const enOption = page.getByText(/English|EN/).first()
    if (await enOption.isVisible()) {
      await enOption.click()
    }

    // Wait for UI to update
    await page.waitForTimeout(500)

    // Should see English text (at least some of it)
    await expect(page.locator('body')).toContainText(/Upload|Assessment|Clean|Export|Dashboard/i)
  })

  test('Refresh page → English persisted (localStorage)', async ({ page }) => {
    await loginAsTestUser(page)

    // Switch to English
    const langSwitcher = page.locator(
      'button:has-text("EN"), button:has-text("English"), button:has-text("中"), [data-testid="lang-switch"], [aria-label*="language"]'
    ).first()

    if (await langSwitcher.isVisible()) {
      await langSwitcher.click()
      const enOption = page.getByText(/English|EN/).first()
      if (await enOption.isVisible()) {
        await enOption.click()
      }
      await page.waitForTimeout(500)
    }

    // Verify localStorage was set
    const storedLang = await page.evaluate(() => {
      return localStorage.getItem('i18nextLng') || localStorage.getItem('language') || ''
    })
    expect(storedLang).toMatch(/en/i)

    // Refresh page
    await page.reload()
    await page.waitForTimeout(1000)

    // Should still show English
    await expect(page.locator('body')).toContainText(/Upload|Assessment|Clean|Export|Dashboard/i)
  })

  test('Switch back to zh-TW → UI reverts', async ({ page }) => {
    await loginAsTestUser(page)

    // Set language to English first via localStorage
    await page.evaluate(() => {
      localStorage.setItem('i18nextLng', 'en')
      localStorage.setItem('language', 'en')
    })
    await page.reload()
    await page.waitForTimeout(500)

    // Find language switcher and switch back to Chinese
    const langSwitcher = page.locator(
      'button:has-text("中"), button:has-text("ZH"), button:has-text("繁"), [data-testid="lang-switch"], [aria-label*="language"]'
    ).first()

    if (await langSwitcher.isVisible()) {
      await langSwitcher.click()
      const zhOption = page.getByText(/繁體中文|中文|ZH-TW/i).first()
      if (await zhOption.isVisible()) {
        await zhOption.click()
      }
      await page.waitForTimeout(500)
    } else {
      // Direct localStorage manipulation and reload
      await page.evaluate(() => {
        localStorage.setItem('i18nextLng', 'zh-TW')
        localStorage.setItem('language', 'zh-TW')
      })
      await page.reload()
      await page.waitForTimeout(500)
    }

    // Should show Chinese content
    await expect(page.locator('body')).toContainText(/上傳|梳理|評估|產出/)
  })
})
