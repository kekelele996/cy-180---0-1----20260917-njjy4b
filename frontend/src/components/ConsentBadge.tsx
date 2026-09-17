// 受访者授权状态徽标。
import { CONSENT_STATUS_TEXT } from '../constants'

const STYLES: Record<string, string> = {
  pending: 'badge-processing',
  verified: 'badge-ready',
  revoked: 'badge-revoked',
}

export default function ConsentBadge({ status }: { status: string }) {
  return (
    <span className={`status-badge ${STYLES[status] || 'badge-default'}`}>
      {CONSENT_STATUS_TEXT[status] || status}
    </span>
  )
}
