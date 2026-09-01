package pages_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	accountpage "github.com/wikinoapp/wikino/go/internal/templates/pages/account"
	emailconfirmationpage "github.com/wikinoapp/wikino/go/internal/templates/pages/email_confirmation"
	passwordpage "github.com/wikinoapp/wikino/go/internal/templates/pages/password"
	signinpage "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in"
	signintwofactorpage "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	signuppage "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_up"
)

// TestAuthFormsRenderOnlyGlobalErrorsInTheSummary covers every form that places the shared global
// error component between its heading and card. Field-only validation must stay beside the field
// without creating an empty flex child, while a global error must still render in the summary.
//
// [Ja] TestAuthFormsRenderOnlyGlobalErrorsInTheSummary は、見出しとカードの間へ共有の
// global error component を置く全フォームを確認する。フィールドだけの validation error は
// 空の flex 子要素を作らずフィールド横だけに残し、global error は概要へ引き続き表示する。
func TestAuthFormsRenderOnlyGlobalErrorsInTheSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		component func(*model.ValidationError) templ.Component
	}{
		{
			name:      "アカウント作成",
			fieldName: "atname",
			component: func(formErrors *model.ValidationError) templ.Component {
				return accountpage.New(accountpage.NewPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "メール確認",
			fieldName: "code",
			component: func(formErrors *model.ValidationError) templ.Component {
				return emailconfirmationpage.Edit(emailconfirmationpage.EditPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "パスワード変更",
			fieldName: "password",
			component: func(formErrors *model.ValidationError) templ.Component {
				return passwordpage.Edit(passwordpage.EditPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "パスワードリセット",
			fieldName: "email",
			component: func(formErrors *model.ValidationError) templ.Component {
				return passwordpage.Reset(passwordpage.ResetPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "サインイン",
			fieldName: "email",
			component: func(formErrors *model.ValidationError) templ.Component {
				return signinpage.New(signinpage.NewPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "2要素認証",
			fieldName: "totp_code",
			component: func(formErrors *model.ValidationError) templ.Component {
				return signintwofactorpage.New(signintwofactorpage.NewPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "リカバリーコード",
			fieldName: "recovery_code",
			component: func(formErrors *model.ValidationError) templ.Component {
				return signintwofactorpage.RecoveryNew(signintwofactorpage.RecoveryNewPageData{FormErrors: formErrors})
			},
		},
		{
			name:      "サインアップ",
			fieldName: "email",
			component: func(formErrors *model.ValidationError) templ.Component {
				return signuppage.New(signuppage.NewPageData{FormErrors: formErrors})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fieldErrors := model.NewValidationError()
			fieldErrors.AddField(tt.fieldName, "field-only-error")
			fieldBody := renderAuthForm(t, tt.component(fieldErrors))
			if !strings.Contains(fieldBody, "field-only-error") {
				t.Error("フィールドエラーが入力欄付近に表示されていない")
			}
			if strings.Contains(fieldBody, `<div class="px-4"></div>`) {
				t.Error("フィールドエラーだけのときに空の概要領域が表示されている")
			}

			globalErrors := model.NewValidationError()
			globalErrors.AddGlobal("global-error")
			globalBody := renderAuthForm(t, tt.component(globalErrors))
			if !strings.Contains(globalBody, `<div class="px-4"><div class="alert" data-variant="destructive">`) {
				t.Error("グローバルエラーの概要領域が表示されていない")
			}
			if !strings.Contains(globalBody, "global-error") {
				t.Error("グローバルエラーが表示されていない")
			}
		})
	}
}

func renderAuthForm(t *testing.T, component templ.Component) string {
	t.Helper()

	var body bytes.Buffer
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	if err := component.Render(ctx, &body); err != nil {
		t.Fatalf("認証フォームをレンダリングできなかった: %v", err)
	}

	return body.String()
}
