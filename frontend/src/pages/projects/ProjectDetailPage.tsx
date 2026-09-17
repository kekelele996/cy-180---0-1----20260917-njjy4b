// 项目详情页：基本信息、受访者授权、采访问题、时间线（录音片段 + 关键节点 + 一句话摘要）。
import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import AudioPlayer from '../../components/AudioPlayer'
import ConfirmDialog from '../../components/ConfirmDialog'
import EmptyState from '../../components/EmptyState'
import StatusBadge from '../../components/StatusBadge'
import {
  CONSENT_STATUS_PENDING,
  CONSENT_STATUS_REVOKED,
  CONSENT_STATUS_VERIFIED,
  PROJECT_STATUS_ARCHIVED,
  PROJECT_STATUS_COMPLETED,
  PROJECT_STATUS_IN_PROGRESS,
  ROLE_ADMIN,
  ROLE_ARCHIVIST,
  ROLE_INTERVIEWER,
} from '../../constants'
import { useAuthStore } from '../../stores/authStore'
import { useConsentStore } from '../../stores/consentStore'
import { useProjectStore } from '../../stores/projectStore'
import { useQuestionStore } from '../../stores/questionStore'
import { useRecordingStore } from '../../stores/recordingStore'
import { useTimelineStore } from '../../stores/timelineStore'
import { formatDateTime, formatDuration } from '../../utils/format'
import type { Recording, TimelineMarker } from '../../api/types'

export default function ProjectDetailPage() {
  const { id } = useParams()
  const projectId = Number(id)
  const navigate = useNavigate()
  const { detail, fetchDetail, transitionStatus, remove } = useProjectStore()
  const { questions, fetchByProject, create: createQuestion, remove: removeQuestion } = useQuestionStore()
  const { recordings, fetchByProject: fetchRecordings } = useRecordingStore()
  const { markers, fetchByProject: fetchMarkers, create: createMarker } = useTimelineStore()
  const { current: currentConsent, fetchByProject: fetchConsents } = useConsentStore()
  const [newQuestion, setNewQuestion] = useState('')
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (projectId) {
      fetchDetail(projectId)
      fetchByProject(projectId)
      fetchRecordings(projectId)
      fetchMarkers(projectId)
      fetchConsents(projectId)
    }
  }, [projectId, fetchDetail, fetchByProject, fetchRecordings, fetchMarkers, fetchConsents])

  const handleAddQuestion = useCallback(async () => {
    if (!newQuestion.trim()) return
    await createQuestion(projectId, newQuestion.trim())
    setNewQuestion('')
    setMessage('采访问题已添加')
    setTimeout(() => setMessage(''), 3000)
  }, [createQuestion, newQuestion, projectId])

  const markersOf = (recordingId: number) => markers.filter((m) => m.recording_id === recordingId)
  const consentEffective = currentConsent?.status === CONSENT_STATUS_VERIFIED
  const archived = detail?.status === PROJECT_STATUS_ARCHIVED

  if (!detail) {
    return <div className="page">加载中…</div>
  }

  return (
    <div className="page">
      {message && <div className="toast success">{message}</div>}
      <div className="page-header">
        <button className="btn btn-plain" onClick={() => navigate('/')}>
          ← 返回列表
        </button>
        <h2>{detail.title}</h2>
        <StatusBadge status={detail.status} type="project" />
      </div>

      <section className="card">
        <div className="card-title">项目信息</div>
        <div className="detail-grid">
          <div>
            <div className="detail-label">受访者</div>
            <div className="detail-value">{detail.interviewee_name}</div>
          </div>
          <div>
            <div className="detail-label">出生年份</div>
            <div className="detail-value">{detail.birth_year}</div>
          </div>
          <div>
            <div className="detail-label">创建时间</div>
            <div className="detail-value">{formatDateTime(detail.created_at)}</div>
          </div>
          <div>
            <div className="detail-label">背景简介</div>
            <div className="detail-value">{detail.background || '-'}</div>
          </div>
        </div>
        <div className="row-actions" style={{ marginTop: 12 }}>
          {detail.status !== PROJECT_STATUS_ARCHIVED && (
            <button
              className="btn btn-primary btn-small"
              onClick={async () => {
                const next =
                  detail.status === PROJECT_STATUS_IN_PROGRESS ? PROJECT_STATUS_COMPLETED : PROJECT_STATUS_IN_PROGRESS
                await transitionStatus(projectId, next)
                setMessage('项目状态已更新')
                setTimeout(() => setMessage(''), 3000)
              }}
            >
              {detail.status === PROJECT_STATUS_IN_PROGRESS ? '标记为已完成' : '开始采访'}
            </button>
          )}
          <ConfirmDialog
            title="删除采访项目"
            message="确定删除该项目吗？此操作不可恢复。"
            danger
            confirmText="删除"
            onConfirm={async () => {
              await remove(projectId)
              navigate('/')
            }}
          >
            <button className="btn btn-danger btn-small">删除项目</button>
          </ConfirmDialog>
          <LinkToInterview projectId={projectId} />
        </div>
      </section>

      <ConsentSection projectId={projectId} archived={archived} onMessage={setMessage} />

      <section className="card">
        <div className="card-title">采访问题</div>
        {questions.length === 0 ? (
          <EmptyState title="还没有采访问题" description="添加采访问题，作为录音的提纲" />
        ) : (
          <ul className="question-list">
            {questions.map((q) => (
              <li key={q.id} className="question-item">
                <span className="question-index">{q.sort_order + 1}</span>
                <span className="question-content">{q.content}</span>
                <ConfirmDialog
                  title="删除采访问题"
                  message="删除问题将同时删除其下的录音片段，确定继续？"
                  danger
                  confirmText="删除"
                  onConfirm={() => removeQuestion(q.id)}
                >
                  <button className="btn btn-plain btn-small">删除</button>
                </ConfirmDialog>
              </li>
            ))}
          </ul>
        )}
        <div className="inline-form">
          <input value={newQuestion} onChange={(e) => setNewQuestion(e.target.value)} placeholder="输入新的采访问题" />
          <button className="btn btn-primary" onClick={handleAddQuestion} disabled={!newQuestion.trim()}>
            添加问题
          </button>
        </div>
      </section>

      <section className="card">
        <div className="card-title">时间线 · 采访片段</div>
        {!consentEffective && (
          <div className="notice warning">
            受访者授权未生效，暂不能新增录音与时间轴节点
            {archived ? '（项目已归档，授权只读）' : ''}
          </div>
        )}
        {recordings.length === 0 ? (
          <EmptyState title="还没有录音片段" description="前往采访工作台开始录音，片段将按时间线展示" />
        ) : (
          <div className="timeline">
            {recordings.map((r) => (
              <TimelineItem
                key={r.id}
                recording={r}
                markers={markersOf(r.id)}
                onCreateMarker={createMarker}
                canAnnotate={consentEffective && !archived}
              />
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

function LinkToInterview({ projectId }: { projectId: number }) {
  return (
    <a className="btn btn-plain btn-small" href={`#/interview?project_id=${projectId}`}>
      前往采访工作台
    </a>
  )
}

// ConsentSection 受访者授权卡片：采访员登记、档案员核验、管理员撤销，归档后只读。
function ConsentSection({
  projectId,
  archived,
  onMessage,
}: {
  projectId: number
  archived: boolean
  onMessage: (msg: string) => void
}) {
  const { consents, current, register, verify, revoke } = useConsentStore()
  const { hasRole } = useAuthStore()
  const [note, setNote] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  const run = async (action: () => Promise<void>, ok: string) => {
    setError('')
    try {
      await action()
      onMessage(ok)
      setTimeout(() => onMessage(''), 3000)
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    }
  }

  const canRegister = hasRole(ROLE_INTERVIEWER) && !archived && (!current || current.status === CONSENT_STATUS_REVOKED)
  const canVerify = hasRole(ROLE_ARCHIVIST) && !archived && current?.status === CONSENT_STATUS_PENDING
  const canRevoke = hasRole(ROLE_ADMIN) && !archived && !!current && current.status !== CONSENT_STATUS_REVOKED

  return (
    <section className="card">
      <div className="card-title">
        受访者授权{' '}
        {current ? <StatusBadge status={current.status} type="consent" /> : <span className="muted">未登记</span>}
      </div>
      {error && <div className="notice error">{error}</div>}
      {archived && <div className="muted">项目已归档，授权信息只读。</div>}
      {current && (
        <div className="detail-grid" style={{ marginBottom: 8 }}>
          <div>
            <div className="detail-label">登记备注</div>
            <div className="detail-value">{current.note || '-'}</div>
          </div>
          <div>
            <div className="detail-label">登记时间</div>
            <div className="detail-value">{formatDateTime(current.created_at)}</div>
          </div>
          {current.status === CONSENT_STATUS_REVOKED && (
            <div>
              <div className="detail-label">撤销原因</div>
              <div className="detail-value">{current.revoke_reason || '-'}</div>
            </div>
          )}
        </div>
      )}
      {canRegister && (
        <div className="inline-form">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="授权范围备注，如：已签署纸质授权书"
          />
          <button
            className="btn btn-primary"
            onClick={() =>
              run(async () => {
                await register(projectId, note.trim())
                setNote('')
              }, '授权已登记，等待档案员核验')
            }
          >
            登记授权
          </button>
        </div>
      )}
      {canVerify && current && (
        <div className="row-actions">
          <button className="btn btn-primary btn-small" onClick={() => run(() => verify(current.id), '授权已核验生效')}>
            ✓ 核验授权
          </button>
        </div>
      )}
      {canRevoke && current && (
        <div className="inline-form">
          <input
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="撤销原因（必填），撤销后立即阻止后续上传"
          />
          <ConfirmDialog
            title="撤销受访者授权"
            message={`确定撤销该授权吗？撤销原因：${reason || '（未填写）'}。撤销后立即阻止新增录音与时间轴节点，已有材料保留供复核。`}
            danger
            confirmText="撤销授权"
            onConfirm={() =>
              run(async () => {
                await revoke(current.id, reason.trim())
                setReason('')
              }, '授权已撤销')
            }
          >
            <button className="btn btn-danger" disabled={!reason.trim()}>
              撤销授权
            </button>
          </ConfirmDialog>
        </div>
      )}
      {consents.length > 1 && (
        <div className="marker-list" style={{ marginTop: 8 }}>
          {consents.map((c) => (
            <span key={c.id} className="marker-chip">
              #{c.id} <StatusBadge status={c.status} type="consent" /> {formatDateTime(c.created_at)}
              {c.revoke_reason ? `（${c.revoke_reason}）` : ''}
            </span>
          ))}
        </div>
      )}
    </section>
  )
}

function TimelineItem({
  recording,
  markers,
  onCreateMarker,
  canAnnotate,
}: {
  recording: Recording
  markers: TimelineMarker[]
  onCreateMarker: (payload: {
    project_id: number
    recording_id: number
    timestamp_second: number
    label: string
    note?: string
  }) => Promise<void>
  canAnnotate: boolean
}) {
  const [label, setLabel] = useState('')
  const question = useQuestionStore((s) => s.questions.find((q) => q.id === recording.question_id))

  return (
    <div className="timeline-item">
      <div className="timeline-dot" />
      <div className="timeline-content">
        <div className="timeline-head">
          <span className="timeline-q">{question ? `问题：${question.content}` : `问题 #${recording.question_id}`}</span>
          <StatusBadge status={recording.status} type="recording" />
          <span className="timeline-duration">{formatDuration(recording.duration_seconds)}</span>
        </div>
        <AudioPlayer recordingId={recording.id} durationSeconds={recording.duration_seconds} />
        <div className="timeline-summary">
          <span className="summary-label">一句话摘要：</span>
          {recording.summary || <span className="muted">暂无摘要</span>}
        </div>
        {markers.length > 0 && (
          <div className="marker-list">
            {markers.map((m) => (
              <span key={m.id} className="marker-chip">
                ⏱ {formatDuration(m.timestamp_second)} · {m.label}
                {m.note ? `（${m.note}）` : ''}
              </span>
            ))}
          </div>
        )}
        {canAnnotate ? (
          <div className="inline-form">
            <input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="标注关键节点，如：回忆童年故居"
            />
            <button
              className="btn btn-plain btn-small"
              disabled={!label.trim()}
              onClick={async () => {
                await onCreateMarker({
                  project_id: recording.project_id,
                  recording_id: recording.id,
                  timestamp_second: recording.duration_seconds > 0 ? Math.floor(recording.duration_seconds / 2) : 0,
                  label: label.trim(),
                })
                setLabel('')
              }}
            >
              ＋ 标注节点
            </button>
          </div>
        ) : (
          <div className="muted">授权未生效或项目已归档，时间轴节点只读</div>
        )}
      </div>
    </div>
  )
}
