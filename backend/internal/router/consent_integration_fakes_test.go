package router

import (
	"sync"

	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
)

// 内存版仓储集合，仅用于 HTTP 集成测试，模拟数据库行为（含 FOR UPDATE 语义：直接返回指针）。

type memDB struct {
	mu           sync.Mutex
	projects     map[uint]*model.Project
	questions    map[uint]*model.Question
	recordings   map[uint]*model.Recording
	markers      map[uint]*model.TimelineMarker
	consents     map[uint]*model.Consent // key: project_id
	auditLogs    []model.AuditLog
	consentSeq   uint
	recordingSeq uint
	markerSeq    uint
	questionSeq  uint
	auditSeq     uint
}

func newMemDB() *memDB {
	return &memDB{
		projects:   map[uint]*model.Project{},
		questions:  map[uint]*model.Question{},
		recordings: map[uint]*model.Recording{},
		markers:    map[uint]*model.TimelineMarker{},
		consents:   map[uint]*model.Consent{},
	}
}

// ---- project ----

type memProjectRepo struct{ db *memDB }

func (r *memProjectRepo) Create(p *model.Project) error { r.db.projects[p.ID] = p; return nil }
func (r *memProjectRepo) FindByID(id uint) (*model.Project, error) {
	if p, ok := r.db.projects[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}
func (r *memProjectRepo) List(page, pageSize int, status string) ([]model.Project, int64, error) {
	return nil, 0, nil
}
func (r *memProjectRepo) ListByUser(userID uint, page, pageSize int) ([]model.Project, int64, error) {
	return nil, 0, nil
}
func (r *memProjectRepo) FindByIDForUpdate(id uint) (*model.Project, error) { return r.FindByID(id) }
func (r *memProjectRepo) Update(p *model.Project) error                     { r.db.projects[p.ID] = p; return nil }
func (r *memProjectRepo) UpdateStatus(p *model.Project) error               { return r.Update(p) }
func (r *memProjectRepo) Delete(id uint) error                              { delete(r.db.projects, id); return nil }
func (r *memProjectRepo) Count() (int64, error)                             { return int64(len(r.db.projects)), nil }

// ---- question ----

type memQuestionRepo struct{ db *memDB }

func (r *memQuestionRepo) Create(q *model.Question) error { r.db.questions[q.ID] = q; return nil }
func (r *memQuestionRepo) FindByID(id uint) (*model.Question, error) {
	if q, ok := r.db.questions[id]; ok {
		return q, nil
	}
	return nil, repository.ErrNotFound
}
func (r *memQuestionRepo) ListByProject(projectID uint) ([]model.Question, error) { return nil, nil }
func (r *memQuestionRepo) Update(q *model.Question) error                         { r.db.questions[q.ID] = q; return nil }
func (r *memQuestionRepo) Delete(id uint) error                                   { delete(r.db.questions, id); return nil }
func (r *memQuestionRepo) CountByProject(p uint) (int64, error)                   { return 0, nil }

// ---- recording ----

type memRecordingRepo struct{ db *memDB }

func (r *memRecordingRepo) Create(rec *model.Recording) error {
	r.db.recordings[rec.ID] = rec
	return nil
}
func (r *memRecordingRepo) FindByID(id uint) (*model.Recording, error) {
	if rec, ok := r.db.recordings[id]; ok {
		return rec, nil
	}
	return nil, repository.ErrNotFound
}
func (r *memRecordingRepo) ListByProject(projectID uint) ([]model.Recording, error) { return nil, nil }
func (r *memRecordingRepo) ListByQuestion(questionID uint) ([]model.Recording, error) {
	return nil, nil
}
func (r *memRecordingRepo) FindByIDForUpdate(id uint) (*model.Recording, error) {
	return r.FindByID(id)
}
func (r *memRecordingRepo) Update(rec *model.Recording) error {
	r.db.recordings[rec.ID] = rec
	return nil
}
func (r *memRecordingRepo) UpdateStatus(rec *model.Recording) error { return r.Update(rec) }
func (r *memRecordingRepo) Delete(id uint) error                    { delete(r.db.recordings, id); return nil }
func (r *memRecordingRepo) CountByProject(p uint) (int64, error)    { return 0, nil }

// ---- marker ----

type memMarkerRepo struct{ db *memDB }

func (r *memMarkerRepo) Create(m *model.TimelineMarker) error { r.db.markers[m.ID] = m; return nil }
func (r *memMarkerRepo) FindByID(id uint) (*model.TimelineMarker, error) {
	if m, ok := r.db.markers[id]; ok {
		return m, nil
	}
	return nil, repository.ErrNotFound
}
func (r *memMarkerRepo) ListByProject(projectID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (r *memMarkerRepo) ListByRecording(recordingID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (r *memMarkerRepo) Update(m *model.TimelineMarker) error { r.db.markers[m.ID] = m; return nil }
func (r *memMarkerRepo) Delete(id uint) error                 { delete(r.db.markers, id); return nil }

// ---- consent ----

type memConsentRepo struct{ db *memDB }

func (r *memConsentRepo) Create(c *model.Consent) error {
	if _, ok := r.db.consents[c.ProjectID]; ok {
		return repository.ErrNotFound
	}
	r.db.consentSeq++
	c.ID = r.db.consentSeq
	r.db.consents[c.ProjectID] = c
	return nil
}
func (r *memConsentRepo) FindByID(id uint) (*model.Consent, error) {
	for _, c := range r.db.consents {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *memConsentRepo) FindByProjectID(projectID uint) (*model.Consent, error) {
	if c, ok := r.db.consents[projectID]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (r *memConsentRepo) FindByProjectIDForUpdate(projectID uint) (*model.Consent, error) {
	return r.FindByProjectID(projectID)
}
func (r *memConsentRepo) Update(c *model.Consent) error { r.db.consents[c.ProjectID] = c; return nil }

// ---- audit ----

type memAuditRepo struct{ db *memDB }

func (r *memAuditRepo) Create(l *model.AuditLog) error {
	r.db.auditSeq++
	l.ID = r.db.auditSeq
	r.db.auditLogs = append(r.db.auditLogs, *l)
	return nil
}
func (r *memAuditRepo) List(page, pageSize int, username string) ([]model.AuditLog, int64, error) {
	return r.db.auditLogs, int64(len(r.db.auditLogs)), nil
}
