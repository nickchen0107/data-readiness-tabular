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
  await page.goto('/register')
  await page.getByPlaceholder('請輸入使用者名稱').fill(username)
  await page.getByPlaceholder('請輸入密碼（8-72 字元）').fill(password)
  await page.getByPlaceholder('請再次輸入密碼').fill(password)
  await page.getByRole('button', { name: '註冊' }).click()

  // Either redirected to landing (success) or shows "already exists" error
  await page.waitForURL(/\/(landing|register)/, { timeout: 10000 })
}

/**
 * Login with given credentials and wait for navigation to landing page.
 */
export async function login(page: Page, username: string, password: string): Promise<void> {
  await page.goto('/login')
  await page.getByPlaceholder('請輸入使用者名稱').fill(username)
  await page.getByPlaceholder('請輸入密碼').fill(password)
  await page.getByRole('button', { name: '登入' }).click()
  await page.waitForURL('**/landing', { timeout: 10000 })
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
 */
export async function ensureTestUser(page: Page): Promise<void> {
  await registerUser(page, TEST_USER.username, TEST_USER.password)
  // Navigate away to clean state
  await page.goto('/login')
}

/**
 * Ensure admin user is registered (call in beforeAll).
 * Note: Admin promotion must be done via direct API/DB in real setup.
 * Here we register and attempt to promote via an admin-bootstrap endpoint if available.
 */
export async function ensureAdminUser(page: Page): Promise<void> {
  await registerUser(page, ADMIN_USER.username, ADMIN_USER.password)
  // Attempt admin promotion via API (if bootstrap endpoint exists)
  try {
    const response = await page.request.post('/api/admin/bootstrap', {
      data: { username: ADMIN_USER.username },
    })
    // Silently handle if endpoint doesn't exist
    if (response.status() === 404) {
      console.warn('Admin bootstrap endpoint not found. Ensure admin_e2e is promoted manually.')
    }
  } catch {
    console.warn('Could not promote admin user via API. Ensure admin_e2e has admin role.')
  }
  await page.goto('/login')
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
