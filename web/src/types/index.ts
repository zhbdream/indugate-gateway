export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface Device {
  id: number
  name: string
  protocol: 'opcua' | 'modbus' | 'mqtt' | 's7' | 'bacnet'
  address: string
  config: string
  status: 'disconnected' | 'connected' | 'error'
  description: string
  created_at: string
  updated_at: string
}

export interface DeviceForm {
  name: string
  protocol: Device['protocol']
  address: string
  config: string
  description: string
}

export interface NodeInfo {
  node_id: string
  browse_name: string
  display_name?: string
  description?: string
  node_class: string
  data_type?: string
  writable: boolean
  path?: string
  has_children?: boolean
}

export interface DataValue {
  node_id: string
  value: unknown
  data_type?: string
  status: string
  timestamp: string
}

export interface Simulator {
  type: string
  status: string
  description: string
  endpoint?: string
  nodes?: string[]
  topics?: string[]
}

export interface SubscriptionInfo {
  id: string
  device_id: number
  node_ids: string[]
  interval: string
  created_at: string
}

export interface DataChangeEvent {
  subscription_id: string
  node_id: string
  value: unknown
  status: string
  timestamp: string
}

export interface DashboardStats {
  device_total: number
  device_connected: number
  device_error: number
  active_alerts: number
  alert_rules: number
  history_records_24h: number
}

export interface AlertRule {
  id: number
  device_id: number
  node_id: string
  name: string
  enabled: boolean
  condition: 'gt' | 'lt' | 'eq' | 'gte' | 'lte' | 'range' | 'change_rate'
  threshold: number
  threshold_max?: number
  level: 'INFO' | 'WARN' | 'ERROR' | 'CRITICAL'
  description: string
  created_at: string
  updated_at: string
}

export interface AlertEvent {
  id: number
  rule_id: number
  device_id: number
  node_id: string
  level: AlertRule['level']
  message: string
  value: string
  status: 'active' | 'resolved'
  triggered_at: string
  resolved_at?: string
  created_at: string
}
