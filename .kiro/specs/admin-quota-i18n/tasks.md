# Implementation Plan: Admin, Quota & i18n

## Overview

Implements three interconnected modules for the SAFE-AI Excel Brushing Tool: role-based admin system, assessment quota management with lazy reset, and internationalization (i18n) with admin translation editor. The build order follows data layer → backend services → frontend integration, ensuring each layer builds on the previous one.

## Tasks

- [x] 1. Database migration and schema changes
  - [x] 1.1 Create migration 003_admin_quota_i18n.sql
    - Add `role` column to users table (VARCHAR, default 'user', CHECK IN ('admin','user'))
    - Add `last_quota_reset` column to users table (TIMESTAMP WITH TIME ZONE, DEFAULT NOW())
    - Create `quota_settings` table with id, max_assessments, reset_period, updated_at
    - Insert default quota_settings row (max_assessments=5, reset_period='daily')
    - Create `translations` table with id, locale, key, value, updated_at
    - Create unique index on translations(locale, key) and index on translations(locale)
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6_

  - [x] 1.2 Seed default translations for both locales
    - Extract all hardcoded Chinese strings from frontend components (page titles, button labels, form labels, error messages, stepper labels, indicator names, status labels)
    - Create INSERT statements for zh-TW translations with all extracted keys
    - Create INSERT statements for en translations with English equivalents
    - Preserve proper nouns untranslated: "SAFE-AI", "S.A.F.E.-AI", "AI Readiness"
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 12.4_

- [x] 2. Backend auth changes (role in JWT + admin middleware)
  - [x] 2.1 Update User model and JWT functions to include role
    - Add `Role` field to `auth.User` struct with db tag and json tag
    - Update `GenerateToken` to accept and embed role claim in JWT payload
    - Update `ValidateToken` to extract and return role from token
    - Update auth repository queries to include role column in SELECT/INSERT
    - Ensure registration always sets role='user'
    - _Requirements: 1.1, 1.2, 1.3_

  - [ ]* 2.2 Write property test for JWT role round-trip (Go/rapid)
    - **Property 2: JWT role round-trip**
    - Generate random (userID, role ∈ {'admin','user'}) → GenerateToken → ValidateToken → assert role matches
    - **Validates: Requirements 1.3**

  - [x] 2.3 Update JWTAuth middleware to extract role and set in gin context
    - Modify `middleware.JWTAuth` to call updated `ValidateToken` that returns role
    - Set `c.Set("user_role", role)` in gin context alongside existing user_id
    - Update auth handler `GetMe` response to include role field
    - _Requirements: 1.3, 2.4_

  - [x] 2.4 Create AdminAuth middleware
    - Create `internal/middleware/admin.go` with `AdminAuth()` gin.HandlerFunc
    - Read `user_role` from gin context; if not "admin", respond 403 with error "權限不足"
    - Must be placed after JWTAuth middleware in route chain
    - _Requirements: 2.3, 2.4_

  - [ ]* 2.5 Write property test for admin middleware rejection (Go/rapid)
    - **Property 3: Admin middleware rejects non-admin**
    - Generate random admin endpoint paths + user JWT with role='user' → send request → assert 403 with "權限不足"
    - **Validates: Requirements 2.3, 2.4**

- [x] 3. Checkpoint - Backend auth verification
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. Backend quota package (service + repository + API)
  - [x] 4.1 Create quota models and repository
    - Create `internal/quota/model.go` with Settings and QuotaInfo structs
    - Create `internal/quota/repository.go` with Repository struct
    - Implement GetSettings, UpdateSettings, GetUsageCount, IncrementUsage, ResetUser methods
    - UsageCount derived by counting assessments since last_quota_reset (no separate counter)
    - _Requirements: 4.1, 4.3, 5.1, 6.3_

  - [x] 4.2 Create quota service with lazy reset logic
    - Create `internal/quota/service.go` with Service struct
    - Implement `CheckAndConsume(ctx, userID)` with lazy reset algorithm
    - Implement `GetUserQuotaInfo(ctx, userID)` returning current status
    - Lazy reset: check if last_quota_reset < period boundary (daily 00:00 UTC+8, weekly Monday 00:00 UTC+8)
    - If reset needed, update last_quota_reset then count usage since new reset time
    - _Requirements: 5.1, 5.2, 6.1, 6.2, 6.3, 6.4_

  - [ ]* 4.3 Write property tests for quota settings round-trip (Go/rapid)
    - **Property 5: Quota settings round-trip**
    - Generate random valid (max_assessments ∈ [1,1000], reset_period ∈ {'daily','weekly'}) → save → read → assert equality
    - **Validates: Requirements 4.2**

  - [ ]* 4.4 Write property tests for quota enforcement (Go/rapid)
    - **Property 6: Invalid quota settings rejected** — generate invalid settings → assert 400
    - **Property 7: Assessment increments quota usage** — user with remaining > 0 → assess → assert count+1
    - **Property 8: Quota limit enforcement** — user at limit → assert 403 with "評估次數已用盡，請聯繫管理員"
    - **Property 9: Quota lazy reset** — user exhausted + last_reset before boundary → assert next request succeeds
    - **Validates: Requirements 4.4, 5.1, 5.2, 6.1, 6.2, 6.3, 6.4**

- [x] 5. Backend translation package (service + repository + API)
  - [x] 5.1 Create translation models and repository
    - Create `internal/translation/model.go` with Translation struct
    - Create `internal/translation/repository.go` with Repository struct
    - Implement FindByLocale, FindByID, Update, Search methods
    - Search supports case-insensitive matching on key or value fields
    - _Requirements: 12.1, 12.4, 13.2, 13.4_

  - [x] 5.2 Create translation service
    - Create `internal/translation/service.go` with Service struct
    - Implement GetByLocale (returns map[string]string), Update, Search methods
    - Validate locale ∈ {'zh-TW', 'en'}, return 400 for invalid
    - _Requirements: 12.1, 13.2, 13.4_

  - [x] 5.3 Create public translation endpoint handler
    - Create handler for `GET /api/translations/:locale` (public, no auth)
    - Returns `{"translations": {"key": "value", ...}}` format
    - Validate locale parameter
    - _Requirements: 12.1, 12.2_

  - [ ]* 5.4 Write property tests for translation operations (Go/rapid)
    - **Property 14: Translation update round-trip** — generate random (key, locale, value) → update → read → assert match
    - **Property 15: Translation search filter** — generate random search strings + datasets → assert all results contain query
    - **Property 16: Translation unique constraint** — generate random (locale, key) → insert twice → assert second fails
    - **Validates: Requirements 13.2, 13.4, 14.5**

- [x] 6. Backend admin handlers and route registration
  - [x] 6.1 Create admin handler package
    - Create `internal/admin/handler.go` with Handler struct (depends on auth.Repository, quota.Repository, translation.Repository, assessment.Repository)
    - Implement ListUsers (paginated, includes quota info per user)
    - Implement GetQuotaSettings, UpdateQuotaSettings (with validation: max_assessments ≥ 1, reset_period ∈ {'daily','weekly'})
    - Implement ListTranslations (paginated, with locale filter and search)
    - Implement UpdateTranslation (by ID, update value)
    - Implement ListAssessments (by user_id, paginated, ordered by created_at DESC)
    - _Requirements: 3.1, 3.2, 4.2, 4.4, 7.1, 7.2, 7.3, 13.1, 13.2, 13.4_

  - [ ]* 6.2 Write property tests for pagination and ordering (Go/rapid)
    - **Property 4: Pagination invariant** — generate random total count N, iterate all pages → assert exactly N distinct records, no duplicates/gaps
    - **Property 10: Assessment records ordered descending** — generate random timestamps → insert → query → assert descending order
    - **Validates: Requirements 3.2, 7.2, 7.3**

  - [x] 6.3 Register all new routes in main.go
    - Initialize quota repository and service
    - Initialize translation repository and service
    - Initialize admin handler with all dependencies
    - Add public route: `GET /api/translations/:locale` (no auth)
    - Add quota check to existing `POST /api/assess` route (wrap or middleware)
    - Add admin route group: `/api/admin` with JWTAuth + AdminAuth middlewares
    - Register admin sub-routes: users, quota, translations, assessments
    - _Requirements: 2.4, 3.1, 4.2, 5.1, 5.2, 12.1_

- [x] 7. Checkpoint - Full backend verification
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 8. Frontend i18n setup (react-i18next + translation files + language switcher)
  - [~] 8.1 Install react-i18next and configure i18n initialization
    - Add `react-i18next`, `i18next`, `i18next-http-backend` to package.json
    - Create `src/i18n/index.ts` with i18next init config (default locale zh-TW, fallback to bundled JSON)
    - Create `src/i18n/fallback/zh-TW.json` with all translation keys and Chinese values
    - Create `src/i18n/fallback/en.json` with all translation keys and English values
    - Configure custom backend to fetch from `/api/translations/:locale`, fallback to bundled JSON on failure
    - Wrap App in I18nextProvider in main.tsx
    - _Requirements: 10.1, 10.2, 12.2, 12.3_

  - [~] 8.2 Create LanguageSwitcher component
    - Create `src/components/LanguageSwitcher.tsx` toggle button (zh-TW ↔ en)
    - On click, call i18next.changeLanguage() and save preference to localStorage
    - On page load, read localStorage preference and apply
    - Add LanguageSwitcher to the header/nav area
    - _Requirements: 10.3, 10.4, 10.5_

  - [~] 8.3 Replace hardcoded Chinese strings with translation keys
    - Replace all hardcoded strings in page components with `t('key')` calls
    - Cover: page titles, button labels, form labels, error messages, stepper labels
    - Cover: indicator names, status labels (Ready/就緒, Conditional/有條件通過, Not Ready/未就緒)
    - Preserve proper nouns: "SAFE-AI", "S.A.F.E.-AI", "AI Readiness"
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_

  - [ ]* 8.4 Write property test for translation lookup correctness (TypeScript/fast-check)
    - **Property 12: Translation lookup correctness**
    - Generate random translation maps + locale + key → verify t() returns correct value from data source
    - **Validates: Requirements 10.3**

- [ ] 9. Frontend StepperContext + stepper navigation fixes
  - [~] 9.1 Create StepperContext with state persistence
    - Create `src/contexts/StepperContext.tsx` with StepperState interface (maxReachedStep, completedSteps, stepData)
    - Implement StepperContextType with markComplete, setStepData, canNavigateTo methods
    - Persist state to localStorage under key `stepper_state`
    - Restore state from localStorage on mount
    - Wrap App in StepperProvider
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [~] 9.2 Implement stepper navigation constraints in UI
    - Update Stepper component: steps with index > maxReachedStep shown in grey and disabled
    - On click of disabled step, show toast "請先完成前面的步驟"
    - Allow clicking completed steps and current step
    - When navigating to completed "上傳" step: show read-only file info + "重新上傳檔案" button
    - When navigating to completed "評估" step: show latest assessment result
    - _Requirements: 9.1, 9.2, 9.3, 8.2, 8.3_

  - [ ]* 9.3 Write property test for stepper navigation constraint (TypeScript/fast-check)
    - **Property 11: Stepper navigation constraint**
    - Generate random maxReachedStep (0-7) → verify canNavigateTo returns true for ≤ M, false for > M
    - **Validates: Requirements 9.1, 9.3**

- [ ] 10. Frontend UploadPage state preservation + quota enforcement UI
  - [~] 10.1 Implement quota enforcement UI on UploadPage
    - Fetch user quota info from backend (add API call)
    - When quota exhausted: disable "重新上傳檔案" / "開始評估" button
    - Show tooltip on hover of disabled button: "評估次數已用盡，請聯繫管理員"
    - Display remaining quota count in UI
    - _Requirements: 5.3, 5.4_

- [~] 11. Checkpoint - Frontend i18n and stepper verification
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 12. Frontend admin pages
  - [~] 12.1 Create AdminRoute guard and admin layout
    - Create `src/components/AdminRoute.tsx` — checks role from AuthContext, redirects to '/' with toast "無權限存取" if not admin
    - Create `src/pages/admin/AdminLayout.tsx` with sidebar navigation (Users, Quota, Translations, Records)
    - Update AuthContext to include role field from JWT / /auth/me response
    - Show admin nav link in header when role='admin'
    - Register `/admin` routes in App router wrapped with AdminRoute
    - _Requirements: 1.4, 2.1, 2.2_

  - [~] 12.2 Create UsersPage (admin user management)
    - Create `src/pages/admin/UsersPage.tsx`
    - Fetch paginated user list from `GET /api/admin/users`
    - Display table: email, used quota, remaining quota
    - Implement pagination (default 20 per page)
    - _Requirements: 3.1, 3.2, 3.3_

  - [~] 12.3 Create QuotaSettingsPage
    - Create `src/pages/admin/QuotaSettingsPage.tsx`
    - Fetch current settings from `GET /api/admin/quota`
    - Form to edit max_assessments (number input, min 1) and reset_period (select: daily/weekly)
    - Submit via `PUT /api/admin/quota` with validation feedback
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [~] 12.4 Create TranslationsPage (translation editor)
    - Create `src/pages/admin/TranslationsPage.tsx`
    - Fetch paginated translations from `GET /api/admin/translations` with locale filter and search
    - Display key-value list with editable value fields
    - Submit changes via `PUT /api/admin/translations/:id`
    - On successful save, invalidate frontend translation cache (trigger re-fetch)
    - Provide search input to filter by key or value
    - _Requirements: 13.1, 13.2, 13.3, 13.4_

  - [~] 12.5 Create AssessmentRecordsPage
    - Create `src/pages/admin/AssessmentRecordsPage.tsx`
    - Allow admin to select/filter by user
    - Fetch paginated records from `GET /api/admin/assessments?user_id=...`
    - Display table: timestamp, filename, score, status
    - Results ordered by timestamp descending
    - _Requirements: 7.1, 7.2, 7.3_

- [~] 13. Final checkpoint - Full integration verification
  - Ensure all tests pass, ask the user if questions arise.
  - Verify Docker builds successfully (frontend Dockerfile installs react-i18next dependencies)
  - Verify migration runs and seeds data correctly
  - Verify admin login → access admin pages → manage quota/translations
  - Verify language switching works end-to-end
  - Verify quota enforcement blocks assessment when limit reached

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- The migration must be tested by restarting Docker containers: `docker compose down && docker compose up --build`
- react-i18next must be in package.json before Docker build to be available at runtime
- No per-user quota overrides for MVP — only global settings in quota_settings table

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["2.2", "2.3"] },
    { "id": 3, "tasks": ["2.4", "2.5"] },
    { "id": 4, "tasks": ["4.1", "5.1"] },
    { "id": 5, "tasks": ["4.2", "4.3", "5.2"] },
    { "id": 6, "tasks": ["4.4", "5.3", "5.4"] },
    { "id": 7, "tasks": ["6.1"] },
    { "id": 8, "tasks": ["6.2", "6.3"] },
    { "id": 9, "tasks": ["8.1"] },
    { "id": 10, "tasks": ["8.2", "8.3", "9.1"] },
    { "id": 11, "tasks": ["8.4", "9.2"] },
    { "id": 12, "tasks": ["9.3", "10.1", "12.1"] },
    { "id": 13, "tasks": ["12.2", "12.3", "12.4", "12.5"] }
  ]
}
```
