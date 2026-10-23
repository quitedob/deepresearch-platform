import { clearTokens } from './token'

const INACTIVE_ACCOUNT_CODE = 'ERR_ACCOUNT_INACTIVE'

// Permission and quota failures must not invalidate an otherwise valid session.
export function handleAuthenticationFailure(response) {
  const data = response?.data
  const inactive = data?.code === INACTIVE_ACCOUNT_CODE || data?.error?.code === INACTIVE_ACCOUNT_CODE
  if (response?.status !== 401 && !(response?.status === 403 && inactive)) return false
  const onLoginPage = window.location.pathname.startsWith('/login')

  clearTokens()
  for (const storage of [localStorage, sessionStorage]) storage.removeItem('user')
  if (!onLoginPage) {
    const message = data?.message || data?.error?.message || '账户已被禁用或不可用'
    window.location.href = inactive
      ? `/login?reason=banned&msg=${encodeURIComponent(message)}`
      : '/login'
  }
  return true
}
