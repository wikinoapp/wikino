package oauth_device_verification

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Newはユーザーコードの入力画面を表示します (GET /oauth/device)。
//
// 未ログインの利用者はサインインへ送り、サインイン後にこの画面へ戻す。`verification_uri_complete`
// で開いたときは、クエリの `user_code` を入力欄に入れておく。照合はせず、利用者が端末に表示された
// コードと見比べてから送信するようにする (他人から渡されたURLを開いただけで同意画面へ進ませない)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setResponseHeaders(w)

	user := middleware.UserFromContext(ctx)
	if user == nil {
		redirect.ToSignIn(w, r, r.URL.RequestURI())
		return
	}

	if err := h.getOAuthDeviceVerificationNewUC.Execute(ctx, usecase.GetOAuthDeviceVerificationNewInput{UserID: user.ID}); err != nil {
		handleError(w, r, err)
		return
	}

	h.renderNew(w, r, http.StatusOK, r.URL.Query().Get("user_code"), nil)
}
