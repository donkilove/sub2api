/**
 * Admin Error Passthrough Rules API endpoints
 * Handles error passthrough rule management for administrators
 */

import { apiClient } from '../client'

/**
 * Error passthrough rule interface
 */
export interface ErrorPassthroughRule {
  id: number
  name: string
  enabled: boolean
  priority: number
  error_codes: number[]
  keywords: string[]
  match_mode: 'any' | 'all'
  platforms: string[]
  passthrough_code: boolean
  response_code: number | null
  passthrough_body: boolean
  custom_message: string | null
  skip_monitoring: boolean
  description: string | null
  created_at: string
  updated_at: string
}

/**
 * Upstream error policy shown in the admin strategy list.
 */
export interface UpstreamErrorPolicy {
  category: string
  label: string
  description: string
  default_status_code: number
  default_error_type: string
  default_message: string
  default_retryable: boolean
  custom_enabled: boolean
  status_code?: number | null
  error_type?: string
  message?: string
  retry_enabled: boolean
  max_retries: number
  note?: string
  effective_status_code: number
  effective_error_type: string
  effective_message: string
}

/**
 * Update upstream error policy request.
 */
export interface UpdateUpstreamErrorPolicyRequest {
  custom_enabled?: boolean
  status_code?: number | null
  error_type?: string
  message?: string
  retry_enabled?: boolean
  max_retries?: number
  note?: string
}

/**
 * Create rule request
 */
export interface CreateRuleRequest {
  name: string
  enabled?: boolean
  priority?: number
  error_codes?: number[]
  keywords?: string[]
  match_mode?: 'any' | 'all'
  platforms?: string[]
  passthrough_code?: boolean
  response_code?: number | null
  passthrough_body?: boolean
  custom_message?: string | null
  skip_monitoring?: boolean
  description?: string | null
}

/**
 * Update rule request
 */
export interface UpdateRuleRequest {
  name?: string
  enabled?: boolean
  priority?: number
  error_codes?: number[]
  keywords?: string[]
  match_mode?: 'any' | 'all'
  platforms?: string[]
  passthrough_code?: boolean
  response_code?: number | null
  passthrough_body?: boolean
  custom_message?: string | null
  skip_monitoring?: boolean
  description?: string | null
}

/**
 * List all error passthrough rules
 * @returns List of all rules sorted by priority
 */
export async function list(): Promise<ErrorPassthroughRule[]> {
  const { data } = await apiClient.get<ErrorPassthroughRule[]>('/admin/error-passthrough-rules')
  return data
}

/**
 * Get rule by ID
 * @param id - Rule ID
 * @returns Rule details
 */
export async function getById(id: number): Promise<ErrorPassthroughRule> {
  const { data } = await apiClient.get<ErrorPassthroughRule>(`/admin/error-passthrough-rules/${id}`)
  return data
}

/**
 * Create new rule
 * @param ruleData - Rule data
 * @returns Created rule
 */
export async function create(ruleData: CreateRuleRequest): Promise<ErrorPassthroughRule> {
  const { data } = await apiClient.post<ErrorPassthroughRule>('/admin/error-passthrough-rules', ruleData)
  return data
}

/**
 * Update rule
 * @param id - Rule ID
 * @param updates - Fields to update
 * @returns Updated rule
 */
export async function update(id: number, updates: UpdateRuleRequest): Promise<ErrorPassthroughRule> {
  const { data } = await apiClient.put<ErrorPassthroughRule>(`/admin/error-passthrough-rules/${id}`, updates)
  return data
}

/**
 * Delete rule
 * @param id - Rule ID
 * @returns Success confirmation
 */
export async function deleteRule(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/error-passthrough-rules/${id}`)
  return data
}

/**
 * Toggle rule enabled status
 * @param id - Rule ID
 * @param enabled - New enabled status
 * @returns Updated rule
 */
export async function toggleEnabled(id: number, enabled: boolean): Promise<ErrorPassthroughRule> {
  return update(id, { enabled })
}

/**
 * List all upstream error policies.
 * @returns Default policies merged with saved overrides
 */
export async function listPolicies(): Promise<UpstreamErrorPolicy[]> {
  const { data } = await apiClient.get<UpstreamErrorPolicy[]>('/admin/error-passthrough-rules/policies')
  return data
}

/**
 * Update one upstream error policy.
 * @param category - Error category key
 * @param updates - Policy override fields
 * @returns Updated effective policy
 */
export async function updatePolicy(
  category: string,
  updates: UpdateUpstreamErrorPolicyRequest
): Promise<UpstreamErrorPolicy> {
  const { data } = await apiClient.put<UpstreamErrorPolicy>(
    `/admin/error-passthrough-rules/policies/${encodeURIComponent(category)}`,
    updates
  )
  return data
}

export const errorPassthroughAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteRule,
  toggleEnabled,
  listPolicies,
  updatePolicy
}

export default errorPassthroughAPI
