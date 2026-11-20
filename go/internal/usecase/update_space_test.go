package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// spaceSettingsGeneralFixtureはスペースの一般設定のユースケースのテストが使うスペースとメンバー。
type spaceSettingsGeneralFixture struct {
	queries    *query.Queries
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	userID     model.UserID
}

// setupSpaceSettingsGeneralFixtureは、渡したロールのメンバーを持つスペースをトランザクションに
// 用意する。一般設定のユースケースは自身でトランザクションを開かないため、トランザクションに
// 閉じ込めてよい。
func setupSpaceSettingsGeneralFixture(t *testing.T, identifier string, role model.SpaceRole) spaceSettingsGeneralFixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).WithName("開発チーム").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(role).Build()

	return spaceSettingsGeneralFixture{
		queries:    testutil.QueriesWithTx(tx),
		identifier: model.SpaceIdentifier(identifier),
		spaceID:    spaceID,
		userID:     userID,
	}
}

// newUpdateSpaceUsecaseはクエリを共有するリポジトリでUseCaseを組み立てる。
func newUpdateSpaceUsecase(queries *query.Queries) *UpdateSpaceUsecase {
	spaceRepo := repository.NewSpaceRepository(queries)
	return NewUpdateSpaceUsecase(spaceRepo, repository.NewSpaceMemberRepository(queries), validator.NewSpaceUpdateValidator(spaceRepo))
}

func TestUpdateSpaceUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupSpaceSettingsGeneralFixture(t, "upd-space-success", model.SpaceRoleAdmin)

	output, err := newUpdateSpaceUsecase(f.queries).Execute(ctx, UpdateSpaceInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		Identifier:      "upd-space-renamed",
		Name:            "企画チーム",
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if output.Space.ID != f.spaceID || output.Space.Identifier != "upd-space-renamed" || output.Space.Name != "企画チーム" {
		t.Errorf("output.Space = %+v、更新したスペースを期待", output.Space)
	}

	stored, err := repository.NewSpaceRepository(f.queries).FindByID(ctx, f.spaceID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if stored.Identifier != "upd-space-renamed" || stored.Name != "企画チーム" {
		t.Errorf("保存されたスペース = (%q, %q)、期待値 = (%q, %q)", stored.Identifier, stored.Name, "upd-space-renamed", "企画チーム")
	}
}

func TestUpdateSpaceUsecase_Execute_拒否する(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name       string
		identifier string
		role       model.SpaceRole
		outsider   bool
		wantCode   model.AppErrorCode
	}{
		{name: "編集者は変更できない", identifier: "upd-space-editor", role: model.SpaceRoleEditor, wantCode: model.AppErrCodeForbidden},
		{name: "閲覧者は変更できない", identifier: "upd-space-viewer", role: model.SpaceRoleViewer, wantCode: model.AppErrCodeForbidden},
		{name: "メンバーでないユーザーには見つからない", identifier: "upd-space-outsider", role: model.SpaceRoleAdmin, outsider: true, wantCode: model.AppErrCodeResourceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupSpaceSettingsGeneralFixture(t, tt.identifier, tt.role)
			userID := f.userID
			if tt.outsider {
				userID = model.UserID("00000000-0000-0000-0000-000000000000")
			}

			output, err := newUpdateSpaceUsecase(f.queries).Execute(ctx, UpdateSpaceInput{
				SpaceIdentifier: f.identifier,
				UserID:          userID,
				Identifier:      tt.identifier + "-x",
				Name:            "乗っ取り",
			})
			if output != nil {
				t.Errorf("output = %v、期待値 = nil", output)
			}
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != tt.wantCode {
				t.Fatalf("エラー = %v、コード%vのAppErrorを期待", err, tt.wantCode)
			}

			stored, err := repository.NewSpaceRepository(f.queries).FindByID(ctx, f.spaceID)
			if err != nil {
				t.Fatalf("スペースの取得に失敗: %v", err)
			}
			if stored.Identifier != f.identifier || stored.Name != "開発チーム" {
				t.Errorf("保存されたスペース = (%q, %q)、変わっていないことを期待", stored.Identifier, stored.Name)
			}
		})
	}
}

func TestUpdateSpaceUsecase_Execute_検証エラー(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupSpaceSettingsGeneralFixture(t, "upd-space-invalid", model.SpaceRoleAdmin)

	output, err := newUpdateSpaceUsecase(f.queries).Execute(ctx, UpdateSpaceInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		Identifier:      "upd space",
		Name:            "",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}
	if !ve.HasFieldError("identifier") || !ve.HasFieldError("name") {
		t.Errorf("識別子と名前のフィールドエラーを期待したが、%vだった", ve.Fields)
	}
}

// TestUpdateSpaceUsecase_Execute_IdentifierConflictは、一意性検証では見えなかったスペースが
// UPDATE時には存在する状況を再現する。検証用のスナップショットを固定することで、実行順の
// 偶然に頼らず、一意制約違反の変換を実DBで確認する。
func TestUpdateSpaceUsecase_Execute_IdentifierConflict(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("upd-space-conflict@example.com").
		WithAtname("upd_space_conflict").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("upd-space-conflict").Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleAdmin).Build()

	snapshot, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		t.Fatalf("検証用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = snapshot.Rollback() }()
	validationRepo := spaceRepo.WithTx(snapshot)
	const identifier = "upd-space-race"
	exists, err := validationRepo.ExistsOtherByIdentifier(ctx, spaceID, identifier)
	if err != nil || exists {
		t.Fatalf("競合前の存在確認: exists = %v、エラー = %v", exists, err)
	}

	// スナップショット確定後に別の接続で競合するスペースを作る。Validatorからは見えないが、
	// UPDATEは一意インデックスによりこの行との競合を検出する。
	testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	uc := NewUpdateSpaceUsecase(spaceRepo, repository.NewSpaceMemberRepository(queries), validator.NewSpaceUpdateValidator(validationRepo))
	output, err := uc.Execute(ctx, UpdateSpaceInput{
		SpaceIdentifier: "upd-space-conflict",
		UserID:          userID,
		Identifier:      identifier,
		Name:            "競合するスペース",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}
	if msgs := ve.GetFieldErrors("identifier"); !slices.Equal(msgs, []string{"この識別子は既に使用されています"}) {
		t.Errorf("識別子のエラー = %v、期待した重複エラーと一致しない", msgs)
	}

	stored, err := spaceRepo.FindByID(ctx, spaceID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if stored.Identifier != "upd-space-conflict" {
		t.Errorf("Identifier = %q、期待値 = %q", stored.Identifier, "upd-space-conflict")
	}
}
