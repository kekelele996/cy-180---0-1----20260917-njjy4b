import { get, post } from '../utils/request'
import type { Consent } from './types'

export interface RegisterConsentPayload {
  interviewee_name: string
  scope: string
  statement?: string
}

// 查询项目的受访者授权（未登记时后端返回 404，调用方按 null 处理）。
export function getConsent(projectId: number) {
  return get<Consent>(`/projects/${projectId}/consent`)
}

export function registerConsent(projectId: number, payload: RegisterConsentPayload) {
  return post<Consent>(`/projects/${projectId}/consent/register`, payload)
}

export function verifyConsent(projectId: number) {
  return post<Consent>(`/projects/${projectId}/consent/verify`)
}

export function revokeConsent(projectId: number, reason: string) {
  return post<Consent>(`/projects/${projectId}/consent/revoke`, { reason })
}
