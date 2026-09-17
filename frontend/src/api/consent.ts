import { get, post, put } from '../utils/request'
import type { Consent } from './types'

export function listConsents(projectId: number) {
  return get<{ list: Consent[]; current: Consent | null }>('/consents', { project_id: projectId })
}

export function registerConsent(payload: { project_id: number; note?: string }) {
  return post<Consent>('/consents', payload)
}

export function verifyConsent(id: number) {
  return put<Consent>(`/consents/${id}/verify`)
}

export function revokeConsent(id: number, reason: string) {
  return put<Consent>(`/consents/${id}/revoke`, { reason })
}
