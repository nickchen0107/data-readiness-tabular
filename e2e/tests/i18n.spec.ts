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
    await page.goto('./login')
    await page.evaluate(() => {
      localStorage.removeItem('i18nextLng')
      localStorage.removeItem('language')
    })
  })

  test('Default language is English', async ({ page }) => {
    await loginAsTestUser(page)

    // Page should display English content by default
    await expect(page.locator('body')).toContainText(/Upload|Assessment|Clean|Export|Evidence/i)
  })

  test('Click language switcher → UI changes to English', async ({ page }) => {
    await loginAsTestUser(page)

    // The LanguageSwitcher shows "中" when in Chinese mode (clicking toggles to English)
    // It shows "EN" when in English mode
    const langSwitcher = page.locator('button').filter({ hasText: /^中$|^EN$/ }).first()
    await expect(langSwitcher).toBeVisible({ timeout: 5000 })
    await langSwitcher.click()

    // Wait for UI to update
    await page.waitForTimeout(1000)

    // Should see English text
    await expect(page.locator('body')).toContainText(/Upload|Assessment|Clean|Export|Dashboard/i)
  })

  test('Refresh page → language persisted (localStorage)', async ({ page }) => {
    await loginAsTestUser(page)

    // Default is English. Switch to Chinese by clicking the language toggle
    const langSwitcher = page.locator('button').filter({ hasText: /^中$|^EN$/ }).first()
    await expect(langSwitcher).toBeVisible({ timeout: 5000 })
    await langSwitcher.click()
    await page.waitForTimeout(1000)

    // Verify localStorage was set to 'zh-TW'
    const storedLang = await page.evaluate(() => {
      return localStorage.getItem('language') || localStorage.getItem('i18nextLng') || ''
    })
    expect(storedLang).toMatch(/zh/i)

    // Refresh page
    await page.reload()
    await page.waitForTimeout(2000)

    // Verify localStorage still has 'zh-TW' after reload
    const storedLangAfterReload = await page.evaluate(() => {
      return localStorage.getItem('language') || localStorage.getItem('i18nextLng') || ''
    })
    expect(storedLangAfterReload).toMatch(/zh/i)

    // Should still show Chinese
    await expect(page.locator('body')).toContainText(/上傳|梳理|評估|產出/, { timeout: 5000 })
  })

  test('Switch back to zh-TW → UI reverts', async ({ page }) => {
    await loginAsTestUser(page)

    // Set language to English first via localStorage
    await page.evaluate(() => {
      localStorage.setItem('i18nextLng', 'en')
      localStorage.setItem('language', 'en')
    })
    await page.reload()
    await page.waitForTimeout(1500)

    // Now switch back to Chinese - the button should show "EN" (current lang is English)
    const langSwitcher = page.locator('button').filter({ hasText: /^中$|^EN$/ }).first()
    await expect(langSwitcher).toBeVisible({ timeout: 5000 })
    await langSwitcher.click()
    await page.waitForTimeout(1000)

    // Should show Chinese content
    await expect(page.locator('body')).toContainText(/上傳|梳理|評估|產出/)
  })
})
