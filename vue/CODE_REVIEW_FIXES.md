# Vue Frontend Code Review - Fixes Applied

## Summary of Changes Made

This document summarizes the fixes applied to address the issues identified in the code analysis report.

---

## ✅ Critical Issues Fixed

### 1. Security: Token Exposed in URL Query Parameters (CRITICAL)

**Files Modified:**
- `vue/src/api/research.js`
- `vue/src/utils/apiClient.js`

**Changes:**
- Replaced insecure `EventSource` with token in URL query parameters with secure `fetch + ReadableStream` approach
- Token is now passed via `Authorization` header instead of URL
- Added `AbortController` support for proper cleanup and request cancellation
- Added comprehensive JSDoc comments explaining the security rationale

**Before (Insecure):**
```javascript
const eventSource = new EventSource(
  `${baseURL}/research/stream/${sessionId}?token=${token}`
)
```

**After (Secure):**
```javascript
const response = await fetch(`${baseURL}/research/stream/${sessionId}`, {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Accept': 'text/event-stream'
  },
  signal: abortController.signal
})
```

---

## ✅ High Priority Issues Fixed

### 2. Orchestrator Store Uses Centralized API

**File Modified:** `vue/src/stores/orchestrator.js`

**Changes:**
- Replaced raw `axios` import with centralized `apiClient` from `@/api/index`
- Updated all API calls to use relative URLs (removed `/api` prefix)
- Updated response handling to access data directly (apiClient returns unwrapped data)

**Benefits:**
- Automatic token injection via interceptors
- Consistent error handling
- Proper baseURL configuration
- Unified request logging and metrics

### 3. Removed Debug Console.log Statements

**Files Modified:**
- `vue/src/router/index.js` - Removed 15+ console.log statements from route guards
- `vue/src/views/Login.vue` - Removed 6+ console.log statements from login flow

**Rationale:**
- Debug logs in production expose routing logic
- Login logs could reveal authentication flow details
- Reduces console clutter in production

### 4. Removed Unused Dependencies

**File Modified:** `vue/package.json`

**Changes:**
- Removed `vuex` (Pinia is the proper state management solution in use)
- Removed `marked` (duplicate markdown library; `markdown-it` is sufficient)

**Impact:**
- Reduced bundle size
- Eliminated confusion from redundant libraries

---

## ✅ Medium Priority Issues Fixed

### 5. Removed Unused Import and Code in App.vue

**File Modified:** `vue/src/App.vue`

**Changes:**
- Removed unused `computed` import
- Removed unused `isAdmin` computed property (was defined but never used in template)

### 6. Replaced Magic Numbers with Named Constants

**File Modified:** `vue/src/views/AISpace.vue`

**Changes:**
```javascript
// Before
const pageSize = 3; // 每页3道题

// After
// 分页配置 - 每页显示3道题目，平衡了阅读效率和滚动便利性
const QUESTIONS_PER_PAGE = 3;
```

---

## 📋 Remaining Issues (Recommended for Follow-up)

### Pre-existing Linting Errors (43 total in other files)
The codebase has existing linting errors that should be addressed:

1. **HealthCheck.vue** - Missing imports for `computed`, `onUnmounted`, `watch`
2. **MessageItem.vue** - Multiple issues including unreachable code and undefined variables
3. **LoadingSpinner.vue** - Parsing error
4. **Various components** - Unused variable warnings

### Architectural Recommendations

1. **Consolidate API Clients** - There are still 3 separate axios instance configurations:
   - `utils/apiClient.js`
   - `services/api.js`
   - `api/index.js`
   
   Recommend keeping only `api/index.js` as the single source of truth.

2. **Refactor AISpace.vue** - At 1,410 lines, this component should be split into:
   - `ChatContainer.vue`
   - `QuestionsPanel.vue`
   - `QuestionPagination.vue`
   - `ChatInput.vue`

3. **Standardize Error Handling** - Choose one approach:
   - `composables/useErrorHandler.js` (recommended - most complete)

4. **Add EventSource Cleanup** - Ensure SSE connections are properly closed in component unmount hooks

5. **Add Request Cancellation** - Implement `AbortController` for all API requests that can be cancelled

---

## 🧪 Testing Recommendations

After these changes, please verify:

1. **SSE Streaming** - Test research progress streaming works with the new fetch-based approach
2. **Orchestrator** - Verify orchestrator functionality with the new apiClient integration
3. **Navigation** - Test route guards work correctly without console logs
4. **Login Flow** - Verify login/logout works correctly

---

## Files Changed Summary

| File | Changes |
|------|---------|
| `package.json` | Removed vuex, marked dependencies |
| `api/research.js` | Fixed SSE token security |
| `utils/apiClient.js` | Fixed createSSEConnection security |
| `stores/orchestrator.js` | Use centralized apiClient |
| `router/index.js` | Removed console.logs |
| `views/Login.vue` | Removed console.logs |
| `App.vue` | Removed unused code |
| `views/AISpace.vue` | Magic number → constant |
