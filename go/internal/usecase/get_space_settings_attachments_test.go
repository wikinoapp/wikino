package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// spaceSettingsAttachmentsFixtureは、添付ファイルの管理の画面の取得のテストが使うスペースとメンバー
type spaceSettingsAttachmentsFixture struct {
	tx         *sql.Tx
	queries    *query.Queries
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	userID     model.UserID
	memberID   model.SpaceMemberID
}

func setupSpaceSettingsAttachmentsFixture(t *testing.T, identifier string, role model.SpaceRole) spaceSettingsAttachmentsFixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).Build()
	memberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(role).Build()

	return spaceSettingsAttachmentsFixture{
		tx:         tx,
		queries:    testutil.QueriesWithTx(tx),
		identifier: model.SpaceIdentifier(identifier),
		spaceID:    spaceID,
		userID:     userID,
		memberID:   memberID,
	}
}

func (f spaceSettingsAttachmentsFixture) usecase(objectStorage storage.ObjectStorage) *GetSpaceSettingsAttachmentsUsecase {
	return NewGetSpaceSettingsAttachmentsUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		repository.NewAttachmentRepository(f.queries),
		objectStorage,
	)
}

func (f spaceSettingsAttachmentsFixture) add(t *testing.T, filename, contentType string, attachedAt time.Time) model.AttachmentID {
	t.Helper()
	return testutil.NewAttachmentBuilder(t, f.tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.memberID).
		WithFilename(filename).
		WithContentType(contentType).
		WithBlobKey("ssa-" + string(f.identifier) + "-" + filename).
		WithAttachedAt(attachedAt).
		Build()
}

func (f spaceSettingsAttachmentsFixture) input() GetSpaceSettingsAttachmentsInput {
	return GetSpaceSettingsAttachmentsInput{SpaceIdentifier: f.identifier, UserID: f.userID, Page: 1, PageLimit: 50}
}

func TestGetSpaceSettingsAttachmentsUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	base := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	t.Run("管理者には新しい順の添付ファイルと、画像だけのサムネイルを返す", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-admin", model.SpaceRoleAdmin)
		png := f.add(t, "photo.png", "image/png", base)
		svg := f.add(t, "icon.svg", "image/svg+xml", base.Add(time.Minute))
		pdf := f.add(t, "doc.pdf", "application/pdf", base.Add(2*time.Minute))

		output, err := f.usecase(storage.NewFakeObjectStorage()).Execute(ctx, f.input())
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		var gotIDs []model.AttachmentID
		for _, a := range output.Attachments {
			gotIDs = append(gotIDs, a.ID)
		}
		if want := []model.AttachmentID{pdf, svg, png}; len(gotIDs) != 3 || gotIDs[0] != want[0] || gotIDs[1] != want[1] || gotIDs[2] != want[2] {
			t.Errorf("添付ファイル = %v、期待値 = %v", gotIDs, want)
		}
		if output.TotalCount != 3 {
			t.Errorf("TotalCount = %d、期待値 = 3", output.TotalCount)
		}
		if !output.CanDeleteAttachment {
			t.Error("CanDeleteAttachment = false、管理者はtrueを期待")
		}
		if _, ok := output.ThumbnailURLs[pdf]; ok {
			t.Error("画像でない添付ファイルにサムネイルがある")
		}
		if len(output.ThumbnailURLs) != 2 {
			t.Fatalf("サムネイルの数 = %d、期待値 = 2", len(output.ThumbnailURLs))
		}

		// SVGも `<img>` で表示できるよう、記録された種類のまま返させる
		u, err := url.Parse(output.ThumbnailURLs[svg])
		if err != nil {
			t.Fatalf("サムネイルのURLを解釈できない: %v", err)
		}
		if got := u.Query().Get("response-content-type"); got != "image/svg+xml" {
			t.Errorf("response-content-type = %q、期待値 = %q", got, "image/svg+xml")
		}
		if got := u.Query().Get("expires_in"); got != "3600" {
			t.Errorf("expires_in = %q、期待値 = %q", got, "3600")
		}
	})

	t.Run("ファイル名で絞り込む", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-query", model.SpaceRoleAdmin)
		match := f.add(t, "Meeting.png", "image/png", base)
		f.add(t, "other.png", "image/png", base)

		input := f.input()
		input.Query = "meeting"
		output, err := f.usecase(storage.NewFakeObjectStorage()).Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(output.Attachments) != 1 || output.Attachments[0].ID != match || output.TotalCount != 1 {
			t.Errorf("添付ファイル = %v (総数 %d)、%vだけを期待", output.Attachments, output.TotalCount, match)
		}
	})

	t.Run("オブジェクトストレージが無くても、サムネイルを出さずに一覧を返す", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-nostorage", model.SpaceRoleAdmin)
		f.add(t, "photo.png", "image/png", base)

		output, err := f.usecase(nil).Execute(ctx, f.input())
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(output.Attachments) != 1 {
			t.Errorf("件数 = %d、期待値 = 1", len(output.Attachments))
		}
		if len(output.ThumbnailURLs) != 0 {
			t.Errorf("サムネイルの数 = %d、期待値 = 0", len(output.ThumbnailURLs))
		}
	})

	t.Run("署名付きURLを作れなければエラーを返す", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-presignerr", model.SpaceRoleAdmin)
		f.add(t, "photo.png", "image/png", base)
		objectStorage := storage.NewFakeObjectStorage()
		objectStorage.PresignErr = errors.New("署名の失敗")

		if _, err := f.usecase(objectStorage).Execute(ctx, f.input()); err == nil {
			t.Error("エラーを期待したが、nilだった")
		}
	})

	t.Run("スペースを変更できないメンバーは拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-editor", model.SpaceRoleEditor)
		_, err := f.usecase(storage.NewFakeObjectStorage()).Execute(ctx, f.input())
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Errorf("エラー = %v、ForbiddenのAppErrorを期待", err)
		}
	})

	t.Run("メンバーでない閲覧者と存在しないスペースは見つからないとする", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsAttachmentsFixture(t, "get-ssa-outsider", model.SpaceRoleAdmin)
		outsiderID := testutil.NewUserBuilder(t, f.tx).WithEmail("get-ssa-outsider2@example.com").WithAtname("get_ssa_outsider2").Build()
		uc := f.usecase(storage.NewFakeObjectStorage())

		input := f.input()
		input.UserID = outsiderID
		if _, err := uc.Execute(ctx, input); model.AsAppError(err) == nil || model.AsAppError(err).Code != model.AppErrCodeResourceNotFound {
			t.Errorf("メンバーでない閲覧者のエラー = %v、ResourceNotFoundのAppErrorを期待", err)
		}

		input = f.input()
		input.SpaceIdentifier = "get-ssa-missing"
		if _, err := uc.Execute(ctx, input); model.AsAppError(err) == nil || model.AsAppError(err).Code != model.AppErrCodeResourceNotFound {
			t.Errorf("存在しないスペースのエラー = %v、ResourceNotFoundのAppErrorを期待", err)
		}
	})
}

func TestGetSpaceSettingsAttachmentsUsecase_Execute_QueryCount(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	base := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	countQueries := func(t *testing.T, attachments int) int {
		t.Helper()

		f := setupSpaceSettingsAttachmentsFixture(t, fmt.Sprintf("get-ssa-count-%d", attachments), model.SpaceRoleAdmin)
		for i := range attachments {
			f.add(t, fmt.Sprintf("photo-%d.png", i), "image/png", base.Add(time.Duration(i)*time.Minute))
		}

		counter := testutil.NewQueryCounter(f.tx)
		uc := NewGetSpaceSettingsAttachmentsUsecase(
			repository.NewSpaceRepository(counter.Queries()),
			repository.NewSpaceMemberRepository(counter.Queries()),
			repository.NewAttachmentRepository(counter.Queries()),
			storage.NewFakeObjectStorage(),
		)
		output, err := uc.Execute(ctx, f.input())
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(output.Attachments) != attachments {
			t.Fatalf("件数 = %d、期待値 = %d", len(output.Attachments), attachments)
		}
		return counter.CountContaining("-- name:")
	}

	one := countQueries(t, 1)
	many := countQueries(t, 5)
	if one != many {
		t.Errorf("クエリの数 (1件) = %d、(5件) = %d、件数によらず同じであることを期待", one, many)
	}
}
