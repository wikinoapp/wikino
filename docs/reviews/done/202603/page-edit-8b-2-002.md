# コードレビュー: page-edit-8b-2

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-01                                    |
| 対象ブランチ               | page-edit-8b-2                                |
| ベースブランチ             | page-edit                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md  |
| 変更ファイル数             | 11 ファイル（うち自動生成 2、ドキュメント 2） |
| 変更行数（実装）           | +113 / -11 行（自動生成ファイル除く）         |
| 変更行数（テスト）         | +122 / -0 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/templates/components/page_backlink_list.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/templates/components/page_backlink_list_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/components/page_backlink_list_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`

### ドキュメント

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/reviews/done/202603/page-edit-8b-2-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計改善の提案

### `go/internal/templates/pages/page/edit.templ`: `page-backlink-list` の `data-on:draft-autosaved__window` が重複リクエストを発生させる

**ステータス**: 要確認

**現状**:

`#page-link-list` と `#page-backlink-list` の両方が同じ `data-on:draft-autosaved__window` イベントハンドラーを持ち、同じURL（`GoPageDraftPagePath`）に `@get()` を発行する:

```html
<!-- リンク一覧: draft-autosaved イベントでGETを発行 -->
<div id="page-link-list" class="mt-6" data-on:draft-autosaved__window="@get('...')"></div>

<!-- バックリンク一覧: 同じイベントで同じURLへGETを発行 -->
<div id="page-backlink-list" class="mt-6" data-on:draft-autosaved__window="@get('...')"></div>
```

`draft-autosaved` イベント発火時に、Datastarは両方のハンドラーを実行するため、同じエンドポイントに対して2回のHTTPリクエストが発生する。`draft_page/show.go` のSSEレスポンスには保存時刻・リンク一覧・バックリンク一覧の3つのフラグメントがすべて含まれており、Datastarは `WithSelectorID` で指定されたIDに基づいてグローバルにDOM更新を行う。そのため、`#page-link-list` のリクエストだけで `#page-backlink-list` も更新される。

**提案**:

`#page-backlink-list` から `data-on:draft-autosaved__window` 属性を削除する:

```html
<!-- バックリンク一覧: リンク一覧のリクエストで自動的に更新される -->
<div id="page-backlink-list" class="mt-6">@components.PageBacklinkList(data.BacklinkList)</div>
```

`#page-draft-saved-at` も同様にイベントハンドラーを持たず、`#page-link-list` のリクエストのSSEレスポンスで更新される設計になっている。

**メリット**:

- 自動保存のたびに発生するHTTPリクエストが2回から1回に削減される
- `#page-draft-saved-at` と同じ設計パターンに統一される

**トレードオフ**:

- `#page-link-list` が何らかの理由でDOMから消えた場合、バックリンク一覧が更新されなくなる（ただし、そのケースは発生しない想定）

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 提案通り `data-on:draft-autosaved__window` を削除する
- [ ] 現状のまま（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

タスク 8b-2（ページ編集画面のバックリンク一覧表示の追加）が作業計画書の仕様通りに実装されている。

**良い点**:

- バックリンクデータのソースが公開済みページ（`pg`）に基づいており、仕様通り
- 既存の `BacklinkList` ビューモデルを適切に再利用している
- `PageBacklinkList` という命名で、既存の `BacklinkList`（リンク一覧内のネストされたバックリンク）と明確に区別している
- Datastar SSEパターンに準拠し、`draft_page/show.go` でバックリンク一覧フラグメントも送信している
- テストが正常系・空リスト・タイトルなしの3パターンを網羅している
- 翻訳ファイルが日英両方に追加されている
- `edit.templ` の `data-markdown-editor-draft-save-url` が `GoPageDraftPagePath` ヘルパーに置き換えられている改善も含まれている

**改善提案**: 1件（重複リクエストの削減。軽微であり、マージをブロックするものではない）
