package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUnpinPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	uc := NewUnpinPageUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		pageRepo,
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)
	pinnedAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	f := setupPagePinFixture(t, tx, "punpin", &pinnedAt)

	tests := []struct {
		name       string
		userID     model.UserID
		pageNumber int32
		wantErr    bool
		wantCode   model.AppErrorCode
	}{
		{name: "正常系: スペース編集者はピンを外せる", userID: f.editorID, pageNumber: pagePinPublicPageNumber},
		{name: "正常系: トピック編集者のロールでもピンを外せる", userID: f.topicEditorID, pageNumber: pagePinTopicEditPageNumber},
		{name: "異常系: スペース閲覧者はピンを外せない", userID: f.viewerID, pageNumber: pagePinViewerPageNumber, wantErr: true, wantCode: model.AppErrCodeForbidden},
		{name: "異常系: スペースメンバーでないユーザーはピンを外せない", userID: f.nonMemberID, pageNumber: pagePinGuestPageNumber, wantErr: true, wantCode: model.AppErrCodeForbidden},
		{name: "異常系: 開けないトピックのページは見つからない", userID: f.nonMemberID, pageNumber: pagePinPrivatePageNumber, wantErr: true, wantCode: model.AppErrCodeResourceNotFound},
		{name: "異常系: 存在しないページは見つからない", userID: f.editorID, pageNumber: 999, wantErr: true, wantCode: model.AppErrCodeResourceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), UnpinPageInput{
				SpaceIdentifier: f.spaceIdentifier,
				PageNumber:      tt.pageNumber,
				UserID:          tt.userID,
			})

			if tt.wantErr {
				ae := model.AsAppError(err)
				if ae == nil {
					t.Fatalf("AppErrorを期待したが、エラー = %v", err)
				}
				if ae.Code != tt.wantCode {
					t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, tt.wantCode)
				}
				if tt.pageNumber != 999 {
					if page := findPagePinTestPage(t, pageRepo, f.spaceID, model.PageNumber(tt.pageNumber)); page.PinnedAt == nil {
						t.Error("page.PinnedAt = nil、期待値 = 元の日時 (拒否したので更新されない)")
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Space == nil || output.Space.Identifier != f.spaceIdentifier {
				t.Errorf("output.Space = %v、期待値 = スペース%s", output.Space, f.spaceIdentifier)
			}
			if page := findPagePinTestPage(t, pageRepo, f.spaceID, model.PageNumber(tt.pageNumber)); page.PinnedAt != nil {
				t.Errorf("page.PinnedAt = %v、期待値 = nil", page.PinnedAt)
			}
		})
	}
}
