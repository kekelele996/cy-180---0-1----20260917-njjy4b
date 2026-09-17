// 通用状态徽标组件，跨页面复用。
import { CONSENT_STATUS_TEXT, PROJECT_STATUS_TEXT, RECORDING_STATUS_TEXT } from '../constants'

interface StatusBadgeProps {
  status: string
  type?: 'project' | 'recording' | 'consent'
}

const STYLES: Record<string, string> = {
  draft: 'badge-draft',
  in_progress: 'badge-progress',
  completed: 'badge-completed',
  archived: 'badge-archived',
  recording: 'badge-recording',
  processing: 'badge-processing',
  ready: 'badge-ready',
  failed: 'badge-failed',
  pending: 'badge-pending',
  verified: 'badge-verified',
  revoked: 'badge-revoked',
}

const TEXT_MAP = {
  project: PROJECT_STATUS_TEXT,
  recording: RECORDING_STATUS_TEXT,
  consent: CONSENT_STATUS_TEXT,
}

export default function StatusBadge({ status, type = 'project' }: StatusBadgeProps) {
  const text = TEXT_MAP[type][status]
  return <span className={`status-badge ${STYLES[status] || 'badge-default'}`}>{text || status}</span>
}
