package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SentryTransaction wires chi's matched route pattern into Sentry as the
// transaction name (e.g. "GET /s/{space_identifier}/topics/{topic_number}"),
// so the high-cardinality URL parameter values do not blow up the transaction
// list in Sentry. The middleware does two things:
//
//  1. On entry it attaches an EventProcessor to the request scope that lazily
//     fills in event.Transaction for error events using chi's RoutePattern at
//     capture time. This covers both panic events emitted by sentryhttp's
//     recoverWithSentry and explicit hub.CaptureException calls inside the
//     handler. chi populates RoutePattern as part of route dispatch (i.e.
//     before the matched handler runs), so reading it at capture time is safe.
//
//  2. On exit (defer) it overwrites the in-flight transaction's Name and
//     Source so the transaction event (performance trace) emitted by
//     sentryhttp's transaction.Finish() carries the route pattern instead of
//     the raw URL.
//
// Register this middleware AFTER sentryhttp so it lives inside the sentryhttp
// wrapper. LIFO defer order then guarantees:
//   - on a normal response: this defer rewrites span Name / Source before
//     sentryhttp's defer calls transaction.Finish().
//   - on a panic: this defer runs first as the stack unwinds, so by the time
//     sentryhttp's recoverWithSentry captures the panic event, the route
//     pattern is already reachable for the EventProcessor that fills in
//     event.Transaction.
//
// [Ja] chi のマッチしたルートパターン (例:
// "GET /s/{space_identifier}/topics/{topic_number}") を Sentry の
// transaction 名として使うミドルウェア。URL パラメータ値の高カーディナリティで
// Sentry のトランザクション一覧が爆発するのを防ぐ。ミドルウェアは 2 つの仕事
// をする:
//
//  1. 入口でリクエストスコープに EventProcessor を仕込む。これは error
//     イベントの event.Transaction を chi.RouteContext().RoutePattern() から
//     キャプチャ時に遅延埋めする。sentryhttp の recoverWithSentry 経由の
//     panic イベントも、ハンドラー内の明示的な hub.CaptureException も
//     どちらでも動く。chi はルートマッチ時 (= ハンドラー実行前) に
//     RoutePattern を確定させるため、キャプチャ時点での読み出しは安全。
//
//  2. 出口の defer で進行中のトランザクションの Name / Source を上書きする。
//     sentryhttp の transaction.Finish() が送るトランザクションイベント
//     (パフォーマンストレース) に、生 URL ではなくルートパターンが乗る。
//
// このミドルウェアは sentryhttp の **あとに登録** すること (= sentryhttp の
// 内側)。LIFO の defer 順序により以下が保証される:
//   - 正常応答時: 本 defer が span の Name / Source を書き換えてから
//     sentryhttp の defer が transaction.Finish() を呼ぶ。
//   - panic 時: スタック巻き戻し中に本 defer が先に走るため、sentryhttp の
//     recoverWithSentry が panic イベントを捕捉する時点で、EventProcessor が
//     event.Transaction を埋めるために必要なルートパターンに到達できる。
func SentryTransaction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.Scope().AddEventProcessor(sentryTransactionEventProcessor(r))
		}
		defer applySentryTransactionName(r)
		next.ServeHTTP(w, r)
	})
}

// sentryTransactionEventProcessor returns an EventProcessor that fills in
// event.Transaction from chi's RoutePattern for error events. Transaction-type
// events get their Transaction field from span.Name (set in
// applySentryTransactionName) and are left untouched here.
//
// [Ja] error イベントの event.Transaction を chi の RoutePattern から埋める
// EventProcessor を返す。transaction 種別のイベントは span.Name
// (applySentryTransactionName で設定) から Transaction を得るため、ここでは
// 触らない。
func sentryTransactionEventProcessor(r *http.Request) sentry.EventProcessor {
	return func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
		if event == nil || event.Type == "transaction" || event.Transaction != "" {
			return event
		}
		if pattern := matchedRoutePattern(r); pattern != "" {
			event.Transaction = r.Method + " " + pattern
		}
		return event
	}
}

func applySentryTransactionName(r *http.Request) {
	pattern := matchedRoutePattern(r)
	if pattern == "" {
		return
	}
	name := r.Method + " " + pattern
	if transaction := sentry.TransactionFromContext(r.Context()); transaction != nil {
		transaction.Name = name
		transaction.Source = sentry.SourceRoute
	}
}

func matchedRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return rctx.RoutePattern()
}

// SentryUserContext attaches the authenticated user to the Sentry Hub scope so
// subsequent events captured during this request carry the user's ID and (when
// available) Atname as Username. Register this middleware AFTER both:
//
//   - the auth middleware that populates UserFromContext (so the user is
//     available on the context), and
//   - sentryhttp (so a per-request Hub is on the context to scope the user to
//     this request rather than the global hub).
//
// The middleware silently no-ops when:
//
//   - the request is unauthenticated (UserFromContext(ctx) == nil), so anonymous
//     traffic never inherits a stale user from a previous request, and
//   - the context has no Sentry Hub (e.g. paths routed outside the main
//     sentryhttp chain such as static files), so the middleware is safe to add
//     anywhere in the tree.
//
// The Hub stored on the request context is a per-request clone created by
// sentryhttp, so calling SetUser here does not leak user information into other
// requests' scopes.
//
// [Ja] 認証済みユーザー情報を Sentry の Hub スコープに紐付ける。本ミドルウェア
// 通過後に同一リクエスト内でキャプチャされたイベントには、ユーザーの ID と
// (利用可能なら) Atname が Username として乗る。以下の **両方** より後に登録
// すること:
//
//   - UserFromContext を埋める認証ミドルウェア (ユーザーを context から
//     取り出せるようにするため)
//   - sentryhttp (グローバル Hub ではなくリクエスト単位の Hub にユーザーを
//     スコープするため)
//
// 以下のケースでは何もせず次のハンドラーに進む:
//
//   - 未認証リクエスト (UserFromContext(ctx) == nil)。匿名トラフィックが
//     直前のリクエストのユーザー情報を引き継ぐのを防ぐ。
//   - context に Sentry Hub が存在しない (例: 静的ファイル経路など、本番の
//     sentryhttp チェーンを通らないパス)。ツリーのどこに置いても安全に動く。
//
// リクエスト context に乗っている Hub は sentryhttp が clone した
// リクエスト単位のものなので、ここで SetUser を呼んでも他のリクエストの
// スコープに漏れることはない。
func SentryUserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := UserFromContext(ctx)
		if user == nil {
			next.ServeHTTP(w, r)
			return
		}
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			next.ServeHTTP(w, r)
			return
		}
		hub.Scope().SetUser(sentryUserFromModel(user))
		next.ServeHTTP(w, r)
	})
}

// sentryUserFromModel builds a sentry.User from the domain user, omitting the
// Username when Atname is empty so events from users that predate atname
// assignment still carry a stable ID.
//
// [Ja] domain の User から sentry.User を組み立てる。Atname が空の場合は
// Username を省略し、Atname 未設定の (移行前データなどの) ユーザーでも
// 安定した ID は乗るようにする。
func sentryUserFromModel(user *model.User) sentry.User {
	su := sentry.User{ID: user.ID.String()}
	if user.Atname != "" {
		su.Username = user.Atname
	}
	return su
}
