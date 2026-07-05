import { type Page, expect } from '@playwright/test'

export const TEST_USER = {
  username: 'testuser_e2e',
  password: 'Test1234!',
}

export const ADMIN_USER = {
  username: 'admin_e2e',
  password: 'Admin1234!',
}

/**
 * Register a new user. Silently succeeds if user already exists.
 */
export async function registerUser(page: Page, username: string, password: string): Promise<void> {
  await page.goto('./register')
  await page.waitForTimeout(300)
  await page.locator('input[placeholder*="使用者名稱"]').first().fill(username)
  await page.locator('input[type="password"]').first().fill(password)
  await page.locator('input[placeholder*="再次"]').fill(password)
  await page.getByRole('button', { name: /註冊|register/i }).click()

  // Either redirected to landing (success) or shows "already exists" error
  await page.waitForURL(/\/(landing|register)/, { timeout: 10000 }).catch(() => {})
  await page.waitForTimeout(300)
}

/**
 * Login with given credentials and wait for navigation to landing page.
 */
export async function login(page: Page, username: string, password: string): Promise<void> {
  await page.goto('./login')
  await page.waitForTimeout(300)
  await page.locator('input[placeholder*="使用者名稱"]').first().fill(username)
  await page.locator('input[type="password"]').fill(password)
  await page.getByRole('button', { name: /登入|login/i }).click()
  await page.waitForURL(/\/(landing|upload|assessment)/, { timeout: 10000 })
}

/**
 * Logout the current user.
 */
export async function logout(page: Page): Promise<void> {
  // Look for logout button in the header/nav
  const logoutBtn = page.getByRole('button', { name: /登出|logout/i })
  if (await logoutBtn.isVisible()) {
    await logoutBtn.click()
  } else {
    // Try settings or avatar menu
    const avatar = page.locator('[data-testid="user-menu"], .user-avatar, header button').first()
    if (await avatar.isVisible()) {
      await avatar.click()
      await page.getByText(/登出|logout/i).click()
    }
  }
  await page.waitForURL('**/login', { timeout: 10000 })
}

/**
 * Ensure test user is registered (call in beforeAll).
 * If registration fails (user already exists), just proceed to login to verify creds work.
 */
export async function ensureTestUser(page: Page): Promise<void> {
  await page.goto('./register')
  await page.waitForTimeout(500)
  await page.locator('input[placeholder*="使用者名稱"]').first().fill(TEST_USER.username)
  await page.locator('input[type="password"]').first().fill(TEST_USER.password)
  await page.locator('input[placeholder*="再次"]').fill(TEST_USER.password)
  await page.getByRole('button', { name: /註冊|register/i }).click()

  // Wait for either success redirect or error (already exists)
  await page.waitForTimeout(2000)

  // If we got redirected to landing, user was created successfully
  if (page.url().includes('/landing')) {
    // Logout so tests start fresh
    await page.goto('./login')
    return
  }

  // User already exists — verify we can login
  await page.goto('./login')
  await page.waitForTimeout(300)
  await page.locator('input[placeholder*="使用者名稱"]').first().fill(TEST_USER.username)
  await page.locator('input[type="password"]').fill(TEST_USER.password)
  await page.getByRole('button', { name: /登入|login/i }).click()
  await page.waitForURL(/\/(landing|upload|assessment)/, { timeout: 10000 }).catch(() => {})
  await page.waitForTimeout(300)
  await page.goto('./login')
}

/**
 * Ensure admin user is registered (call in beforeAll).
 * Note: Admin promotion must be done via direct API/DB in real setup.
 */
export async function ensureAdminUser(page: Page): Promise<void> {
  await page.goto('./register')
  await page.waitForTimeout(500)
  await page.locator('input[placeholder*="使用者名稱"]').first().fill(ADMIN_USER.username)
  await page.locator('input[type="password"]').first().fill(ADMIN_USER.password)
  await page.locator('input[placeholder*="再次"]').fill(ADMIN_USER.password)
  await page.getByRole('button', { name: /註冊|register/i }).click()

  // Wait for result
  await page.waitForTimeout(2000)

  // Attempt admin promotion via API (if bootstrap endpoint exists)
  try {
    const response = await page.request.post('./api/admin/bootstrap', {
      data: { username: ADMIN_USER.username },
    })
    if (response.status() === 404) {
      console.warn('Admin bootstrap endpoint not found. Ensure admin_e2e is promoted manually.')
    }
  } catch {
    console.warn('Could not promote admin user via API. Ensure admin_e2e has admin role.')
  }

  await page.goto('./login')
  await page.waitForTimeout(300)
}

/**
 * Login as the default test user.
 */
export async function loginAsTestUser(page: Page): Promise<void> {
  await login(page, TEST_USER.username, TEST_USER.password)
}

/**
 * Login as the admin user.
 */
export async function loginAsAdmin(page: Page): Promise<void> {
  await login(page, ADMIN_USER.username, ADMIN_USER.password)
}
