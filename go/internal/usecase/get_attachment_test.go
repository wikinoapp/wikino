package usecase

import (
	"database/sql"
	"errors"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// attachmentFixtureは、Web画面の添付ファイルの表示のテストのフィクスチャ
type attachmentFixture struct {
	apiAttachmentFixture
	// outsiderUserIDは、スペースに参加していないユーザーのID
	outsiderUserID model.UserID
}

// setupAttachmentFixtureは、公開APIの添付ファイルのフィクスチャに、公開トピックのゴミ箱のページだけが
// 参照する添付ファイル (trashed) と、スペースに参加していないユーザーを足す
func setupAttachmentFixture(t *testing.T, tx *sql.Tx, identifier string) attachmentFixture {
	t.Helper()

	f := attachmentFixture{apiAttachmentFixture: setupAPIAttachmentFixture(t, tx, identifier, model.SpaceRoleViewer)}

	trashedID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(f.space.ID).
		WithSpaceMemberID(f.spaceMemberID).
		WithFilename("trashed.png").
		WithContentType("image/png").
		WithByteSize(2048).
		WithBlobKey(identifier + "-trashed").
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(10).WithTitle("trashed").WithTrashed().Build()
	parRepo := repository.NewPageAttachmentReferenceRepository(testutil.QueriesWithTx(tx))
	if _, err := parRepo.CreateBatch(t.Context(), pageID, f.space.ID, []model.AttachmentID{trashedID}); err != nil {
		t.Fatalf("添付ファイルtrashedの参照の作成に失敗: %v", err)
	}
	f.attachmentIDs["trashed"] = trashedID

	outsider := identifier + "_outsider"
	f.outsiderUserID = testutil.NewUserBuilder(t, tx).WithAtname(outsider).WithEmail(outsider + "@example.com").Build()
	return f
}

func newGetAttachmentUC(tx *sql.Tx, objectStorage storage.ObjectStorage) *GetAttachmentUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewGetAttachmentUsecase(repository.NewAttachmentRepository(q), repository.NewSpaceMemberRepository(q), objectStorage)
}

func TestGetAttachmentUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "get-attachment")
	other := setupAttachmentFixture(t, tx, "get-attachment-other")
	uc := newGetAttachmentUC(tx, storage.NewFakeObjectStorage())

	tests := []struct {
		name string
		// viewerは閲覧者。member: スペースのメンバー、outsider: スペースに参加していないユーザー、空: ログインしていない
		viewer       string
		attachmentID model.AttachmentID
		wantFound    bool
		wantKey      string
	}{
		{name: "メンバーは公開トピックのページが参照する添付ファイルを見られる", viewer: "member", attachmentID: f.attachmentIDs["public"], wantFound: true, wantKey: "get-attachment-public"},
		{name: "メンバーは非公開トピックのページが参照する添付ファイルを見られる", viewer: "member", attachmentID: f.attachmentIDs["private"], wantFound: true, wantKey: "get-attachment-private"},
		{name: "メンバーはどのページも参照しない添付ファイルを見られる", viewer: "member", attachmentID: f.attachmentIDs["unreferenced"], wantFound: true, wantKey: "get-attachment-unreferenced"},
		{name: "メンバーはゴミ箱のページが参照する添付ファイルを見られる", viewer: "member", attachmentID: f.attachmentIDs["trashed"], wantFound: true, wantKey: "get-attachment-trashed"},
		{name: "ログインしていなくても公開トピックのページだけが参照する添付ファイルを見られる", attachmentID: f.attachmentIDs["public"], wantFound: true, wantKey: "get-attachment-public"},
		{name: "ログインしていなければ非公開トピックのページが参照する添付ファイルは見られない", attachmentID: f.attachmentIDs["private"]},
		{name: "ログインしていなければゴミ箱のページだけが参照する添付ファイルは見られない", attachmentID: f.attachmentIDs["trashed"]},
		{name: "ログインしていなければどのページも参照しない添付ファイルは見られない", attachmentID: f.attachmentIDs["unreferenced"]},
		{name: "参加していないユーザーは公開トピックのページだけが参照する添付ファイルを見られる", viewer: "outsider", attachmentID: f.attachmentIDs["public"], wantFound: true, wantKey: "get-attachment-public"},
		{name: "参加していないユーザーは非公開トピックのページが参照する添付ファイルを見られない", viewer: "outsider", attachmentID: f.attachmentIDs["private"]},
		{name: "ほかのスペースのメンバーは、そのスペースの非公開の添付ファイルを見られない", viewer: "member", attachmentID: other.attachmentIDs["private"]},
		{name: "存在しないIDは見つからない", viewer: "member", attachmentID: model.AttachmentID("00000000-0000-0000-0000-000000000000")},
		{name: "不正な形のIDは見つからない", viewer: "member", attachmentID: model.AttachmentID("not-a-uuid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := GetAttachmentInput{AttachmentID: tt.attachmentID}
			switch tt.viewer {
			case "member":
				input.UserID = &f.userID
			case "outsider":
				input.UserID = &f.outsiderUserID
			}

			output, err := uc.Execute(t.Context(), input)
			if !tt.wantFound {
				ae := model.AsAppError(err)
				if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
					t.Fatalf("Execute()のエラー = %v、期待値 = 未存在のAppError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			parsed, err := url.Parse(output.DownloadURL)
			if err != nil {
				t.Fatalf("DownloadURLのパースに失敗: %v", err)
			}
			if parsed.Path != "/"+tt.wantKey {
				t.Errorf("DownloadURLのパス = %q、期待値 = %q", parsed.Path, "/"+tt.wantKey)
			}
		})
	}
}

func TestGetAttachmentUsecase_Execute_ResponseHeaders(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "get-attachment-headers")
	svgID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(f.space.ID).
		WithSpaceMemberID(f.spaceMemberID).
		WithFilename("diagram.svg").
		WithContentType("image/svg+xml").
		WithByteSize(512).
		WithBlobKey("get-attachment-headers-svg").
		Build()
	uc := newGetAttachmentUC(tx, storage.NewFakeObjectStorage())

	tests := []struct {
		name            string
		attachmentID    model.AttachmentID
		wantDisposition string
		wantContentType string
	}{
		{
			name:            "画像はインラインで、記録された種類で返させる",
			attachmentID:    f.attachmentIDs["public"],
			wantDisposition: `inline; filename="public.png"; filename*=UTF-8''public.png`,
			wantContentType: "image/png",
		},
		{
			name:            "SVGはダウンロードさせ、application/octet-streamで返させる",
			attachmentID:    svgID,
			wantDisposition: `attachment; filename="diagram.svg"; filename*=UTF-8''diagram.svg`,
			wantContentType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(t.Context(), GetAttachmentInput{AttachmentID: tt.attachmentID, UserID: &f.userID})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			parsed, err := url.Parse(output.DownloadURL)
			if err != nil {
				t.Fatalf("DownloadURLのパースに失敗: %v", err)
			}
			query := parsed.Query()
			if got := query.Get("response-content-disposition"); got != tt.wantDisposition {
				t.Errorf("response-content-disposition = %q、期待値 = %q", got, tt.wantDisposition)
			}
			if got := query.Get("response-content-type"); got != tt.wantContentType {
				t.Errorf("response-content-type = %q、期待値 = %q", got, tt.wantContentType)
			}
			// 有効期間はRails版のリダイレクト先と同じ5分
			if got := query.Get("expires_in"); got != "300" {
				t.Errorf("expires_in = %q、期待値 = %q", got, "300")
			}
		})
	}
}

func TestGetAttachmentUsecase_Execute_StorageErrors(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "get-attachment-storage")

	t.Run("オブジェクトストレージが設定されていなければエラーにする", func(t *testing.T) {
		_, err := newGetAttachmentUC(tx, nil).Execute(t.Context(), GetAttachmentInput{AttachmentID: f.attachmentIDs["public"]})
		if err == nil || model.AsAppError(err) != nil {
			t.Errorf("Execute()のエラー = %v、期待値 = AppErrorでないエラー", err)
		}
	})

	t.Run("署名に失敗したらエラーにする", func(t *testing.T) {
		wantErr := errors.New("署名に失敗")
		fake := storage.NewFakeObjectStorage()
		fake.PresignErr = wantErr
		_, err := newGetAttachmentUC(tx, fake).Execute(t.Context(), GetAttachmentInput{AttachmentID: f.attachmentIDs["public"]})
		if !errors.Is(err, wantErr) {
			t.Errorf("Execute()のエラー = %v、期待値 = %v", err, wantErr)
		}
	})
}
