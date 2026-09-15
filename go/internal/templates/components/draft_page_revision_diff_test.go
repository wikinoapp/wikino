package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// newTestRevisionDiffは新旧のタイトル・本文のペアから差分ビューモデルを生成する。
func newTestRevisionDiff(t *testing.T, oldTitle, newTitle, oldBody, newBody string) viewmodel.DraftPageRevisionDiff {
	t.Helper()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	previous := &model.DraftPageRevision{
		Title:     oldTitle,
		Body:      oldBody,
		CreatedAt: createdAt.Add(-time.Minute),
	}
	revision := &model.DraftPageRevision{
		Title:     newTitle,
		Body:      newBody,
		CreatedAt: createdAt,
	}
	return viewmodel.NewDraftPageRevisionDiff(revision, previous)
}

// renderRevisionDiffはDraftPageRevisionDiffコンポーネントを文字列にレンダリングする。
// isCurrentは表示中のリビジョンが最新かどうか (最新のときは復元領域を隠す) を切り替える。
func renderRevisionDiff(t *testing.T, ctx context.Context, diff viewmodel.DraftPageRevisionDiff, isCurrent bool) string {
	t.Helper()

	var buf bytes.Buffer
	if err := components.DraftPageRevisionDiff(components.DraftPageRevisionDiffData{
		Diff:       diff,
		RestoreURL: "/s/test-space/pages/1/draft_page_revisions/test-revision-id/restore",
		CSRFToken:  "test-csrf-token",
		IsCurrent:  isCurrent,
	}).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

func TestDraftPageRevisionDiff(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), "ja")

	t.Run("タイトル変更と本文差分を表示する", func(t *testing.T) {
		t.Parallel()

		diff := newTestRevisionDiff(t, "Old Title", "New Title", "line one\n", "line one\nline two\n")
		html := renderRevisionDiff(t, ctx, diff, false)

		if !strings.Contains(html, "タイトルの変更") {
			t.Error("タイトル変更ラベルが含まれていない")
		}
		if !strings.Contains(html, "Old Title") || !strings.Contains(html, "New Title") {
			t.Error("新旧タイトルが含まれていない")
		}
		if !strings.Contains(html, "line two") {
			t.Error("追加行が含まれていない")
		}
	})

	t.Run("タイトルが同じならタイトル変更を表示しない", func(t *testing.T) {
		t.Parallel()

		diff := newTestRevisionDiff(t, "Same Title", "Same Title", "a\n", "a\nb\n")
		html := renderRevisionDiff(t, ctx, diff, false)

		if strings.Contains(html, "タイトルの変更") {
			t.Error("タイトル変更ラベルが含まれている")
		}
	})

	t.Run("本文に変更がない場合は差分なしメッセージを表示する", func(t *testing.T) {
		t.Parallel()

		diff := newTestRevisionDiff(t, "Old Title", "New Title", "same\n", "same\n")
		html := renderRevisionDiff(t, ctx, diff, false)

		if !strings.Contains(html, "本文に変更はありません") {
			t.Error("差分なしメッセージが含まれていない")
		}
	})

	t.Run("現在でないバージョンでは復元ボタンとインライン確認を表示する", func(t *testing.T) {
		t.Parallel()

		diff := newTestRevisionDiff(t, "Old Title", "New Title", "a\n", "b\n")
		html := renderRevisionDiff(t, ctx, diff, false)

		if !strings.Contains(html, "このバージョンに戻す") {
			t.Error("復元ボタンが含まれていない")
		}
		if !strings.Contains(html, "本当にこのバージョンに戻しますか？") {
			t.Error("インライン確認メッセージが含まれていない")
		}
		// 確認フォームはCSRFトークン付きで復元URLへPOSTすること。
		if !strings.Contains(html, `action="/s/test-space/pages/1/draft_page_revisions/test-revision-id/restore"`) {
			t.Error("復元フォームのaction属性が含まれていない")
		}
		if !strings.Contains(html, `value="test-csrf-token"`) {
			t.Error("CSRFトークンが含まれていない")
		}
		// 確認UIは初期状態で非表示で、ボタン領域をインラインで置き換える (2重モーダルなし)。
		if !strings.Contains(html, `id="page-edit-revision-restore-confirm" class="hidden"`) {
			t.Error("インライン確認が初期非表示になっていない")
		}
	})

	t.Run("現在のバージョンでは復元ボタンを表示しない", func(t *testing.T) {
		t.Parallel()

		// 最新リビジョンは下書きの現在状態のため、そこへの復元はno-opであり復元領域は隠れる。
		// 差分自体は引き続き描画される。
		diff := newTestRevisionDiff(t, "Old Title", "New Title", "a\n", "b\n")
		html := renderRevisionDiff(t, ctx, diff, true)

		if strings.Contains(html, "このバージョンに戻す") {
			t.Error("現在のバージョンで復元ボタンが含まれている")
		}
		if strings.Contains(html, "本当にこのバージョンに戻しますか？") {
			t.Error("現在のバージョンでインライン確認メッセージが含まれている")
		}
		if strings.Contains(html, `action="/s/test-space/pages/1/draft_page_revisions/test-revision-id/restore"`) {
			t.Error("現在のバージョンで復元フォームのaction属性が含まれている")
		}
		// 復元領域が隠れていても差分本文は表示される。
		if !strings.Contains(html, "タイトルの変更") {
			t.Error("差分のタイトル変更ラベルが含まれていない")
		}
	})
}
