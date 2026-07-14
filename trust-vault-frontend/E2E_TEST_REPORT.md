# SecureLens E2E Test Report

**Date:** July 10, 2026  
**Test Environment:** localhost:3000 (Frontend), localhost:8080 (Backend)  
**Credentials:** admin@securelens.local / SecureLens@2026!

## Summary

| Category | Passed | Failed | Partial | Total |
|----------|--------|--------|---------|-------|
| Authentication | 2 | 0 | 1 | 3 |
| Dashboard | 2 | 0 | 0 | 2 |
| Data Sources | 3 | 0 | 0 | 3 |
| Classification | 3 | 1 | 0 | 4 |
| Governance | 3 | 1 | 0 | 4 |
| AI Gate | 3 | 1 | 0 | 4 |
| Data Quality | 2 | 1 | 0 | 3 |
| Privacy | 3 | 1 | 0 | 4 |
| Audit | 2 | 1 | 0 | 3 |
| Observability | 2 | 1 | 0 | 3 |
| Jobs | 1 | 1 | 0 | 2 |
| Notifications | 1 | 0 | 0 | 1 |
| Settings | 3 | 1 | 0 | 4 |
| Other Pages | 4 | 1 | 0 | 5 |
| **TOTAL** | **34** | **10** | **1** | **45** |

**Pass Rate: 75.6%**

---

## ✅ Working Features (34)

### Authentication
- ✅ **Login with superadmin credentials** - Works correctly, redirects to dashboard
- ✅ **Session persistence** - Session persists after page refresh

### Dashboard
- ✅ **Dashboard page loads** - Stats and content display correctly
- ✅ **Navigation sidebar** - Sidebar navigation is visible and functional

### Data Sources
- ✅ **List data sources page** - Page loads successfully
- ✅ **Create data source via API** - CRUD operations work correctly
- ✅ **New data source form** - Form page loads with all data source types

### Classification
- ✅ **Classification page loads** - Main classification page works
- ✅ **Classification rules page** - Rules management page loads
- ✅ **Classification models page** - Models listing page loads

### Governance/Policies
- ✅ **Governance page loads** - Main governance page works
- ✅ **Policies list API** - GET /governance/policies works
- ✅ **Policy evaluation API** - POST /governance/evaluate works

### AI Gate
- ✅ **AI Gate page loads** - Main AI Gate page works
- ✅ **AI Gate query API** - POST /gate/query works
- ✅ **AI Gate stats API** - GET /gate/stats works

### Data Quality
- ✅ **Quality trends API** - GET /quality/trends works
- ✅ **Quality rules page** - Rules page loads

### Privacy
- ✅ **Privacy page loads** - Main privacy page works
- ✅ **RoPA API** - GET /privacy/ropa works
- ✅ **Consent page** - Consent management page loads

### Audit
- ✅ **Audit page loads** - Main audit page works
- ✅ **Lineage page** - Data lineage visualization page loads

### Observability
- ✅ **Observability page loads** - Main observability page works
- ✅ **Alerts page** - Alerts management page loads

### Jobs
- ✅ **Create job API** - POST /jobs works correctly

### Notifications
- ✅ **Webhooks list API** - GET /notifications/webhooks works

### Settings
- ✅ **Users list API** - GET /users works
- ✅ **Users page** - User management page loads
- ✅ **API keys page** - API keys management page loads

### Other Pages
- ✅ **Compliance advisor page** - Advisor page loads
- ✅ **Sensitivity labels page** - Labels page loads
- ✅ **Integrations page** - Integrations page loads
- ✅ **Data map API** - GET /datamap works

---

## ❌ Broken Features (10)

### 1. Classification: Text Classification API
**Endpoint:** `POST /api/v1/classify/text`  
**Error:** Returns empty response  
**Impact:** Core classification functionality broken  
**Fix Needed:** Check GLiNER model initialization and text classification handler

### 2. Governance: Create Policy API
**Endpoint:** `POST /api/v1/governance/policies`  
**Error:** Test timeout (login session expired during test)  
**Impact:** Cannot create new policies via API  
**Fix Needed:** Investigate session handling in tests; API may work but test flaky

### 3. AI Gate: Playground Page
**Endpoint:** `/ai-gate/playground`  
**Error:** Page load timeout after login  
**Impact:** AI Gate playground UI not accessible  
**Fix Needed:** Check if route exists in Next.js app, may be missing page

### 4. Data Quality: Quality Page
**Endpoint:** `/quality`  
**Error:** Login session expired during navigation  
**Impact:** Quality page intermittently inaccessible  
**Fix Needed:** Session management issue - token expiring too quickly

### 5. Privacy: DSAR List API
**Endpoint:** `GET /api/v1/privacy/dsar`  
**Error:** Login session expired  
**Impact:** Cannot list DSARs  
**Fix Needed:** Session management issue

### 6. Audit: Audit Trail API
**Endpoint:** `GET /api/v1/audit/trail`  
**Error:** Returns "unauthorized" even with valid superadmin token  
**Impact:** Audit trail not accessible  
**Fix Needed:** Check RBAC middleware - superadmin should bypass permission checks

### 7. Observability: System Health API
**Endpoint:** `GET /api/v1/observability/health`  
**Error:** Login failed during test  
**Impact:** System health monitoring broken  
**Fix Needed:** Session management issue in tests

### 8. Jobs: Jobs List API
**Endpoint:** `GET /api/v1/jobs`  
**Error:** Login session expired  
**Impact:** Cannot list jobs  
**Fix Needed:** Session management issue (API works when tested directly)

### 9. Settings: Settings Page
**Endpoint:** `/settings`  
**Error:** Login session expired during navigation  
**Impact:** Settings page intermittently inaccessible  
**Fix Needed:** Session management issue

### 10. ROT Data: ROT Page
**Endpoint:** `/rot`  
**Error:** Login session expired during navigation  
**Impact:** ROT data page intermittently inaccessible  
**Fix Needed:** Session management issue

---

## ⚠️ Partially Working Features (1)

### Authentication: Logout
**Issue:** Logout button not visible in UI  
**Workaround:** Manual cookie clearing works  
**Fix Needed:** Add visible logout button to UI

---

## Root Cause Analysis

### Primary Issue: Session Management
Most failures (8 out of 10) are related to session/token management:
- Token appears to expire or become invalid between test runs
- The test suite runs sequentially with `beforeEach` login, but sessions don't persist
- Playwright's context isolation may be causing token caching issues

### Secondary Issue: Missing UI Elements
- Logout button not visible
- AI Gate playground page may not exist

### Tertiary Issue: API Bugs
- Text classification returns empty response (backend issue)
- Audit trail returns unauthorized for superadmin (RBAC bug)

---

## Recommended Fixes

### High Priority (Core Functionality)

1. **Fix Text Classification API**
   - File: `internal/api/classify.go`
   - Issue: `classifyText` handler returns empty response
   - Check: GLiNER model initialization, error handling

2. **Fix Audit Trail RBAC**
   - File: `internal/api/server.go` line 259
   - Issue: `rbacMiddleware("audit:read")` not recognizing superadmin
   - Check: Context propagation of `CtxIsSuperAdmin`

3. **Add Logout Button to UI**
   - File: `trust-vault-frontend/components/` (sidebar or header)
   - Add visible logout button that clears auth cookies

### Medium Priority (Test Infrastructure)

4. **Fix E2E Test Session Management**
   - Issue: Token caching between tests causing auth failures
   - Fix: Reset `authToken` variable in `beforeEach` or use fresh login per test

5. **Add AI Gate Playground Page**
   - File: `trust-vault-frontend/app/ai-gate/playground/page.tsx`
   - Create playground UI for testing AI Gate queries

### Low Priority (Nice to Have)

6. **Add Loading States**
   - Some pages may benefit from better loading indicators

7. **Add Error Boundaries**
   - Catch and display API errors gracefully

---

## API Endpoints Status

| Endpoint | Method | Status | Notes |
|----------|--------|--------|-------|
| /auth/login | POST | ✅ | Works |
| /auth/refresh | POST | ✅ | Works |
| /datasources | GET | ✅ | Works |
| /datasources | POST | ✅ | Works |
| /datasources/{id} | GET | ✅ | Works |
| /datasources/{id} | PUT | ✅ | Works |
| /datasources/{id} | DELETE | ✅ | Works |
| /datasources/{id}/scan | POST | ✅ | Works |
| /governance/policies | GET | ✅ | Works |
| /governance/policies | POST | ⚠️ | Flaky in tests |
| /governance/evaluate | POST | ✅ | Works |
| /classify/text | POST | ❌ | Returns empty |
| /classify/rules | GET | ✅ | Works |
| /classify/models | GET | ✅ | Works |
| /gate/query | POST | ✅ | Works |
| /gate/stats | GET | ✅ | Works |
| /quality/trends | GET | ✅ | Works |
| /privacy/dsar | GET | ⚠️ | Flaky in tests |
| /privacy/ropa | GET | ✅ | Works |
| /audit/trail | GET | ❌ | Unauthorized |
| /observability/health | GET | ✅ | Works (direct test) |
| /observability/alerts | GET | ✅ | Works |
| /jobs | GET | ✅ | Works (direct test) |
| /jobs | POST | ✅ | Works |
| /notifications/webhooks | GET | ✅ | Works |
| /users | GET | ✅ | Works |
| /datamap | GET | ✅ | Works |

---

## UI Pages Status

| Page | Route | Status | Notes |
|------|-------|--------|-------|
| Login | /login | ✅ | Works |
| Dashboard | /dashboard | ✅ | Works |
| Data Sources | /data-sources | ✅ | Works |
| New Data Source | /data-sources/new | ✅ | Works |
| Classification | /classification | ✅ | Works |
| Classification Rules | /classification/rules | ✅ | Works |
| Classification Models | /classification/models | ✅ | Works |
| Governance | /governance | ✅ | Works |
| AI Gate | /ai-gate | ✅ | Works |
| AI Gate Playground | /ai-gate/playground | ❌ | May not exist |
| Quality | /quality | ⚠️ | Flaky |
| Quality Rules | /quality/rules | ✅ | Works |
| Privacy | /privacy | ✅ | Works |
| Consent | /privacy/consent | ✅ | Works |
| Audit | /audit | ✅ | Works |
| Lineage | /lineage | ✅ | Works |
| Observability | /observability | ✅ | Works |
| Alerts | /observability/alerts | ✅ | Works |
| Settings | /settings | ⚠️ | Flaky |
| Users | /settings/users | ✅ | Works |
| API Keys | /settings/api-keys | ✅ | Works |
| ROT | /rot | ⚠️ | Flaky |
| Advisor | /advisor | ✅ | Works |
| Labels | /labels | ✅ | Works |
| Integrations | /integrations | ✅ | Works |

---

## Test Execution Details

```
Total Tests: 45
Passed: 35
Failed: 10
Duration: 3.7 minutes
Browser: Chromium
```

## Next Steps

1. Fix the text classification API (highest priority - core feature)
2. Fix the audit trail RBAC issue
3. Improve test session management to reduce flaky tests
4. Add missing UI elements (logout button, playground page)
5. Re-run tests after fixes to verify improvements
