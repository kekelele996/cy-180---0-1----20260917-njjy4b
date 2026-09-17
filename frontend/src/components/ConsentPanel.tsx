// 受访者授权面板：采访员登记、档案员核验、管理员撤销；授权生效前锁定录音/标注入口。
import { useCallback, useEffect, useState } from 'react'
import ConsentBadge from '../components/ConsentBadge'
import {
  CONSENT_STATUS_PENDING,
  CONSENT_STATUS_REVOKED,
  CONSENT_STATUS_VERIFIED,
  PROJECT_STATUS_ARCHIVED,
  ROLE_ADMIN,
  ROLE_ARCHIVIST,
  ROLE_INTERVIEWER,
} from '../constants'
import { useAuthStore } from '../stores/authStore'
import { useConsentStore } from '../stores/consentStore'
import { formatDateTime } from '../utils/format'
import type { Consent } from '../api/types'

interface Props {
  projectId: number
  projectStatus: string
  onChanged?: (consent: Consent | null) => void
}

export default function ConsentPanel({ projectId, projectStatus, onChanged }: Props) {
  const user = useAuthStore((s) => s.user)
  const { consents, fetchByProject, register, verify, revoke } = useConsentStore()
  const consent = consents[projectId] || null
  const archived = projectStatus === PROJECT_STATUS_ARCHIVED

  const [registering, setRegistering] = useState(false)
  const [intervieweeName, setIntervieweeName] = useState('')
  const [scope, setScope] = useState('')
  const [statement, setStatement] = useState('')
  const [revoking, setRevoking] = useState(false)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<{ kind: 'info' | 'error'; text: string } | null>(null)

  useEffect(() => {
    fetchByProject(projectId).then((c) => onChanged?.(c))
  }, [projectId, fetchByProject, onChanged])

  const flash = useCallback((kind: 'info' | 'error', text: string) => {
    setMessage({ kind, text })
    window.setTimeout(() => setMessage(null), 4000)
  }, [])

  const handleRegister = async () => {
    if (!intervieweeName.trim() || !scope.trim()) {
      flash('error', '请填写受访者姓名与授权范围')
      return
    }
    setBusy(true)
    try {
      const c = await register(projectId, {
        interviewee_name: intervieweeName.trim(),
        scope: scope.trim(),
        statement: statement.trim(),
      })
      setRegistering(false)
      setIntervieweeName('')
      setScope('')
      setStatement('')
      onChanged?.(c)
      flash('info', '授权已登记，等待档案员核验')
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '授权登记失败')
    } finally {
      setBusy(false)
    }
  }

  const handleVerify = async () => {
    setBusy(true)
    try {
      const c = await verify(projectId)
      onChanged?.(c)
      flash('info', '授权核验通过，录音与时间轴标注已解锁')
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '授权核验失败')
    } finally {
      setBusy(false)
    }
  }

  const handleRevoke = async () => {
    if (!reason.trim()) {
      flash('error', '撤销必须填写原因')
      return
    }
    setBusy(true)
    try {
      const c = await revoke(projectId, reason.trim())
      setRevoking(false)
      setReason('')
      onChanged?.(c)
      flash('error', '授权已撤销，后续录音上传与时间轴标注已立即阻止；已有材料仍可复核')
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '授权撤销失败')
    } finally {
      setBusy(false)
    }
  }

  const canRegister = !!user && (user.role === ROLE_INTERVIEWER || user.role === ROLE_ADMIN)
  const canVerify = !!user && (user.role === ROLE_ARCHIVIST || user.role === ROLE_ADMIN)
  const canRevoke = !!user && user.role === ROLE_ADMIN

  return (
    <section className="card">
      <div className="card-title">
        受访者授权（知情同意）
        {consent && (
          <span style={{ marginLeft: 12 }}>
            <ConsentBadge status={consent.status} />
          </span>
        )}
      </div>

      {message && <div className={`toast ${message.kind === 'error' ? 'error' : 'success'}`}>{message.text}</div>}

      {archived && (
        <div className="consent-notice">项目已归档，授权信息只读，不能登记、核验或撤销。</div>
      )}

      {!consent ? (
        <div>
          <p className="muted">尚未登记受访者授权。授权生效前，该项目不能新增录音或时间轴节点。</p>
          {registering ? (
            <div className="consent-form">
              <label>
                受访者姓名
                <input value={intervieweeName} onChange={(e) => setIntervieweeName(e.target.value)} maxLength={64} />
              </label>
              <label>
                授权范围（如：采集、整理、内部研究、公开播放）
                <input value={scope} onChange={(e) => setScope(e.target.value)} maxLength={512} />
              </label>
              <label>
                授权说明（可选，纸质同意书编号/签署方式等）
                <input value={statement} onChange={(e) => setStatement(e.target.value)} maxLength={512} />
              </label>
              <div className="row-actions">
                <button className="btn btn-primary btn-small" disabled={busy} onClick={handleRegister}>
                  {busy ? '提交中…' : '提交登记'}
                </button>
                <button className="btn btn-plain btn-small" disabled={busy} onClick={() => setRegistering(false)}>
                  取消
                </button>
              </div>
            </div>
          ) : (
            canRegister &&
            !archived && (
              <button className="btn btn-primary btn-small" onClick={() => setRegistering(true)}>
                登记受访者授权
              </button>
            )
          )}
          {!canRegister && !registering && (
            <p className="muted">仅采访员可登记授权，请联系项目采访员。</p>
          )}
        </div>
      ) : (
        <div className="consent-detail">
          <div className="detail-grid">
            <div>
              <div className="detail-label">受访者</div>
              <div className="detail-value">{consent.interviewee_name}</div>
            </div>
            <div>
              <div className="detail-label">授权范围</div>
              <div className="detail-value">{consent.scope}</div>
            </div>
            <div>
              <div className="detail-label">登记人 / 时间</div>
              <div className="detail-value">
                {consent.registered_by_name || `用户#${consent.registered_by}`} · {formatDateTime(consent.registered_at)}
              </div>
            </div>
            {consent.verified_at && (
              <div>
                <div className="detail-label">核验人 / 时间</div>
                <div className="detail-value">
                  {consent.verified_by_name || `用户#${consent.verified_by}`} · {formatDateTime(consent.verified_at || '')}
                </div>
              </div>
            )}
            {consent.statement && (
              <div>
                <div className="detail-label">授权说明</div>
                <div className="detail-value">{consent.statement}</div>
              </div>
            )}
          </div>

          {consent.status === CONSENT_STATUS_REVOKED && (
            <div className="consent-revoked">
              <div>
                <strong>授权已撤销</strong>（{consent.revoked_by_name || `用户#${consent.revoked_by}`} ·{' '}
                {formatDateTime(consent.revoked_at || '')}）
              </div>
              <div>撤销原因：{consent.revoke_reason}</div>
              <div className="muted">后续录音上传与时间轴标注已阻止；已有材料保留，管理员可继续复核。</div>
            </div>
          )}

          {!archived && (
            <div className="row-actions" style={{ marginTop: 12 }}>
              {consent.status === CONSENT_STATUS_PENDING && canVerify && (
                <button className="btn btn-primary btn-small" disabled={busy} onClick={handleVerify}>
                  {busy ? '核验中…' : '核验通过（档案员）'}
                </button>
              )}
              {consent.status === CONSENT_STATUS_VERIFIED && canRevoke && !revoking && (
                <button className="btn btn-danger btn-small" onClick={() => setRevoking(true)}>
                  撤销授权（管理员）
                </button>
              )}
              {consent.status === CONSENT_STATUS_PENDING && canRevoke && !revoking && (
                <button className="btn btn-danger btn-small" onClick={() => setRevoking(true)}>
                  撤销授权（管理员）
                </button>
              )}
            </div>
          )}

          {revoking && canRevoke && !archived && (
            <div className="consent-form">
              <label>
                撤销原因（必填）
                <textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  maxLength={512}
                  rows={3}
                  placeholder="例如：受访者事后要求撤回公开播放授权"
                />
              </label>
              <div className="row-actions">
                <button className="btn btn-danger btn-small" disabled={busy} onClick={handleRevoke}>
                  {busy ? '撤销中…' : '确认撤销并立即阻止上传'}
                </button>
                <button
                  className="btn btn-plain btn-small"
                  disabled={busy}
                  onClick={() => {
                    setRevoking(false)
                    setReason('')
                  }}
                >
                  取消
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
