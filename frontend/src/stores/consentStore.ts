// 受访者授权状态管理（按项目维度缓存当前详情项目的授权）。
import { create } from 'zustand'
import {
  getConsent,
  registerConsent,
  revokeConsent,
  verifyConsent,
  type RegisterConsentPayload,
} from '../api/consent'
import type { Consent } from '../api/types'

interface ConsentState {
  // 以 project_id 为键，支持多页面切换时保留已加载授权
  consents: Record<number, Consent>
  loading: boolean
  fetchByProject: (projectId: number) => Promise<Consent | null>
  register: (projectId: number, payload: RegisterConsentPayload) => Promise<Consent>
  verify: (projectId: number) => Promise<Consent>
  revoke: (projectId: number, reason: string) => Promise<Consent>
}

export const useConsentStore = create<ConsentState>((set) => ({
  consents: {},
  loading: false,

  async fetchByProject(projectId) {
    set({ loading: true })
    try {
      const consent = await getConsent(projectId)
      set((s) => ({ consents: { ...s.consents, [projectId]: consent }, loading: false }))
      return consent
    } catch {
      // 404 表示尚未登记授权，页面按「未登记」态渲染。
      set((s) => {
        const next = { ...s.consents }
        delete next[projectId]
        return { consents: next, loading: false }
      })
      return null
    }
  },

  async register(projectId, payload) {
    const consent = await registerConsent(projectId, payload)
    set((s) => ({ consents: { ...s.consents, [projectId]: consent } }))
    return consent
  },

  async verify(projectId) {
    const consent = await verifyConsent(projectId)
    set((s) => ({ consents: { ...s.consents, [projectId]: consent } }))
    return consent
  },

  async revoke(projectId, reason) {
    const consent = await revokeConsent(projectId, reason)
    set((s) => ({ consents: { ...s.consents, [projectId]: consent } }))
    return consent
  },
}))
