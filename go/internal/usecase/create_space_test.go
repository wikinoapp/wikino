package usecase

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newCreateSpaceUsecaseは共有プールを使うUseCaseを組み立てる。UseCaseは自身でトランザクションを
// 開くため、テストのフィクスチャはトランザクションに閉じ込めずコミットする。
func newCreateSpaceUsecase() (*CreateSpaceUsecase, *repository.SpaceRepository, *repository.SpaceMemberRepository) {
	db := testutil.GetTestDB()
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	return NewCreateSpaceUsecase(db, spaceRepo, spaceMemberRepo, validator.NewSpaceCreateValidator(spaceRepo)), spaceRepo, spaceMemberRepo
}

func TestCreateSpaceUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	uc, spaceRepo, spaceMemberRepo := newCreateSpaceUsecase()

	userID := testutil.NewUserBuilderDB(t, testutil.GetTestDB()).
		WithEmail("create-space-success@example.com").
		WithAtname("create_space_success").
		Build()

	output, err := uc.Execute(ctx, CreateSpaceInput{
		UserID:     userID,
		Identifier: "create-space-success",
		Name:       "新しいスペース",
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	space, err := spaceRepo.FindByIdentifier(ctx, "create-space-success")
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if space == nil {
		t.Fatal("作成したスペースが見つからない")
	}
	if space.ID != output.Space.ID {
		t.Errorf("space.ID = %v、期待値 = %v", space.ID, output.Space.ID)
	}
	if space.Name != "新しいスペース" {
		t.Errorf("space.Name = %q、期待値 = %q", space.Name, "新しいスペース")
	}
	if space.Plan != model.PlanFree {
		t.Errorf("space.Plan = %v、期待値 = PlanFree", space.Plan)
	}

	member, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		t.Fatalf("スペースメンバーの取得に失敗: %v", err)
	}
	if member == nil {
		t.Fatal("作成者がスペースのアクティブなメンバーになっていない")
	}
	if member.Role != model.SpaceRoleAdmin {
		t.Errorf("member.Role = %q、期待値 = %q", member.Role, model.SpaceRoleAdmin)
	}

	// Rails版が作成者を管理者として判定できるよう、scopesにはspace:adminを書く
	var scopes []string
	if err := testutil.GetTestDB().QueryRowContext(ctx,
		`SELECT scopes FROM space_members WHERE id = $1 AND space_id = $2`,
		string(member.ID), string(space.ID),
	).Scan(pq.Array(&scopes)); err != nil {
		t.Fatalf("scopesの取得に失敗: %v", err)
	}
	if want := []string{"space:admin"}; !slices.Equal(scopes, want) {
		t.Errorf("scopes = %v、期待値 = %v", scopes, want)
	}
}

func TestCreateSpaceUsecase_Execute_ValidationError(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	uc, spaceRepo, _ := newCreateSpaceUsecase()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-space-invalid@example.com").
		WithAtname("create_space_invalid").
		Build()
	testutil.NewSpaceBuilderDB(t, db).WithIdentifier("create-space-taken").Build()

	tests := []struct {
		name       string
		identifier string
		spaceName  string
		field      string
	}{
		{name: "識別子の形式が不正", identifier: "create space", spaceName: "テスト", field: "identifier"},
		{name: "識別子が使われている", identifier: "create-space-taken", spaceName: "テスト", field: "identifier"},
		{name: "名前が空", identifier: "create-space-noname", spaceName: "", field: "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(ctx, CreateSpaceInput{
				UserID:     userID,
				Identifier: tt.identifier,
				Name:       tt.spaceName,
			})
			if output != nil {
				t.Errorf("output = %v、期待値 = nil", output)
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("%sのフィールドエラーが無い", tt.field)
			}
		})
	}

	// 検証で拒否した入力ではスペースを作らない
	spaces, err := spaceRepo.ListActiveByUser(ctx, userID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if len(spaces) != 0 {
		t.Errorf("len(spaces) = %d、期待値 = 0", len(spaces))
	}
}

// TestCreateSpaceUsecase_Execute_IdentifierConflictは、一意性検証では見えなかったスペースが
// INSERT時には存在する状況を再現する。検証用のスナップショットを固定することで、実行順の
// 偶然に頼らず、一意制約違反の変換を実DBで確認する。
func TestCreateSpaceUsecase_Execute_IdentifierConflict(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	_, spaceRepo, spaceMemberRepo := newCreateSpaceUsecase()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-space-conflict@example.com").
		WithAtname("create_space_conflict").
		Build()

	snapshot, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		t.Fatalf("検証用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = snapshot.Rollback() }()
	validationRepo := spaceRepo.WithTx(snapshot)
	const identifier = "create-space-race"
	exists, err := validationRepo.ExistsByIdentifier(ctx, identifier)
	if err != nil || exists {
		t.Fatalf("競合前の存在確認: exists = %v、エラー = %v", exists, err)
	}

	// スナップショット確定後に別の接続で競合するスペースを作る。Validatorからは見えないが、
	// 書き込みのトランザクションは一意インデックスによりこの行との競合を検出する。
	competingSpaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	uc := NewCreateSpaceUsecase(db, spaceRepo, spaceMemberRepo, validator.NewSpaceCreateValidator(validationRepo))
	output, err := uc.Execute(ctx, CreateSpaceInput{UserID: userID, Identifier: identifier, Name: "競合するスペース"})
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

	space, err := spaceRepo.FindByIdentifier(ctx, identifier)
	if err != nil || space == nil {
		t.Fatalf("競合相手のスペースの取得に失敗: %v", err)
	}
	if space.ID != competingSpaceID {
		t.Errorf("space.ID = %v、期待値 = %v", space.ID, competingSpaceID)
	}
	member, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		t.Fatalf("スペースメンバーの取得に失敗: %v", err)
	}
	if member != nil {
		t.Error("競合した作成で既存スペースにメンバーが追加された")
	}
}

func TestCreateSpaceUsecase_Execute_RollbackOnMemberFailure(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	uc, spaceRepo, _ := newCreateSpaceUsecase()
	const identifier = "create-space-failed"
	// 存在しないユーザーにより、スペースのINSERT後にメンバーの外部キー制約違反を起こす。
	output, err := uc.Execute(ctx, CreateSpaceInput{
		UserID: model.UserID(uuid.NewString()), Identifier: identifier, Name: "作成に失敗するスペース",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	var dbErr *pq.Error
	if !errors.As(err, &dbErr) || dbErr.Code != "23503" {
		t.Fatalf("外部キー制約違反を期待したが、%vだった", err)
	}
	if model.AsValidationError(err) != nil {
		t.Error("外部キー制約違反が入力エラーに変換された")
	}
	exists, err := spaceRepo.ExistsByIdentifier(ctx, identifier)
	if err != nil {
		t.Fatalf("失敗後のスペースの存在確認に失敗: %v", err)
	}
	if exists {
		t.Error("メンバー作成に失敗したスペースがロールバックされていない")
	}
}
