package legal_document

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/legaldoc"
)

// Showは法的文書の固定URL (GET /terms・/terms/api・/privacyと、その英語版の /en/...) を、本文を置く
// Wikinoのページへ301で転送します。固定URLは機械向けの記述にも載せる恒久的なURLで、本文のページは
// 改定してもページ番号が変わらないため、恒久的な転送にします。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	doc, ok := legaldoc.Lookup(r.URL.Path)
	if !ok {
		httperror.NotFound(w, r)
		return
	}
	http.Redirect(w, r, doc.PagePath, http.StatusMovedPermanently)
}
