import http from './client'

export interface User {
  id: number
  username: string
  role: 'admin' | 'operator' | 'viewer'
  created_at: string
  updated_at: string
}

export function listUsers() {
  return http.get<unknown, User[]>('/api/v1/users')
}

export function createUser(data: { username: string; password: string; role: User['role'] }) {
  return http.post<unknown, User>('/api/v1/users', data)
}

export function updateUser(id: number, role: User['role']) {
  return http.put<unknown, User>(`/api/v1/users/${id}`, { role })
}

export function changeUserPassword(id: number, password: string) {
  return http.put(`/api/v1/users/${id}/password`, { password })
}

export function deleteUser(id: number) {
  return http.delete(`/api/v1/users/${id}`)
}

export function getUserDevices(id: number) {
  return http.get<unknown, { device_ids: number[] }>(`/api/v1/users/${id}/devices`)
}

export function setUserDevices(id: number, deviceIDs: number[]) {
  return http.put(`/api/v1/users/${id}/devices`, { device_ids: deviceIDs })
}
