# コードレビュー: page-edit-8b-1

## レビュー情報

| 項目                       | 内容                                                                                                                             |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-02-28                                                                                                                       |
| 対象ブランチ               | page-edit-8b-1                                                                                                                   |
| ベースブランチ             | page-edit                                                                                                                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md（タスク8b-1d, 8b-1e）、docs/plans/1_doing/datastar-skill.md（タスク2-1）             |
| 変更ファイル数             | 17 ファイル（うち自動生成・依存関係ファイル5）                                                                                   |
| 変更行数（実装）           | +392 / -2 行（go.mod/go.sum/\_templ.go除く）                                                                                    |
| 変更行数（テスト）         | +0 / -0 行                                                                                                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page_link_list/handler.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/web/markdown-editor/markdown-editor.ts`
- [x] `go/cmd/server/main.go`

### テストファイル

（テストファイルなし）

### 設定・その他

- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/internal/templates/components/link_list_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/1_doing/datastar-skill.md`

## ファイルごとのレビュー結果

### テストファイル全般

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化
- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テストのベストプラクティス

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: 実装コードに対応するテストが含まれていない

  PRガイドラインでは「実装コードとそのテストコードは同じPRに含める」「テストがない実装は原則としてマージしない」と定められている。今回の変更では以下のテストが不足している:

  - `internal/handler/page_link_list/show.go` のハンドラーテスト（認証・認可チェック、SSEレスポンス）
  - `internal/viewmodel/link_list.go` のユニットテスト（ViewModel変換ロジック）

  **修正案**:

  最低限以下のテストを追加する:

  1. `internal/viewmodel/link_list_test.go`: `NewLinkList`のユニットテスト（ページリスト→ViewModel変換、タイトルnilの場合の空文字フォールバック）
  2. `internal/handler/page_link_list/show_test.go`: SSEハンドラーの統合テスト（認証なし→401、スペース不在→404、正常系→200 SSEレスポンス）

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] テストを追加する（上記の修正案に従う）
  - [ ] テストは別PRで追加する（理由を回答欄に記入）
  - [ ] テスト不要と判断する（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `docs/plans/1_doing/page-edit-go-migration.md`

**ステータス**: 要修正

**チェックしたガイドライン**:

- タスクリストの進捗管理

**問題点・改善提案**:

- **タスクチェックボックスの未更新**: タスク8b-1dと8b-1eが実装済みだがチェックボックスが `[ ]` のまま

  ```markdown
  // 現在の状態
  - [ ] **8b-1d**: [Go] Datastar Go SDKの追加とリンク一覧SSEエンドポイントの作成
  - [ ] **8b-1e**: リンク一覧更新のDatastar化（plain JS→Datastar SSE）
  ```

  **修正案**:

  ```markdown
  - [x] **8b-1d**: [Go] Datastar Go SDKの追加とリンク一覧SSEエンドポイントの作成
  - [x] **8b-1e**: リンク一覧更新のDatastar化（plain JS→Datastar SSE）
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] チェックボックスを更新する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

Datastar Go SDK の導入とリンク一覧SSEエンドポイントの実装、およびplain JS→Datastarへのリファクタリングが作業計画書通りに実施されている。

**良かった点**:

- ハンドラーの構造が既存パターン（page, draft_page）と一貫している
- 認証・認可チェック（ユーザー認証→スペースメンバー→トピックポリシー）が適切に実装されている
- Datastar Go SDKの使用パターン（`NewSSE` + `PatchElementTempl` + `WithSelectorID` + `WithModeInner`）が正しい
- カスタムイベント（`draft-autosaved`）によるJS→Datastar連携がクリーンに実装されている
- `slog.ErrorContext` によるエラーログが一貫して使用されている
- I18n（ja/en両方）が適切に追加されている
- ViewModel（`LinkList`）がtemplテンプレートガイドのデータ構造体パターンに従っている
- edit.go のリンク一覧取得ロジック（DraftPage優先→Page fallback）が仕様通り

**改善が必要な点**:

- テストが含まれていない（PRガイドライン違反）。ハンドラーとViewModelのテストを追加すべき
- 作業計画書のタスクチェックボックスが未更新（軽微）
