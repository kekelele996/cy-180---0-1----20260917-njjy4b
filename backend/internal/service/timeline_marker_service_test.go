package service

import (
	"log/slog"
	"testing"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
)

// fakeMarkerRepo 内存版时间轴节点仓储。
type fakeMarkerRepo struct {
	markers map[uint]*model.TimelineMarker
}

func (f *fakeMarkerRepo) Create(marker *model.TimelineMarker) error {
	marker.ID = uint(len(f.markers) + 1)
	f.markers[marker.ID] = marker
	return nil
}
func (f *fakeMarkerRepo) FindByID(id uint) (*model.TimelineMarker, error) {
	if m, ok := f.markers[id]; ok {
		return m, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeMarkerRepo) ListByProject(projectID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (f *fakeMarkerRepo) ListByRecording(recordingID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (f *fakeMarkerRepo) Update(marker *model.TimelineMarker) error {
	f.markers[marker.ID] = marker
	return nil
}
func (f *fakeMarkerRepo) Delete(id uint) error {
	delete(f.markers, id)
	return nil
}

func newMarkerSvc(consentStatus string) TimelineMarkerService {
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Title: "项目一", Status: constants.ProjectStatusInProgress},
		2: {ID: 2, Title: "项目二", Status: constants.ProjectStatusInProgress},
	}}
	recordingRepo := &fakeRecordingRepo{recordings: map[uint]*model.Recording{
		101: {ID: 101, ProjectID: 1, QuestionID: 11, Status: constants.RecordingStatusReady},
		202: {ID: 202, ProjectID: 2, QuestionID: 22, Status: constants.RecordingStatusReady},
	}}
	consentRepo := &fakeConsentRepo{}
	if consentStatus != "" {
		consentRepo.items = append(consentRepo.items, &model.Consent{
			ID: 1, ProjectID: 1, Status: consentStatus, RegisteredBy: 1,
		})
	}
	return NewTimelineMarkerService(&fakeMarkerRepo{markers: map[uint]*model.TimelineMarker{}}, projectRepo, recordingRepo, consentRepo, slog.Default())
}

func TestMarkerCreateConsentGate(t *testing.T) {
	actor := &model.User{ID: 1, Username: "archivist", Role: constants.RoleArchivist}
	cases := []struct {
		name          string
		consentStatus string
		wantCode      int
	}{
		{name: "no consent registered", consentStatus: "", wantCode: constants.CodeConsentRequired},
		{name: "pending consent blocks marker", consentStatus: constants.ConsentStatusPending, wantCode: constants.CodeConsentRequired},
		{name: "verified consent allows marker", consentStatus: constants.ConsentStatusVerified, wantCode: 0},
		{name: "revoked consent blocks marker immediately", consentStatus: constants.ConsentStatusRevoked, wantCode: constants.CodeConsentRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newMarkerSvc(tc.consentStatus)
			_, err := svc.Create(actor, &dto.CreateTimelineMarkerRequest{
				ProjectID: 1, RecordingID: 101, TimestampSecond: 12, Label: "关键节点",
			})
			if tc.wantCode == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			wantAppCode(t, err, tc.wantCode)
		})
	}
}

func TestMarkerCreateCrossProjectRecording(t *testing.T) {
	svc := newMarkerSvc(constants.ConsentStatusVerified)
	actor := &model.User{ID: 1, Username: "archivist", Role: constants.RoleArchivist}
	// 录音 202 属于项目 2，引用到项目 1 必须被拒绝
	_, err := svc.Create(actor, &dto.CreateTimelineMarkerRequest{
		ProjectID: 1, RecordingID: 202, TimestampSecond: 5, Label: "跨项目节点",
	})
	wantAppCode(t, err, constants.CodeCrossProject)
}
