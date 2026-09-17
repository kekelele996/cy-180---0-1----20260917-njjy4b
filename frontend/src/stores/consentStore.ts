// 受访者授权状态管理。
import { create } from 'zustand'
import { listConsents, registerConsent, revokeConsent, verifyConsent } from '../api/consent'
import type { Consent } from '../api/types'

interface ConsentState {
  consents: Consent[]
  current: Consent | null
  loading: boolean
  fetchByProject: (projectId: number) => Promise<void>
  register: (projectId: number, note: string) => Promise<void>
  verify: (id: number) => Promise<void>
  revoke: (id: number, reason: string) => Promise<void>
}

export const useConsentStore = create<ConsentState>((set, get) => ({
  consents: [],
  current: null,
  loading: false,

  async fetchByProject(projectId) {
    set({ loading: true })
    try {
      const res = await listConsents(projectId)
      set({ consents: res.list || [], current: res.current, loading: false })
    } catch (e) {
      console.error('fetch consents failed', e)
      set({ loading: false })
    }
  },

  async register(projectId, note) {
    await registerConsent({ project_id: projectId, note })
    await get().fetchByProject(projectId)
  },

  async verify(id) {
    const updated = await verifyConsent(id)
    set((s) => ({
      consents: s.consents.map((c) => (c.id === id ? updated : c)),
      current: s.current?.id === id ? updated : s.current,
    }))
  },

  async revoke(id, reason) {
    const updated = await revokeConsent(id, reason)
    set((s) => ({
      consents: s.consents.map((c) => (c.id === id ? updated : c)),
      current: s.current?.id === id ? updated : s.current,
    }))
  },
}))
