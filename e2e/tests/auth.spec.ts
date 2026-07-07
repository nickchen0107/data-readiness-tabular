import { test, expect } from '@playwright/test'
import { TEST_USER, login, logout, registerUser } from '../helpers/auth'

test.describe('Authentication flows', () => {
  const uniqueUser = {
    username: `testuser_auth_${Date.now()}`,
    password: 'Test1234!',
  }

  test('Register a new user → redirected to landing page', async ({ page }) => {
    await page.goto('./register')
    await page.waitForTimeout(500)

    await page.locator('input[type="text"]').first().fill(uniqueUser.username)
    await page.locator('input[type="password"]').first().fill(uniqueUser.password)
    await page.locator('input[placeholder*="再次"], input[placeholder*="Re-enter"], input[placeholder*="Confirm"]').fill(uniqueUser.password)
    await page.getByRole('button', { name: /註冊|register/i }).click()

    // Should auto-login and redirect to landing
    await expect(page).toHaveURL(/\/landing/, { timeout: 10000 })
  })

  test('Login with valid credentials → see dashboard', async ({ page }) => {
    // Use the user we just registered above, or the standard test user
    await page.goto('./login')
    await page.waitForTimeout(300)
    await page.locator('input[type="text"]').first().fill(TEST_USER.username)
    await page.locator('input[type="password"]').first().fill(TEST_USER.password)
    await page.getByRole('button', { name: /登入|login/i }).click()

    // Might need to register if user doesn't exist
    const loginSucceeded = await page.waitForURL(/\/landing/, { timeout: 5000 }).then(() => true).catch(() => false)

    if (!loginSucceeded) {
      // Register the test user first
      await registerUser(page, TEST_USER.username, TEST_USER.password)
      await page.waitForTimeout(500)
      await page.goto('./login')
      await page.waitForTimeout(300)
      await page.locator('input[type="text"]').first().fill(TEST_USER.username)
      await page.locator('input[type="password"]').first().fill(TEST_USER.password)
      await page.getByRole('button', { name: /登入|login/i }).click()
      await expect(page).toHaveURL(/\/landing/, { timeout: 10000 })
    }

    // Verify we see some dashboard content
    await expect(page.locator('body')).toContainText(/SAFE-AI|梳理|上傳|Dashboard/i)
  })

  test('Login with invalid credentials → see error message', async ({ page }) => {
    await page.goto('./login')
    await page.waitForTimeout(300)
    await page.locator('input[type="text"]').first().fill('nonexistent_user_xyz')
    await page.locator('input[type="password"]').first().fill('wrong_password_123')
    await page.getByRole('button', { name: /登入|login/i }).click()

    // Should stay on login page and show error
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/login/)

    // Wait for error to appear
    await expect(page.getByText(/帳號或密碼錯誤|登入失敗|failed|error|不存在/i).first()).toBeVisible({ timeout: 5000 })
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
