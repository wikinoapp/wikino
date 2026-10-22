package sentry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/getsentry/sentry-go"
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

// baseハンドラーをSentryイベント用ハンドラーと合成して返す。
// ErrorとFatalだけをイベント化し、Sentry Logs APIには送信しない。
func NewSlogHandler(base slog.Handler) slog.Handler {
	return newMultiHandler(base, &sentryEventHandler{})
}

type sentryEventAttr struct {
	key       string
	value     string
	exception error
}

// 属性は追加時に解決して保持し、派生ハンドラー間で書き換えを共有しない。
// グループはドット区切りのキーにし、beforeSendで末端の機密キーも判定できるようにする。
type sentryEventHandler struct {
	attrs  []sentryEventAttr
	prefix string
}

func (h *sentryEventHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level == slog.LevelError || level == sentryslog.LevelFatal
}

func (h *sentryEventHandler) Handle(ctx context.Context, record slog.Record) error {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	client := hub.Client()
	if !client.IsEnabled() {
		return nil
	}

	event := sentry.NewEvent()
	event.Timestamp = record.Time.UTC()
	event.Level = sentry.LevelError
	if record.Level == sentryslog.LevelFatal {
		event.Level = sentry.LevelFatal
	}
	event.Message = record.Message
	event.Logger = "slog"

	var originalErr error
	addAttrs := func(attrs []sentryEventAttr) {
		for _, attr := range attrs {
			if attr.exception != nil {
				if originalErr == nil {
					originalErr = attr.exception
				}
				continue
			}
			event.Tags[attr.key] = attr.value
		}
	}
	addAttrs(h.attrs)
	record.Attrs(func(attr slog.Attr) bool {
		addAttrs(flattenSlogAttr(h.prefix, attr))
		return true
	})
	event.SetException(originalErr, client.Options().MaxErrorDepth)
	hub.CaptureEventWithHint(event, &sentry.EventHint{Context: ctx, OriginalException: originalErr})
	return nil
}

func (h *sentryEventHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := &sentryEventHandler{
		attrs:  make([]sentryEventAttr, len(h.attrs), len(h.attrs)+len(attrs)),
		prefix: h.prefix,
	}
	copy(out.attrs, h.attrs)
	for _, attr := range attrs {
		out.attrs = append(out.attrs, flattenSlogAttr(h.prefix, attr)...)
	}
	return out
}

func (h *sentryEventHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &sentryEventHandler{attrs: h.attrs, prefix: h.prefix + name + "."}
}

// 無名グループを展開し、error / errは例外情報としてタグから分離する。
func flattenSlogAttr(prefix string, attr slog.Attr) []sentryEventAttr {
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			prefix += attr.Key + "."
		}
		var attrs []sentryEventAttr
		for _, child := range value.Group() {
			attrs = append(attrs, flattenSlogAttr(prefix, child)...)
		}
		return attrs
	}
	if attr.Key == "" {
		return nil
	}
	var exception error
	if attr.Key == "error" || attr.Key == "err" {
		exception, _ = value.Any().(error)
	}
	return []sentryEventAttr{{key: prefix + attr.Key, value: fmt.Sprint(value.Any()), exception: exception}}
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
