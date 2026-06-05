import http from './client'

export interface AuthConfig {
  auth_enabled: boolean
  jwt_enabled: boolean
  device_acl_enabled: boolean
}

export interface LoginResult {
  token: string
  user: {
    id: number
    username: string
    role: string
  }
}

export function getAuthConfig() {
  return http.get<unknown, AuthConfig>('/api/v1/auth/config')
}

export function login(username: string, password: string) {
  return http.post<unknown, LoginResult>('/api/v1/auth/login', { username, password })
}

export function getMe() {
  return http.get<unknown, LoginResult['user']>('/api/v1/auth/me')
}
