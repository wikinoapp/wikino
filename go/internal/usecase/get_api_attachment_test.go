package usecase

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// apiAttachmentFixtureは添付ファイルの取得のテストのフィクスチャ
type apiAttachmentFixture struct {
	apiTopicFixture
	// attachmentIDsは添付ファイルの名前ごとのID
	attachmentIDs map[string]model.AttachmentID
}

// setupAPIAttachmentFixtureは、参照元の異なる添付ファイルを作る。
// public: 公開トピックのページだけが参照する。private: 参加していない非公開トピックのページだけが参照する。
// unreferenced: どのページも参照しない
func setupAPIAttachmentFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiAttachmentFixture {
	t.Helper()

	f := apiAttachmentFixture{
		apiTopicFixture: setupAPITopicFixture(t, tx, identifier, memberRole),
		attachmentIDs:   map[string]model.AttachmentID{},
	}
	parRepo := repository.NewPageAttachmentReferenceRepository(testutil.QueriesWithTx(tx))

	attachments := []struct {
		name string
		// topicNumberは参照元のページのトピック番号。0なら参照元のページを作らない
		topicNumber int32
		pageNumber  model.PageNumber
	}{
		{name: "public", topicNumber: 1, pageNumber: 1},
		{name: "private", topicNumber: 3, pageNumber: 2},
		{name: "unreferenced"},
	}
	for _, a := range attachments {
		id := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithSpaceMemberID(f.spaceMemberID).
			WithFilename(a.name + ".png").
			WithContentType("image/png").
			WithByteSize(2048).
			WithBlobKey(identifier + "-" + a.name).
			Build()
		f.attachmentIDs[a.name] = id
		if a.topicNumber == 0 {
			continue
		}
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[a.topicNumber]).WithNumber(a.pageNumber).WithTitle(a.name).Build()
		if _, err := parRepo.CreateBatch(t.Context(), pageID, f.space.ID, []model.AttachmentID{id}); err != nil {
			t.Fatalf("添付ファイル%sの参照の作成に失敗: %v", a.name, err)
		}
	}
	return f
}

func newGetAPIAttachmentUC(tx *sql.Tx, objectStorage storage.ObjectStorage) *GetAPIAttachmentUsecase {
	return NewGetAPIAttachmentUsecase(repository.NewAttachmentRepository(testutil.QueriesWithTx(tx)), objectStorage)
}

func TestGetAPIAttachmentUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		// attachmentは指定する添付ファイルの名前。otherは別のスペースの添付ファイル、空ならどこにも無いID
		attachment string
		wantFound  bool
	}{
		{
			name:        "公開トピックのページが参照する添付ファイルを返す",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
			attachment:  "public",
			wantFound:   true,
		},
		{
			name:        "トークンがtopic:readを持たなくても、非公開トピックのページだけが参照する添付ファイルを返す",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
			attachment:  "private",
			wantFound:   true,
		},
		{
			name:        "どのページも参照しない添付ファイルを返す",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
			attachment:  "unreferenced",
			wantFound:   true,
		},
		{
			name:        "トークンがattachment:readを持たなければ、公開トピックのページだけが参照する添付ファイルも未存在",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			attachment:  "public",
		},
		{
			name:       "スコープの無いトークンでは公開添付ファイルも未存在",
			attachment: "public",
		},
		{
			name:        "トークンがattachment:readを持たなければ、非公開トピックのページが参照する添付ファイルは未存在",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			attachment:  "private",
		},
		{
			name:        "トークンがattachment:readを持たなければ、どのページも参照しない添付ファイルは未存在",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			attachment:  "unreferenced",
		},
		{
			name:        "別のスペースの添付ファイルは未存在",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
			attachment:  "other",
		},
		{
			name:        "存在しない添付ファイルは未存在",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
		},
		{
			name:        "トークンが属さないスペースは未存在",
			tokenScopes: []model.Scope{model.ScopeAttachmentRead},
			identifier:  "api-attachment-get-other",
			attachment:  "public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIAttachmentFixture(t, tx, "api-attachment-get", apiTopicRegularMemberRole)
			other := setupAPIAttachmentFixture(t, tx, "api-attachment-get-other", apiTopicRegularMemberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}
			attachmentID := model.AttachmentID("00000000-0000-0000-0000-000000000000")
			switch tt.attachment {
			case "":
			case "other":
				attachmentID = other.attachmentIDs["public"]
			default:
				attachmentID = f.attachmentIDs[tt.attachment]
			}

			before := time.Now().UTC().Truncate(time.Second)
			output, err := newGetAPIAttachmentUC(tx, storage.NewFakeObjectStorage()).Execute(t.Context(), GetAPIAttachmentInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				AttachmentID:    attachmentID,
			})

			if !tt.wantFound {
				if output != nil {
					t.Errorf("output = %+v、期待値 = nil", output)
				}
				if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
					t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			want := model.Attachment{
				ID:          attachmentID,
				SpaceID:     f.space.ID,
				Filename:    tt.attachment + ".png",
				BlobKey:     "api-attachment-get-" + tt.attachment,
				ContentType: "image/png",
				ByteSize:    2048,
			}
			if *output.Attachment != want {
				t.Errorf("Attachment = %+v、期待値 = %+v", *output.Attachment, want)
			}
			// フェイクのストレージはキーと有効期間 (秒) からURLを組み立てる
			if wantURL := "https://storage.example.com/api-attachment-get-" + tt.attachment + "?expires_in=300"; output.DownloadURL != wantURL {
				t.Errorf("DownloadURL = %q、期待値 = %q", output.DownloadURL, wantURL)
			}
			after := time.Now().UTC()
			if got := output.DownloadURLExpiresAt; got.Before(before.Add(model.AttachmentDownloadExpiration)) || got.After(after.Add(model.AttachmentDownloadExpiration)) || got.Nanosecond() != 0 {
				t.Errorf("DownloadURLExpiresAt = %v、期待値 = 秒未満の無い、取得から5分後の時刻", got)
			}
		})
	}
}

func TestGetAPIAttachmentUsecase_Execute_署名付きURLを作れない(t *testing.T) {
	t.Parallel()

	t.Run("オブジェクトストレージが設定されていなければエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupAPIAttachmentFixture(t, tx, "api-attachment-get-nostorage", apiTopicRegularMemberRole)

		output, err := newGetAPIAttachmentUC(tx, nil).Execute(t.Context(), GetAPIAttachmentInput{
			Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopeAttachmentRead}),
			SpaceIdentifier: f.space.Identifier,
			AttachmentID:    f.attachmentIDs["public"],
		})
		if output != nil || err == nil || model.AsAppError(err) != nil {
			t.Errorf("output = %+v、error = %v、期待値 = AppErrorでないエラー", output, err)
		}
	})

	t.Run("署名に失敗すればエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupAPIAttachmentFixture(t, tx, "api-attachment-get-presignerr", apiTopicRegularMemberRole)
		objectStorage := storage.NewFakeObjectStorage()
		objectStorage.PresignErr = errors.New("presign failed")

		output, err := newGetAPIAttachmentUC(tx, objectStorage).Execute(t.Context(), GetAPIAttachmentInput{
			Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopeAttachmentRead}),
			SpaceIdentifier: f.space.Identifier,
			AttachmentID:    f.attachmentIDs["public"],
		})
		if output != nil || !errors.Is(err, objectStorage.PresignErr) {
			t.Errorf("output = %+v、error = %v、期待値 = 署名のエラー", output, err)
		}
	})
}
