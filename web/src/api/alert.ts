import http from './client'
import type { AlertEvent, AlertRule, DashboardStats } from '@/types'

export function getDashboardStats() {
  return http.get<unknown, DashboardStats>('/api/v1/dashboard/stats')
}

export function listAlertRules(deviceId?: number) {
  return http.get<unknown, AlertRule[]>('/api/v1/alerts/rules', {
    params: deviceId ? { device_id: deviceId } : {},
  })
}

export function createAlertRule(data: Omit<AlertRule, 'id' | 'created_at' | 'updated_at'>) {
  return http.post<unknown, AlertRule>('/api/v1/alerts/rules', data)
}

export function updateAlertRule(id: number, data: Partial<AlertRule>) {
  return http.put<unknown, AlertRule>(`/api/v1/alerts/rules/${id}`, data)
}

export function deleteAlertRule(id: number) {
  return http.delete(`/api/v1/alerts/rules/${id}`)
}

export function listAlertEvents(params?: { device_id?: number; status?: string; limit?: number }) {
  return http.get<unknown, AlertEvent[]>('/api/v1/alerts/events', { params })
}

export function acknowledgeAlertEvent(id: number) {
  return http.post<unknown, AlertEvent>(`/api/v1/alerts/events/${id}/acknowledge`)
}

export function queryHistory(deviceId: number, params?: { node_id?: string; limit?: number; since?: string }) {
  return http.get<unknown, HistoryRow[]>(`/api/v1/devices/${deviceId}/data/history`, { params })
}

export function exportHistoryCSV(deviceId: number, params?: { node_id?: string; limit?: number }) {
  const qs = new URLSearchParams()
  if (params?.node_id) qs.set('node_id', params.node_id)
  if (params?.limit) qs.set('limit', String(params.limit))
  const query = qs.toString()
  return `${http.defaults.baseURL || ''}/api/v1/devices/${deviceId}/data/history/export${query ? `?${query}` : ''}`
}

export interface HistoryRow {
  id: number
  device_id: number
  node_id: string
  value: string
  data_type: string
  status: string
  timestamp: string
}
