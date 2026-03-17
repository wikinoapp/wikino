# コードレビュー: suggestion-2-3

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-17                           |
| 対象ブランチ               | suggestion-2-3                       |
| ベースブランチ             | page-title-rename                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md     |
| 変更ファイル数             | 10 ファイル                          |
| 変更行数（実装）           | +84 / -25 行（自動生成ファイル除く） |
| 変更行数（テスト）         | +106 / -3 行                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/topic/show.go`
- [x] `go/internal/usecase/get_topic_detail.go`
- [x] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/templates/pages/topic/show_templ.go`（自動生成）
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/topic/show_test.go`
- [x] `go/internal/usecase/get_topic_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/get_topic_detail.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の設計
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/coding-guide.md#コーディング規約]**: `uc.featureFlagRepo != nil` のnilガードが不要と思われる

  ```go
  // 現在のコード (120行目)
  if input.UserID != nil && uc.featureFlagRepo != nil {
  ```

  `featureFlagRepo` はコンストラクタ `NewGetTopicDetailUsecase` の必須引数として渡されており、nilが渡されることは想定されていない。他のリポジトリ（`spaceRepo`, `topicRepo` 等）にはnilガードがないため、一貫性がない。このnilガードがあると「nilを渡しても安全」という誤ったメッセージを与え、バグを隠す可能性がある。

  **修正案**:

  ```go
  if input.UserID != nil {
  ```

  **対応方針**:
  - [x] nilガードを削除する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク2-3（トピック詳細画面に「編集提案」タブを追加）の要件を正確に実装している。

**良かった点**:

- フィーチャーフラグによるタブの表示制御が作業計画書の仕様通りに実装されている
- UseCase → Handler → Template の依存方向が正しく、アーキテクチャガイドに準拠
- フィーチャーフラグの有効/無効の両方のテストケースが追加されている
- 翻訳ファイル（ja.toml, en.toml）の追加が i18n ガイドの命名規則に従っている
- `SuggestionEnabled` を `bool` としてテンプレートに渡す設計がシンプルで良い

**指摘事項**:

- 1件の軽微な指摘（`featureFlagRepo` のnilガード）のみ。機能面・セキュリティ面の問題はなし
