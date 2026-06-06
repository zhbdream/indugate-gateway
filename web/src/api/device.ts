import http from './client'
import type { DataChangeEvent, DataValue, Device, DeviceForm, NodeInfo, SubscriptionInfo } from '@/types'

export function listDevices() {
  return http.get<unknown, Device[]>('/api/v1/devices')
}

export function getDevice(id: number) {
  return http.get<unknown, Device>(`/api/v1/devices/${id}`)
}

export function createDevice(data: DeviceForm) {
  return http.post<unknown, Device>('/api/v1/devices', data)
}

export function updateDevice(id: number, data: DeviceForm) {
  return http.put<unknown, Device>(`/api/v1/devices/${id}`, data)
}

export function deleteDevice(id: number) {
  return http.delete(`/api/v1/devices/${id}`)
}

export function connectDevice(id: number) {
  return http.post<unknown, Device>(`/api/v1/devices/${id}/connect`)
}

export function disconnectDevice(id: number) {
  return http.post<unknown, Device>(`/api/v1/devices/${id}/disconnect`)
}

export function browseNodes(id: number, params?: { node?: string; depth?: number; children_only?: boolean }) {
  return http.get<unknown, NodeInfo[]>(`/api/v1/devices/${id}/nodes`, { params })
}

export function readData(id: number, nodeId: string, options?: { silent?: boolean }) {
  return http.get<unknown, DataValue>(`/api/v1/devices/${id}/data/${encodeURIComponent(nodeId)}`, {
    params: { node: nodeId },
    silent: options?.silent,
  })
}

export function writeData(id: number, nodeId: string, value: unknown) {
  return http.post(`/api/v1/devices/${id}/data/${encodeURIComponent(nodeId)}`, { value }, {
    params: { node: nodeId },
  })
}

export function subscribeData(id: number, nodeIds: string[], intervalMs = 1000) {
  return http.post<unknown, SubscriptionInfo>(`/api/v1/devices/${id}/subscribe`, {
    node_ids: nodeIds,
    interval_ms: intervalMs,
  })
}

export function pollSubscriptionEvents(id: number, subId: string, clear = true) {
  return http.get<unknown, DataChangeEvent[]>(
    `/api/v1/devices/${id}/subscriptions/${subId}/events`,
    { params: { clear } }
  )
}

export function unsubscribe(id: number, subId: string) {
  return http.delete(`/api/v1/devices/${id}/subscriptions/${subId}`)
}
