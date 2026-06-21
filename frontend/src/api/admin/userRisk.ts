import { apiClient } from '../client'

export type UserRiskLevel = 'low' | 'medium' | 'high' | 'critical'

export interface UserRiskWindow {
  label: string
  start_at: string
  end_at: string
  prev_start_at: string
}

export interface UserRiskSummary {
  total_users: number
  risky_users: number
  high_risk_users: number
  critical_risk_users: number
  request_count: number
  error_count: number
  error_rate: number
  active_concurrency: number
  waiting_in_queue: number
  shared_ip_users: number
  shared_ip_groups: number
}

export interface UserRiskUser {
  id: number
  email: string
  username: string
  notes: string
  role: string
  status: string
  signup_source: string
  balance: number
  created_at: string
  last_login_at?: string
  last_active_at?: string
}

export interface UserRiskReason {
  code: string
  label: string
  severity: string
  detail: string
}

export interface UserRiskMetrics {
  request_count: number
  previous_request_count: number
  request_growth_ratio: number
  token_count: number
  cost: number
  previous_cost: number
  cost_growth_ratio: number
  avg_latency_ms: number
  unique_ips: number
  unique_models: number
  error_count: number
  error_rate: number
  rate_limited_count: number
  auth_error_count: number
  upstream_5xx_count: number
  timeout_count: number
  last_request_at?: string
  last_error_at?: string
}

export interface UserRiskIPRisk {
  shared_ip_count: number
  linked_user_count: number
  max_users_on_same_ip: number
  new_users_on_shared_ip: number
  same_ua_user_count: number
  auth_event_count: number
  register_event_count: number
}

export interface UserRiskLimits {
  legacy_concurrency: number
  legacy_rpm: number
  user_concurrency_override?: number | null
  user_rpm_limit_override?: number | null
}

export interface UserRiskAPIKeyStats {
  total: number
  active: number
  last_used_at?: string
  active_ratio: number
}

export interface UserRiskConcurrency {
  current_in_use: number
  waiting_in_queue: number
  max_capacity: number
  load_percentage: number
  collected: boolean
}

export interface UserRiskRecentError {
  id: number
  created_at: string
  phase: string
  type: string
  status_code: number
  message: string
  request_id: string
  model: string
}

export interface UserRiskItem {
  user: UserRiskUser
  score: number
  level: UserRiskLevel
  reasons: UserRiskReason[]
  metrics: UserRiskMetrics
  limits: UserRiskLimits
  api_keys: UserRiskAPIKeyStats
  concurrency: UserRiskConcurrency
  ip_risk: UserRiskIPRisk
  last_error?: UserRiskRecentError
}

export interface UserRiskListResponse {
  items: UserRiskItem[]
  total: number
  page: number
  page_size: number
  pages: number
  summary: UserRiskSummary
  window: UserRiskWindow
  generated_at: string
}

export interface UserRiskListParams {
  page?: number
  page_size?: number
  window?: '1h' | '24h' | '7d' | '30d'
  search?: string
  status?: '' | 'active' | 'disabled'
  risk_level?: '' | UserRiskLevel
  only_risky?: boolean
}

export interface UserRiskAPIKeyDetail {
  id: number
  name: string
  status: string
  group_id?: number
  group_name: string
  request_count: number
  error_count: number
  cost: number
  last_used_at?: string
  last_request_at?: string
}

export interface UserRiskTopModel {
  model: string
  request_count: number
  token_count: number
  cost: number
  last_used_at?: string
}

export interface UserRiskIPStat {
  ip: string
  request_count: number
  error_count: number
  last_seen_at?: string
}

export interface UserRiskLinkedUser {
  id: number
  email: string
  username: string
  status: string
  signup_source: string
  balance: number
  created_at: string
  request_count: number
  auth_event_count: number
  register_event_count: number
  last_seen_at?: string
  last_auth_event_at?: string
  shared_user_agent_hint: boolean
}

export interface UserRiskIPLink {
  ip: string
  source: string
  user_count: number
  other_user_count: number
  new_user_count: number
  same_ua_user_count: number
  request_count: number
  error_count: number
  auth_event_count: number
  register_event_count: number
  cost: number
  last_seen_at?: string
  linked_users: UserRiskLinkedUser[]
}

export interface UserRiskAuthEvent {
  id: number
  user_id?: number
  email: string
  event_type: string
  provider: string
  ip_address: string
  success: boolean
  reason: string
  created_at: string
}

export interface UserRiskDetail {
  item: UserRiskItem
  api_keys: UserRiskAPIKeyDetail[]
  recent_errors: UserRiskRecentError[]
  top_models: UserRiskTopModel[]
  top_ips: UserRiskIPStat[]
  ip_links: UserRiskIPLink[]
  auth_events: UserRiskAuthEvent[]
  window: UserRiskWindow
  generated_at: string
}

export async function list(params: UserRiskListParams): Promise<UserRiskListResponse> {
  const { data } = await apiClient.get<UserRiskListResponse>('/admin/user-risk', { params })
  return data
}

export async function getDetail(userID: number, window = '24h'): Promise<UserRiskDetail> {
  const { data } = await apiClient.get<UserRiskDetail>(`/admin/user-risk/${userID}`, {
    params: { window }
  })
  return data
}

const userRiskAPI = {
  list,
  getDetail
}

export default userRiskAPI
