package usecase

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreatePersonalAccessTokenUsecaseForTest(q *query.Queries) *CreatePersonalAccessTokenUsecase {
	return NewCreatePersonalAccessTokenUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewPersonalAccessTokenRepository(q),
		validator.NewPersonalAccessTokenCreateValidator(),
	)
}

func TestCreatePersonalAccessTokenUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupPATMember(t, tx, "cpat-success", model.SpaceRoleViewer, true)

	before := time.Now()
	output, err := newCreatePersonalAccessTokenUsecaseForTest(q).Execute(context.Background(), CreatePersonalAccessTokenInput{
		SpaceIdentifier: "cpat-success",
		UserID:          f.userID,
		Name:            "CLI",
		Scopes:          []string{"page:write", "topic:read"},
		ExpirationDays:  "90",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if !strings.HasPrefix(output.TokenValue, string(auth.PersonalAccessTokenPrefix)) {
		t.Errorf("TokenValue = %q、期待値は接頭辞%qで始まる", output.TokenValue, auth.PersonalAccessTokenPrefix)
	}

	// データベースには値ではなくダイジェストが残り、そのダイジェストで引ける。
	stored, err := repository.NewPersonalAccessTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(output.TokenValue))
	if err != nil {
		t.Fatalf("FindByTokenDigest()のエラー = %v", err)
	}
	if stored == nil || stored.ID != output.Token.ID {
		t.Fatalf("保存されたトークン = %v、期待値のID = %v", stored, output.Token.ID)
	}
	if stored.TokenDigest == output.TokenValue {
		t.Error("トークンの値がそのまま保存されている")
	}
	if stored.SpaceID != f.spaceID || stored.SpaceMemberID != f.spaceMemberID {
		t.Errorf("束縛先 = (%v, %v)、期待値 = (%v, %v)", stored.SpaceID, stored.SpaceMemberID, f.spaceID, f.spaceMemberID)
	}
	if stored.Name != "CLI" {
		t.Errorf("Name = %q、期待値 = %q", stored.Name, "CLI")
	}
	if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(stored.Scopes, want) {
		t.Errorf("Scopes = %v、期待値 = %v", stored.Scopes, want)
	}
	if want := auth.OpaqueTokenLastChars(output.TokenValue); stored.TokenLastChars != want {
		t.Errorf("TokenLastChars = %q、期待値 = %q", stored.TokenLastChars, want)
	}
	wantExpiresAt := before.AddDate(0, 0, 90)
	if stored.ExpiresAt.Before(wantExpiresAt) || stored.ExpiresAt.After(wantExpiresAt.Add(time.Minute)) {
		t.Errorf("ExpiresAt = %v、期待値はおよそ%v", stored.ExpiresAt, wantExpiresAt)
	}
}

func TestCreatePersonalAccessTokenUsecase_Execute_発行できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		role        model.SpaceRole
		flagEnabled bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "フィーチャーフラグが無効",
			key:         "cpat-noflag",
			role:        model.SpaceRoleViewer,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.role, tt.flagEnabled)

			_, err := newCreatePersonalAccessTokenUsecaseForTest(q).Execute(context.Background(), CreatePersonalAccessTokenInput{
				SpaceIdentifier: model.SpaceIdentifier(tt.key),
				UserID:          f.userID,
				Name:            "CLI",
				Scopes:          []string{"page:read"},
				ExpirationDays:  "30",
			})
			assertAppErrCode(t, err, tt.wantErrCode)
			assertNoPersonalAccessToken(t, q, f)
		})
	}

	t.Run("入力が不正ならValidationErrorを返し発行しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "cpat-invalid", model.SpaceRoleViewer, true)

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
		_, err := newCreatePersonalAccessTokenUsecaseForTest(q).Execute(ctx, CreatePersonalAccessTokenInput{
			SpaceIdentifier: "cpat-invalid",
			UserID:          f.userID,
			Name:            "CLI",
			Scopes:          []string{"space:admin"},
			ExpirationDays:  "30",
		})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = ValidationError", err)
		}
		if !ve.HasFieldError("scopes") {
			t.Error("scopesのフィールドエラーが無い")
		}
		assertNoPersonalAccessToken(t, q, f)
	})
}

// assertNoPersonalAccessTokenは、メンバーのトークンが1つも作られていないことを確かめる。
func assertNoPersonalAccessToken(t *testing.T, q *query.Queries, f patMemberFixture) {
	t.Helper()

	tokens, err := repository.NewPersonalAccessTokenRepository(q).ListUnrevokedBySpaceMember(context.Background(), f.spaceID, f.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("作られたトークンの件数 = %d、期待値 = 0", len(tokens))
	}
}
