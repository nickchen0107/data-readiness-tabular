---
inclusion: auto
---

# E2E Testing Rules

## Rule: New features must have E2E coverage

When implementing any new functionality or testable behavior:

1. Check if existing E2E tests in `e2e/tests/` cover the new feature
2. If not, add corresponding test cases to the appropriate spec file
3. Always run `npx playwright test` in `e2e/` directory after changes and before pushing
4. All 25+ tests must pass before pushing to main

## Test structure

- `e2e/tests/auth.spec.ts` — Login, Register, Logout
- `e2e/tests/upload-assess.spec.ts` — Upload file, select sheet, start assessment
- `e2e/tests/cleaning-export.spec.ts` — Cleaning rules, export, downloads
- `e2e/tests/admin.spec.ts` — Admin panel (users, quota, translations, records)
- `e2e/tests/quota.spec.ts` — Quota enforcement
- `e2e/tests/i18n.spec.ts` — Language switching

## Test accounts

- Regular user: `testuser_e2e` / `Test1234!`
- Admin user: `admin_e2e` / `Admin1234!`

## Before running tests

Always clear rate limit first:
```bash
docker compose exec db psql -U safeai -d safeai -c "DELETE FROM login_attempts;"
```
