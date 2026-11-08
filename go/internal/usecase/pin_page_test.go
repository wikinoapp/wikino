package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// pagePinFixtureは、ピンを打つ・外すUseCaseのテストで使うスペース・メンバー・ページの組。
type pagePinFixture struct {
	spaceIdentifier model.SpaceIdentifier
	spaceID         model.SpaceID
	// editorIDはスペース編集者 (page_pin:*を持つ)
	editorID model.UserID
	// viewerIDはスペース閲覧者 (page_pin:*を持たない)
	viewerID model.UserID
	// topicEditorIDはスペース閲覧者で、非公開トピックのトピック編集者
	topicEditorID model.UserID
	// nonMemberIDはスペースに参加していないログイン済みユーザー
	nonMemberID model.UserID
}

// 公開トピックのページ番号
const (
	pagePinPublicPageNumber = 1
	pagePinViewerPageNumber = 2
	pagePinGuestPageNumber  = 3
)

// 非公開トピックのページ番号
const (
	pagePinPrivatePageNumber   = 4
	pagePinTopicEditPageNumber = 5
)

// setupPagePinFixtureは、prefixで識別子を分けたスペースを作る。pinnedAtがnilでなければ、
// すべてのページにピンを打った状態で作る。
func setupPagePinFixture(t *testing.T, tx *sql.Tx, prefix string, pinnedAt *time.Time) pagePinFixture {
	t.Helper()

	newUser := func(name string) model.UserID {
		return testutil.NewUserBuilder(t, tx).
			WithEmail(prefix + "-" + name + "@example.com").
			WithAtname(prefix + name).
			Build()
	}
	f := pagePinFixture{
		spaceIdentifier: model.SpaceIdentifier(prefix + "-space"),
		editorID:        newUser("editor"),
		viewerID:        newUser("viewer"),
		topicEditorID:   newUser("topiceditor"),
		nonMemberID:     newUser("nonmember"),
	}

	f.spaceID = testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(string(f.spaceIdentifier)).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(f.editorID).
		WithRole(model.SpaceRoleEditor).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(f.viewerID).
		WithRole(model.SpaceRoleViewer).
		Build()
	topicEditorSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(f.topicEditorID).
		WithRole(model.SpaceRoleViewer).
		Build()

	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithNumber(2).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithTopicID(privateTopicID).
		WithSpaceMemberID(topicEditorSpaceMemberID).
		WithRole(model.TopicRoleEditor).
		Build()

	newPage := func(topicID model.TopicID, number model.PageNumber, title string) {
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(number).
			WithTitle(title).
			WithLinkedPageIDs([]model.PageID{})
		if pinnedAt != nil {
			b = b.WithPinnedAt(*pinnedAt)
		}
		b.Build()
	}
	newPage(publicTopicID, pagePinPublicPageNumber, "Public Page")
	newPage(publicTopicID, pagePinViewerPageNumber, "Viewer Page")
	newPage(publicTopicID, pagePinGuestPageNumber, "Guest Page")
	newPage(privateTopicID, pagePinPrivatePageNumber, "Private Page")
	newPage(privateTopicID, pagePinTopicEditPageNumber, "Topic Editor Page")

	return f
}

func findPagePinTestPage(t *testing.T, pageRepo *repository.PageRepository, spaceID model.SpaceID, number model.PageNumber) *model.Page {
	t.Helper()

	page, err := pageRepo.FindBySpaceAndNumber(context.Background(), spaceID, number)
	if err != nil {
		t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
	}
	if page == nil {
		t.Fatal("FindBySpaceAndNumber()がnilを返した、期待値 = ページ")
	}
	return page
}

func TestPinPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	uc := NewPinPageUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		pageRepo,
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)
	f := setupPagePinFixture(t, tx, "ppin", nil)

	tests := []struct {
		name       string
		userID     model.UserID
		pageNumber int32
		wantErr    bool
		wantCode   model.AppErrorCode
	}{
		{name: "正常系: スペース編集者はピンを打てる", userID: f.editorID, pageNumber: pagePinPublicPageNumber},
		{name: "正常系: トピック編集者のロールでもピンを打てる", userID: f.topicEditorID, pageNumber: pagePinTopicEditPageNumber},
		{name: "異常系: スペース閲覧者はピンを打てない", userID: f.viewerID, pageNumber: pagePinViewerPageNumber, wantErr: true, wantCode: model.AppErrCodeForbidden},
		{name: "異常系: スペースメンバーでないユーザーはピンを打てない", userID: f.nonMemberID, pageNumber: pagePinGuestPageNumber, wantErr: true, wantCode: model.AppErrCodeForbidden},
		{name: "異常系: 開けないトピックのページは見つからない", userID: f.nonMemberID, pageNumber: pagePinPrivatePageNumber, wantErr: true, wantCode: model.AppErrCodeResourceNotFound},
		{name: "異常系: 存在しないページは見つからない", userID: f.editorID, pageNumber: 999, wantErr: true, wantCode: model.AppErrCodeResourceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), PinPageInput{
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
					if page := findPagePinTestPage(t, pageRepo, f.spaceID, model.PageNumber(tt.pageNumber)); page.PinnedAt != nil {
						t.Errorf("page.PinnedAt = %v、期待値 = nil (拒否したので更新されない)", page.PinnedAt)
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
			if page := findPagePinTestPage(t, pageRepo, f.spaceID, model.PageNumber(tt.pageNumber)); page.PinnedAt == nil {
				t.Error("page.PinnedAt = nil、期待値 = ピンを打った日時")
			}
		})
	}
}
