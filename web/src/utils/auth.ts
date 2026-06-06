const TOKEN_KEY = 'indugate_api_token'
const JWT_KEY = 'indugate_jwt_token'
const USER_ROLE_KEY = 'indugate_user_role'
const USER_NAME_KEY = 'indugate_user_name'

export function getApiToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setApiToken(token: string) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
    // Static API token is treated as admin on the backend when RBAC is enabled.
    localStorage.setItem(USER_ROLE_KEY, 'admin')
    localStorage.setItem(USER_NAME_KEY, 'api-token')
    localStorage.removeItem(JWT_KEY)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export function getJwtToken(): string {
  return localStorage.getItem(JWT_KEY) || ''
}

export function setJwtToken(token: string) {
  if (token) {
    localStorage.setItem(JWT_KEY, token)
  } else {
    localStorage.removeItem(JWT_KEY)
  }
}

export function setUserSession(username: string, role: string) {
  localStorage.setItem(USER_NAME_KEY, username)
  localStorage.setItem(USER_ROLE_KEY, role)
}

export function getUserRole(): string {
  return localStorage.getItem(USER_ROLE_KEY) || ''
}

export function getUsername(): string {
  return localStorage.getItem(USER_NAME_KEY) || ''
}

export function isAdmin(): boolean {
  return getUserRole() === 'admin'
}

export function isViewer(): boolean {
  return getUserRole() === 'viewer'
}

export function canWrite(): boolean {
  const role = getUserRole()
  if (!role) return true
  return role === 'admin' || role === 'operator'
}

export function roleLabel(role: string): string {
  return ({ admin: '管理员', operator: '操作员', viewer: '只读' } as Record<string, string>)[role] || role || '未知'
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(JWT_KEY)
  localStorage.removeItem(USER_ROLE_KEY)
  localStorage.removeItem(USER_NAME_KEY)
}

export function getAuthHeaderToken(): string {
  return getJwtToken() || getApiToken()
}

export function hasAuthCredentials(): boolean {
  return getAuthHeaderToken().length > 0
}
