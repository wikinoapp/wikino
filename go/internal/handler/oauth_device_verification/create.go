package oauth_device_verification

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	oauthdeviceverificationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_device_verification"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Createはユーザーコードの送信を受けます (POST /oauth/device)。
//
// `decision` の無い送信は入力画面からの送信で、コードを照合して同意画面を表示する。`decision` の
// ある送信は同意画面からの送信で、許可なら利用者が選んだスコープで、拒否ならそのまま要求を
// 更新して結果を示す。どちらの送信もコードを照合し直すため、ユーザー単位とIPアドレス単位で
// レート制限を掛けて総当たりを防ぐ。受け付けないコードは、誤り・期限切れ・使用済みを区別せず
// 入力画面を422で再表示する。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setResponseHeaders(w)

	user := middleware.UserFromContext(ctx)
	if user == nil {
		redirect.ToSignIn(w, r, model.OAuthDeviceVerificationPath)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.WarnContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// レート制限より先にフィーチャーフラグを確かめ、フラグが無効なユーザーには429ではなく404を返す
	if err := h.getOAuthDeviceVerificationNewUC.Execute(ctx, usecase.GetOAuthDeviceVerificationNewInput{UserID: user.ID}); err != nil {
		handleError(w, r, err)
		return
	}

	userCode := r.PostFormValue("user_code")
	if retryAfter, ok := h.checkRateLimit(r, user.ID); !ok {
		h.renderRateLimited(w, r, retryAfter, userCode)
		return
	}

	decision := r.PostFormValue("decision")
	if decision == "" {
		h.renderConsent(w, r, user, userCode, http.StatusOK, nil, nil)
		return
	}

	granted := r.PostForm["granted_scopes"]
	output, err := h.updateOAuthDeviceAuthorizationUC.Execute(ctx, usecase.UpdateOAuthDeviceAuthorizationInput{
		UserID:   user.ID,
		UserCode: userCode,
		// 許可のボタン以外からの送信は、利用者が許可を選んだものとして扱わない
		Approved:      decision == "approve",
		GrantedScopes: granted,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			if ve.HasFieldError(usecase.OAuthDeviceUserCodeField) {
				h.renderNew(w, r, http.StatusUnprocessableEntity, userCode, ve)
				return
			}
			// 選んだスコープが受け付けられなければ、選び直せるよう同意画面を再表示する
			h.renderConsent(w, r, user, userCode, http.StatusUnprocessableEntity, granted, ve)
			return
		}
		handleError(w, r, err)
		return
	}

	// 同意画面を出した後に許可できなくなっていれば、その理由を示す同意画面を出し直す
	if output.DeniedReason != "" {
		h.renderConsent(w, r, user, userCode, http.StatusForbidden, nil, nil)
		return
	}

	content := oauthdeviceverificationpages.Done(oauthdeviceverificationpages.DoneData{
		ApplicationName: output.Application.Name,
		Approved:        output.Approved,
	})
	h.render(w, r, "oauth_device_verification_done_title", nil, content)
}

// renderConsentはユーザーコードを照合し、同意画面をstatusで描画する。formErrorsがnilなら要求された
// スコープをすべてチェックし、拒否された送信の再表示では利用者が選んだスコープ (granted) をチェック
// したまま戻す。コードを受け付けなければ、入力画面を422で再表示する
func (h *Handler) renderConsent(w http.ResponseWriter, r *http.Request, user *model.User, userCode string, status int, granted []string, formErrors *model.ValidationError) {
	ctx := r.Context()

	output, err := h.getOAuthDeviceConsentUC.Execute(ctx, usecase.GetOAuthDeviceConsentInput{
		UserID:   user.ID,
		UserCode: userCode,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderNew(w, r, http.StatusUnprocessableEntity, userCode, ve)
			return
		}
		handleError(w, r, err)
		return
	}

	checked := output.Scopes
	if formErrors != nil {
		checked = model.StringsToScopes(granted)
	}

	content := oauthdeviceverificationpages.Consent(oauthdeviceverificationpages.ConsentData{
		CSRFToken:       middleware.GetCSRFTokenFromContext(ctx),
		UserCode:        output.UserCode,
		ApplicationName: output.Application.Name,
		SpaceName:       output.Space.Name,
		UserAtname:      string(user.Atname),
		RequestedAt:     output.RequestedAt,
		ScopeOptions:    components.NewAPITokenScopeOptions(output.Scopes, checked, output.IneffectiveScopes),
		NotMember:       output.DeniedReason == usecase.OAuthAuthorizationDeniedNotMember,
		NoPermission:    output.DeniedReason == usecase.OAuthAuthorizationDeniedNoPermission,
		FormErrors:      formErrors,
	})

	w.WriteHeader(status)
	h.render(w, r, "oauth_device_verification_consent_title", map[string]any{"ApplicationName": output.Application.Name}, content)
}
