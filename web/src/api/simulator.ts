import http from './client'
import type { Simulator } from '@/types'

export function listSimulators() {
  return http.get<unknown, Simulator[]>('/api/v1/simulators')
}

export function startSimulator(type: string) {
  return http.post<unknown, Simulator>(`/api/v1/simulators/${type}/start`)
}

export function stopSimulator(type: string) {
  return http.post<unknown, Simulator>(`/api/v1/simulators/${type}/stop`)
}

export function updateSimulatorConfig(type: string, configJSON: string) {
  return http.put<unknown, { type: string; message: string }>(
    `/api/v1/simulators/${type}/config`,
    configJSON,
    {
      headers: { 'Content-Type': 'application/json' },
      transformRequest: [(data) => data],
    }
  )
}
