package usecase

import (
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAttachmentSignedURLsUC(q *query.Queries, objectStorage storage.ObjectStorage) *GetAttachmentSignedURLsUsecase {
	return NewGetAttachmentSignedURLsUsecase(repository.NewAttachmentRepository(q), repository.NewSpaceMemberRepository(q), objectStorage)
}

// signedURLKeysは、署名付きURLのパスから、添付ファイルのIDをキーにしたストレージキーを取り出す
func signedURLKeys(t *testing.T, signedURLs map[model.AttachmentID]string) map[model.AttachmentID]string {
	t.Helper()

	keys := make(map[model.AttachmentID]string, len(signedURLs))
	for id, raw := range signedURLs {
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("署名付きURLのパースに失敗: %v", err)
		}
		keys[id] = parsed.Path[1:]
	}
	return keys
}

func TestGetAttachmentSignedURLsUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "signed-urls")
	other := setupAttachmentFixture(t, tx, "signed-urls-other")
	uc := newGetAttachmentSignedURLsUC(testutil.QueriesWithTx(tx), storage.NewFakeObjectStorage())

	all := []model.AttachmentID{
		f.attachmentIDs["public"], f.attachmentIDs["private"], f.attachmentIDs["unreferenced"], f.attachmentIDs["trashed"],
		other.attachmentIDs["public"], other.attachmentIDs["private"],
		"00000000-0000-0000-0000-000000000000", "not-a-uuid",
	}

	tests := []struct {
		name   string
		userID *model.UserID
		ids    []model.AttachmentID
		want   map[model.AttachmentID]string
	}{
		{
			name:   "メンバーには自分のスペースの添付ファイルと、ほかのスペースの公開の添付ファイルだけを返す",
			userID: &f.userID,
			ids:    all,
			want: map[model.AttachmentID]string{
				f.attachmentIDs["public"]:       "signed-urls-public",
				f.attachmentIDs["private"]:      "signed-urls-private",
				f.attachmentIDs["unreferenced"]: "signed-urls-unreferenced",
				f.attachmentIDs["trashed"]:      "signed-urls-trashed",
				other.attachmentIDs["public"]:   "signed-urls-other-public",
			},
		},
		{
			name:   "複数のスペースのIDが混ざっても、スペースごとのメンバーシップで判定する",
			userID: &other.userID,
			ids:    all,
			want: map[model.AttachmentID]string{
				f.attachmentIDs["public"]:      "signed-urls-public",
				other.attachmentIDs["public"]:  "signed-urls-other-public",
				other.attachmentIDs["private"]: "signed-urls-other-private",
			},
		},
		{
			name: "ログインしていなければ公開の添付ファイルだけを返す",
			ids:  all,
			want: map[model.AttachmentID]string{
				f.attachmentIDs["public"]:     "signed-urls-public",
				other.attachmentIDs["public"]: "signed-urls-other-public",
			},
		},
		{
			name:   "参加していないユーザーには公開の添付ファイルだけを返す",
			userID: &f.outsiderUserID,
			ids:    all,
			want: map[model.AttachmentID]string{
				f.attachmentIDs["public"]:     "signed-urls-public",
				other.attachmentIDs["public"]: "signed-urls-other-public",
			},
		},
		{
			name:   "IDが空なら空のmapを返す",
			userID: &f.userID,
			want:   map[model.AttachmentID]string{},
		},
		{
			name:   "存在しないIDと不正な形のIDだけなら空のmapを返す",
			userID: &f.userID,
			ids:    []model.AttachmentID{"00000000-0000-0000-0000-000000000000", "not-a-uuid"},
			want:   map[model.AttachmentID]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(t.Context(), GetAttachmentSignedURLsInput{AttachmentIDs: tt.ids, UserID: tt.userID})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			got := signedURLKeys(t, output.SignedURLs)
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("署名した添付ファイル = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestGetAttachmentSignedURLsUsecase_Execute_ResponseHeaders(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "signed-urls-headers")
	svgID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(f.space.ID).
		WithSpaceMemberID(f.spaceMemberID).
		WithFilename("diagram.svg").
		WithContentType("image/svg+xml").
		WithByteSize(512).
		WithBlobKey("signed-urls-headers-svg").
		Build()
	uc := newGetAttachmentSignedURLsUC(testutil.QueriesWithTx(tx), storage.NewFakeObjectStorage())

	output, err := uc.Execute(t.Context(), GetAttachmentSignedURLsInput{
		AttachmentIDs: []model.AttachmentID{f.attachmentIDs["public"], svgID},
		UserID:        &f.userID,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	tests := []struct {
		name            string
		id              model.AttachmentID
		wantDisposition string
		wantContentType string
	}{
		{
			name:            "画像はインラインで、記録された種類で返させる",
			id:              f.attachmentIDs["public"],
			wantDisposition: `inline; filename="public.png"; filename*=UTF-8''public.png`,
			wantContentType: "image/png",
		},
		{
			name:            "SVGは<img>で表示できるよう、記録された種類で返させる",
			id:              svgID,
			wantDisposition: `attachment; filename="diagram.svg"; filename*=UTF-8''diagram.svg`,
			wantContentType: "image/svg+xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(output.SignedURLs[tt.id])
			if err != nil {
				t.Fatalf("署名付きURLのパースに失敗: %v", err)
			}
			query := parsed.Query()
			if got := query.Get("response-content-disposition"); got != tt.wantDisposition {
				t.Errorf("response-content-disposition = %q、期待値 = %q", got, tt.wantDisposition)
			}
			if got := query.Get("response-content-type"); got != tt.wantContentType {
				t.Errorf("response-content-type = %q、期待値 = %q", got, tt.wantContentType)
			}
			// 有効期間は1時間 (model.AttachmentSignedURLExpiration)
			if got := query.Get("expires_in"); got != "3600" {
				t.Errorf("expires_in = %q、期待値 = %q", got, "3600")
			}
		})
	}
}

func TestGetAttachmentSignedURLsUsecase_Execute_QueryCount(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "signed-urls-count")
	other := setupAttachmentFixture(t, tx, "signed-urls-count-other")

	countQueries := func(t *testing.T, tx *sql.Tx, ids []model.AttachmentID) int {
		t.Helper()
		counter := testutil.NewQueryCounter(tx)
		uc := newGetAttachmentSignedURLsUC(counter.Queries(), storage.NewFakeObjectStorage())
		if _, err := uc.Execute(t.Context(), GetAttachmentSignedURLsInput{AttachmentIDs: ids, UserID: &f.userID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		return counter.CountContaining("-- name:")
	}

	one := countQueries(t, tx, []model.AttachmentID{f.attachmentIDs["public"]})
	many := countQueries(t, tx, []model.AttachmentID{
		f.attachmentIDs["public"], f.attachmentIDs["private"], f.attachmentIDs["unreferenced"], f.attachmentIDs["trashed"],
		other.attachmentIDs["public"], other.attachmentIDs["private"], other.attachmentIDs["unreferenced"],
	})
	if one != 3 {
		t.Errorf("IDが1件のときのクエリの数 = %d、期待値 = 3", one)
	}
	if many != one {
		t.Errorf("IDが7件 (2つのスペース) のときのクエリの数 = %d、期待値 = %d (1件のときと同じ)", many, one)
	}
}

func TestGetAttachmentSignedURLsUsecase_Execute_ObjectStorageNotConfigured(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAttachmentFixture(t, tx, "signed-urls-nostorage")

	_, err := newGetAttachmentSignedURLsUC(testutil.QueriesWithTx(tx), nil).Execute(t.Context(), GetAttachmentSignedURLsInput{
		AttachmentIDs: []model.AttachmentID{f.attachmentIDs["public"]},
	})
	if err == nil || model.AsAppError(err) != nil {
		t.Errorf("Execute()のエラー = %v、期待値 = AppErrorでないエラー", err)
	}
}
