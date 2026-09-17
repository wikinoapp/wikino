package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// chiのマッチしたルートパターン (例:
// "GET /s/{space_identifier}/topics/{topic_number}") をSentryの
// transaction名として使うミドルウェア。URLパラメータ値の高カーディナリティで
// Sentryのトランザクション一覧が爆発するのを防ぐ。ミドルウェアは2つの仕事
// をする:
//
//  1. 入口でリクエストスコープにEventProcessorを仕込む。これはerror
//     イベントのevent.Transactionをchi.RouteContext().RoutePattern() から
//     キャプチャ時に遅延埋めする。sentryhttpのrecoverWithSentry経由の
//     panicイベントも、ハンドラー内の明示的なhub.CaptureExceptionも
//     どちらでも動く。chiはルートマッチ時 (= ハンドラー実行前) に
//     RoutePatternを確定させるため、キャプチャ時点での読み出しは安全。
//
//  2. 出口のdeferで進行中のトランザクションのName / Sourceを上書きする。
//     sentryhttpのtransaction.Finish() が送るトランザクションイベント
//     (パフォーマンストレース) に、生URLではなくルートパターンが乗る。
//
// このミドルウェアはsentryhttpの **あとに登録** すること (= sentryhttpの
// 内側)。LIFOのdefer順序により以下が保証される:
//   - 正常応答時: 本deferがspanのName / Sourceを書き換えてから
//     sentryhttpのdeferがtransaction.Finish() を呼ぶ。
//   - panic時: スタック巻き戻し中に本deferが先に走るため、sentryhttpの
//     recoverWithSentryがpanicイベントを捕捉する時点で、EventProcessorが
//     event.Transactionを埋めるために必要なルートパターンに到達できる。
func SentryTransaction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.Scope().AddEventProcessor(sentryTransactionEventProcessor(r))
		}
		defer applySentryTransactionName(r)
		next.ServeHTTP(w, r)
	})
}

// errorイベントのevent.TransactionをchiのRoutePatternから埋める
// EventProcessorを返す。transaction種別のイベントはspan.Name
// (applySentryTransactionNameで設定) からTransactionを得るため、ここでは
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

// 認証済みユーザー情報をSentryのHubスコープに紐付ける。本ミドルウェア
// 通過後に同一リクエスト内でキャプチャされたイベントには、ユーザーのIDと
// (利用可能なら) AtnameがUsernameとして乗る。以下の **両方** より後に登録
// すること:
//
//   - UserFromContextを埋める認証ミドルウェア (ユーザーをcontextから
//     取り出せるようにするため)
//   - sentryhttp (グローバルHubではなくリクエスト単位のHubにユーザーを
//     スコープするため)
//
// 以下のケースでは何もせず次のハンドラーに進む:
//
//   - 未認証リクエスト (UserFromContext(ctx) == nil)。匿名トラフィックが
//     直前のリクエストのユーザー情報を引き継ぐのを防ぐ。
//   - contextにSentry Hubが存在しない (例: 静的ファイル経路など、本番の
//     sentryhttpチェーンを通らないパス)。ツリーのどこに置いても安全に動く。
//
// リクエストcontextに乗っているHubはsentryhttpがcloneした
// リクエスト単位のものなので、ここでSetUserを呼んでも他のリクエストの
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

// domainのUserからsentry.Userを組み立てる。Atnameが空の場合は
// Usernameを省略し、Atname未設定の (移行前データなどの) ユーザーでも
// 安定したIDは乗るようにする。
func sentryUserFromModel(user *model.User) sentry.User {
	su := sentry.User{ID: user.ID.String()}
	if user.Atname != "" {
		su.Username = user.Atname
	}
	return su
}
