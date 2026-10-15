package validator_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestOAuthDeviceVerificationValidator_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		status     model.OAuthDeviceAuthorizationStatus
		expired    bool
		discarded  bool
		otherSpace bool
		official   bool
		valid      bool
	}{
		{name: "小文字と区切りと前後の空白を正規化する", input: " bcdf-ghjk ", valid: true},
		{name: "公式クライアントも連携先のスペースで照合できる", input: "BCDFGHJK", official: true, valid: true},
		{name: "形式が不正なコード", input: "ABC"},
		{name: "該当しないコード", input: "ZZZZZZZZ"},
		{name: "期限切れ", input: "BCDFGHJK", expired: true},
		{name: "許可済み", input: "BCDFGHJK", status: model.OAuthDeviceAuthorizationStatusApproved},
		{name: "拒否済み", input: "BCDFGHJK", status: model.OAuthDeviceAuthorizationStatusDenied},
		{name: "交換済み", input: "BCDFGHJK", status: model.OAuthDeviceAuthorizationStatusExchanged},
		{name: "削除済みアプリ", input: "BCDFGHJK", discarded: true},
		{name: "アプリが別のスペースに属する", input: "BCDFGHJK", otherSpace: true},
	}
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
			key := fmt.Sprintf("device-validator-%d", index)
			userID := testutil.NewUserBuilder(t, tx).WithEmail(key + "@example.com").WithAtname(strings.ReplaceAll(key, "-", "_")).Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key).Build()
			memberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
			appSpaceID := spaceID
			if tt.otherSpace {
				appSpaceID = testutil.NewSpaceBuilder(t, tx).WithIdentifier(key + "-other").Build()
			}
			app := testutil.NewOAuthApplicationBuilder(t, tx).WithClientID(key)
			if tt.official {
				app.AsOfficialClient()
			} else {
				app.WithSpaceID(appSpaceID)
			}
			appID := app.Build()
			device := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
				WithOAuthApplicationID(appID).WithSpaceID(spaceID).
				WithDeviceCodeDigest(key + "-device").WithUserCodeDigest(auth.DigestOpaqueToken("BCDFGHJK")).
				WithScopes([]model.Scope{model.ScopePageRead}).WithStatus(tt.status)
			if tt.expired {
				device.WithExpiresAt(time.Now().Add(-time.Second))
			}
			if tt.status == model.OAuthDeviceAuthorizationStatusApproved || tt.status == model.OAuthDeviceAuthorizationStatusExchanged {
				grantID := testutil.NewOAuthGrantBuilder(t, tx).WithOAuthApplicationID(appID).WithSpaceID(spaceID).WithSpaceMemberID(memberID).Build()
				device.WithOAuthGrantID(grantID)
			}
			deviceID := device.Build()
			if tt.discarded {
				if _, err := tx.ExecContext(ctx, "UPDATE oauth_applications SET discarded_at = NOW() WHERE id = $1", string(appID)); err != nil {
					t.Fatal(err)
				}
			}
			v := validator.NewOAuthDeviceVerificationValidator(repository.NewOAuthDeviceAuthorizationRepository(q), repository.NewOAuthApplicationRepository(q), repository.NewSpaceRepository(q))
			output, err := v.Validate(ctx, validator.OAuthDeviceVerificationValidatorInput{UserCode: tt.input})
			if tt.valid {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				if output == nil || output.DeviceAuthorization.ID != deviceID || output.Application.ID != appID || output.Space.ID != spaceID || output.UserCode != "BCDF-GHJK" {
					t.Fatalf("出力 = %+v、照合した要求・アプリ・スペースと表示用コードを期待", output)
				}
				return
			}
			ve := model.AsValidationError(err)
			if output != nil || ve == nil {
				t.Fatalf("出力 = %+v、エラー = %v、出力なしとValidationErrorを期待", output, err)
			}
			want := i18n.T(ctx, "validation_oauth_device_user_code_invalid")
			messages := ve.GetFieldErrors(validator.OAuthDeviceUserCodeField)
			if len(messages) != 1 || messages[0] != want {
				t.Errorf("エラー = %v、期待値 = [%s]", messages, want)
			}
		})
	}
}
