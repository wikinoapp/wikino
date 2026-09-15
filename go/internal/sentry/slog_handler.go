package sentry

import (
	"context"
	"log/slog"

	sentryslog "github.com/getsentry/sentry-go/slog"
)

// ログイベントの発生源をタグ付けするslog属性のキー名。本キーに既知の
// 値 (例: ReverseProxySource) を載せたエラーログは、beforeSend側でその値を
// 検出してSentry送信を抑止する。
//
// 別の開発者が無関係な目的で汎用的な "source" 属性を追加した場合に衝突しない
// よう、"wikino_" プレフィックスで名前空間を切っている。呼び出し側では文字列
// リテラルを直書きせず、必ず本定数を参照すること。
const SourceAttrKey = "wikino_source"

// Rails版がエラーを返した際にリバースプロキシミドルウェアが
// SourceAttrKeyに設定する値。Rails側の障害 (HTTP 502など) はRailsの
// Sentryプロジェクトで扱うべきなので、beforeSendで本タグの付いた
// イベントを破棄する。
const ReverseProxySource = "reverse_proxy"

// baseハンドラーをsentryslogハンドラーと合成して返す。slog.LevelError
// とLevelFatalのレコードをSentryにイベントとして送信し、それ以外のレベルは
// baseにだけ流す。Sentry Logs APIへの送信は明示的に無効化している (本連携は
// エラー検出のみを担うため)。
func NewSlogHandler(base slog.Handler) slog.Handler {
	sentryHandler := sentryslog.Option{
		// ErrorとFatalだけをSentryイベント化する。
		EventLevel: []slog.Level{slog.LevelError, sentryslog.LevelFatal},
		// 空 (非nil) のスライスを渡すことでSentry Logs APIへの送信を
		// 無効化する。nilにするとパッケージ既定値 (全レベル送信) に
		// フォールバックしてしまう。
		LogLevel: []slog.Level{},
	}.NewSentryHandler(context.Background())
	return newMultiHandler(base, sentryHandler)
}

// 1レコードを複数のslog.Handlerにファンアウトする。baseのテキスト
// ハンドラーをそのままに、Sentry用ハンドラーを並列で動かすために使う。
// samber/slog-multiのような外部依存を避けるため、ファンアウトに必要最小限の
// 実装を内製している。
type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if !h.Enabled(ctx, record.Level) {
			continue
		}
		// ハンドラーごとにCloneすることで、Record.AddAttrsのように属性
		// 配列を直接書き換える実装が後続ハンドラーに影響しないようにする。
		if err := h.Handle(ctx, record.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	// slog-multiの挙動に合わせて最初のエラーだけ返す。Handleが返せる
	// エラーは1件のため、2件目以降は意図的に捨てている。
	return firstErr
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		out[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: out}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		out[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: out}
}
