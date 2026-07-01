import { test, expect } from '@playwright/test'
import { TEST_USER, login, logout, registerUser } from '../helpers/auth'

test.describe('Authentication flows', () => {
  const uniqueUser = {
    username: `testuser_auth_${Date.now()}`,
    password: 'Test1234!',
  }

  test('Register a new user → redirected to landing page', async ({ page }) => {
    await page.goto('/register')

    await page.getByPlaceholder('請輸入使用者名稱').fill(uniqueUser.username)
    await page.getByPlaceholder('請輸入密碼（8-72 字元）').fill(uniqueUser.password)
    await page.getByPlaceholder('請再次輸入密碼').fill(uniqueUser.password)
    await page.getByRole('button', { name: '註冊' }).click()

    // Should auto-login and redirect to landing
    await expect(page).toHaveURL(/\/landing/, { timeout: 10000 })
  })

  test('Login with valid credentials → see dashboard', async ({ page }) => {
    // Ensure test user exists
    await registerUser(page, TEST_USER.username, TEST_USER.password)

    await page.goto('/login')
    await page.getByPlaceholder('請輸入使用者名稱').fill(TEST_USER.username)
    await page.getByPlaceholder('請輸入密碼').fill(TEST_USER.password)
    await page.getByRole('button', { name: '登入' }).click()

    await expect(page).toHaveURL(/\/landing/, { timeout: 10000 })
    // Verify we see some dashboard content
    await expect(page.locator('body')).toContainText(/SAFE-AI|梳理|上傳|Dashboard/i)
  })

  test('Login with invalid credentials → see error message', async ({ page }) => {
    await page.goto('/login')
    await page.getByPlaceholder('請輸入使用者名稱').fill('nonexistent_user')
    await page.getByPlaceholder('請輸入密碼').fill('wrong_password_123')
    await page.getByRole('button', { name: '登入' }).click()

    // Should stay on login page and show error
    await expect(page).toHaveURL(/\/login/)
    const errorBox = page.locator('[style*="rose"], .error-message, [role="alert"]').first()
    await expect(errorBox).toBeVisible({ timeout: 5000 })
  })

  test('Logout → redirected to login page', async ({ page }) => {
    // Login first
    await login(page, TEST_USER.username, TEST_USER.password)
    await expect(page).toHaveURL(/\/landing/)

    // Perform logout
    await logout(page)
    await expect(page).toHaveURL(/\/login/)
  })
})
