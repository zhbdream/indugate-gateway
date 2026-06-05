import http from './client'

export interface AuditLog {
  id: number
  username: string
  role: string
  method: string
  path: string
  action: string
  detail: string
  client_ip: string
  status_code: number
  success: boolean
  created_at: string
}

export interface AuditLogListResult {
  items: AuditLog[]
  total: number
}

export async function listAuditLogs(params?: {
  username?: string
  action?: string
  limit?: number
  offset?: number
}) {
  const { data } = await http.get<AuditLogListResult>('/api/v1/audit/logs', { params })
  return data as unknown as AuditLogListResult
}
