package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// oauthDeviceVerificationFixtureは、スペースのメンバーと、そのスペースのOAuthアプリが要求したデバイス認可
type oauthDeviceVerificationFixture struct {
	patMemberFixture
	applicationID         model.OAuthApplicationID
	deviceAuthorizationID model.OAuthDeviceAuthorizationID
	deviceCodeDigest      string

	// userCodeは区切りの無い形のユーザーコード。ダイジェストだけを保存している
	userCode string
}

// setupOAuthDeviceVerificationFixtureは、スペースkeyのroleのロールを持つメンバーと、そのスペースのOAuthアプリと、
// アプリがpage:readとpage:writeを要求したデバイス認可を作る。customizeで要求の状態や期限を変えられる
func setupOAuthDeviceVerificationFixture(t *testing.T, key string, role model.SpaceRole, flagEnabled bool, customize func(*testutil.OAuthDeviceAuthorizationBuilder)) oauthDeviceVerificationFixture {
	t.Helper()

	userCode, err := auth.GenerateOAuthUserCode()
	if err != nil {
		t.Fatalf("GenerateOAuthUserCode()のエラー = %v", err)
	}

	var f oauthDeviceVerificationFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		member := setupPATMember(t, tx, key, role, flagEnabled)
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(member.spaceID).
			WithName("連携CLI " + key).
			WithClientID(key + "-client").
			Build()
		b := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(appID).
			WithSpaceID(member.spaceID).
			WithScopes([]model.Scope{model.ScopePageRead, model.ScopePageWrite}).
			WithDeviceCodeDigest(key + "-device-code").
			WithUserCodeDigest(auth.DigestOpaqueToken(userCode))
		if customize != nil {
			customize(b)
		}
		f = oauthDeviceVerificationFixture{
			patMemberFixture:      member,
			applicationID:         appID,
			deviceAuthorizationID: b.Build(),
			deviceCodeDigest:      key + "-device-code",
			userCode:              userCode,
		}
	})
	return f
}

func newGetOAuthDeviceConsentUsecaseForTest(q *query.Queries) *GetOAuthDeviceConsentUsecase {
	return NewGetOAuthDeviceConsentUsecase(
		repository.NewFeatureFlagRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewTopicMemberRepository(q),
		validator.NewOAuthDeviceVerificationValidator(repository.NewOAuthDeviceAuthorizationRepository(q), repository.NewOAuthApplicationRepository(q), repository.NewSpaceRepository(q)),
	)
}

// assertInvalidOAuthDeviceUserCodeは、errがユーザーコードを受け付けないことを表すエラーであることを確かめる
func assertInvalidOAuthDeviceUserCode(t *testing.T, err error) {
	t.Helper()

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("エラー = %v、ValidationErrorを期待", err)
	}
	if !ve.HasFieldError(validator.OAuthDeviceUserCodeField) {
		t.Errorf("エラー = %+v、%sのエラーを期待", ve, validator.OAuthDeviceUserCodeField)
	}
}

func TestGetOAuthDeviceConsentUsecase_Execute(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())

	t.Run("小文字や区切りの無い入力でもコードを照合し、同意画面に示す内容を返す", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devconsent-ok", model.SpaceRoleViewer, true, nil)
		output, err := newGetOAuthDeviceConsentUsecaseForTest(q).Execute(context.Background(), GetOAuthDeviceConsentInput{
			UserID:   f.userID,
			UserCode: " " + strings.ToLower(f.userCode) + " ",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Application.ID != f.applicationID || output.Space.ID != f.spaceID {
			t.Errorf("アプリ = %v・スペース = %v、期待値 = %v・%v", output.Application.ID, output.Space.ID, f.applicationID, f.spaceID)
		}
		if want := auth.FormatOAuthUserCode(f.userCode); output.UserCode != want {
			t.Errorf("UserCode = %q、期待値 = %q", output.UserCode, want)
		}
		if want := []model.Scope{model.ScopePageRead, model.ScopePageWrite}; !slices.Equal(output.Scopes, want) {
			t.Errorf("Scopes = %v、期待値 = %v", output.Scopes, want)
		}
		if time.Since(output.RequestedAt) > time.Minute {
			t.Errorf("RequestedAt = %v、期待値は要求を作った日時", output.RequestedAt)
		}
		if output.DeniedReason != "" {
			t.Errorf("DeniedReason = %q、期待値は空", output.DeniedReason)
		}
		// スペース閲覧者はpage:writeを持たないため、今の権限では効かないことを示す
		if want := []model.Scope{model.ScopePageWrite}; !slices.Equal(output.IneffectiveScopes, want) {
			t.Errorf("IneffectiveScopes = %v、期待値 = %v", output.IneffectiveScopes, want)
		}
	})

	t.Run("連携先のスペースのメンバーでなければ、許可できない理由を返す", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devconsent-outsider", model.SpaceRoleAdmin, true, nil)
		outsider := setupOAuthDeviceVerificationFixture(t, "devconsent-outsider-user", model.SpaceRoleAdmin, true, nil)
		output, err := newGetOAuthDeviceConsentUsecaseForTest(q).Execute(context.Background(), GetOAuthDeviceConsentInput{
			UserID:   outsider.userID,
			UserCode: f.userCode,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DeniedReason != OAuthAuthorizationDeniedNotMember {
			t.Errorf("DeniedReason = %q、期待値 = %q", output.DeniedReason, OAuthAuthorizationDeniedNotMember)
		}
	})

	t.Run("受け付けないコードは、理由を区別せず同じエラーを返す", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			customize func(*testutil.OAuthDeviceAuthorizationBuilder)
			input     func(f oauthDeviceVerificationFixture) string
		}{
			{name: "形式の誤り", input: func(oauthDeviceVerificationFixture) string { return "ABC" }},
			{name: "該当する要求が無い", input: func(f oauthDeviceVerificationFixture) string { return "ZZZZZZZZ" }},
			{
				name:      "期限切れ",
				customize: func(b *testutil.OAuthDeviceAuthorizationBuilder) { b.WithExpiresAt(time.Now().Add(-time.Second)) },
			},
			{
				name: "拒否済み",
				customize: func(b *testutil.OAuthDeviceAuthorizationBuilder) {
					b.WithStatus(model.OAuthDeviceAuthorizationStatusDenied)
				},
			},
		}
		for i, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				f := setupOAuthDeviceVerificationFixture(t, "devconsent-invalid-"+string(rune('a'+i)), model.SpaceRoleAdmin, true, tt.customize)
				input := f.userCode
				if tt.input != nil {
					input = tt.input(f)
				}
				_, err := newGetOAuthDeviceConsentUsecaseForTest(q).Execute(context.Background(), GetOAuthDeviceConsentInput{
					UserID:   f.userID,
					UserCode: input,
				})
				assertInvalidOAuthDeviceUserCode(t, err)
			})
		}
	})

	t.Run("要求したアプリが削除されていれば、コードを受け付けない", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devconsent-discarded", model.SpaceRoleAdmin, true, nil)
		if _, err := testutil.GetTestDB().ExecContext(context.Background(), "UPDATE oauth_applications SET discarded_at = NOW() WHERE id = $1", string(f.applicationID)); err != nil {
			t.Fatalf("アプリの削除に失敗: %v", err)
		}
		_, err := newGetOAuthDeviceConsentUsecaseForTest(q).Execute(context.Background(), GetOAuthDeviceConsentInput{
			UserID:   f.userID,
			UserCode: f.userCode,
		})
		assertInvalidOAuthDeviceUserCode(t, err)
	})

	t.Run("フィーチャーフラグが無効なら、見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devconsent-flag", model.SpaceRoleAdmin, false, nil)
		_, err := newGetOAuthDeviceConsentUsecaseForTest(q).Execute(context.Background(), GetOAuthDeviceConsentInput{
			UserID:   f.userID,
			UserCode: f.userCode,
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、見つからないことを表すAppErrorを期待", err)
		}
	})
}
