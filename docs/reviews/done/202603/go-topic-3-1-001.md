# コードレビュー: go-topic-3-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-08                                    |
| 対象ブランチ               | go-topic-3-1                                  |
| ベースブランチ             | go-topic                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md |
| 変更ファイル数             | 9 ファイル                                    |
| 変更行数（実装）           | +197 / -0 行（templ 生成ファイル除く）        |
| 変更行数（テスト）         | +87 / -0 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/viewmodel/topic.go`
- [ ] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/templates/pages/topic/show_templ.go`（自動生成）
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/viewmodel/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/topic-show-go-migration.md`

## ファイルごとのレビュー結果

### `go/internal/templates/pages/topic/show.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートデータ構造体と ViewModel の関係
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - レイヤー間の依存関係

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレートデータ構造体とViewModelの関係]**: `ShowData` 構造体に `SpaceIdentifier model.SpaceIdentifier` フィールドがあるが、`Space viewmodel.Space` フィールドの `Identifier` と重複している

  `viewmodel.Space` は `Identifier model.SpaceIdentifier` フィールドを持っており、`data.Space.Identifier` でアクセスできる。既存の `page/edit.templ` や `page_move/new.templ` では `data.Space.Identifier.String()` のパターンを使っており、別途 `SpaceIdentifier` フィールドを持っていない。

  実際に `show.templ` 内でもパンくずリスト（40 行目）では `data.Space.Identifier.String()` を使っているが、他の箇所では `data.SpaceIdentifier` を使っており、テンプレート内でも一貫していない。

  ```go
  // 現在のコード
  type ShowData struct {
      Topic           viewmodel.TopicForShow
      Space           viewmodel.Space
      PinnedPages     []viewmodel.CardLinkPage
      Pages           []viewmodel.CardLinkPage
      Pagination      viewmodel.Pagination
      SpaceIdentifier model.SpaceIdentifier  // Spaceと重複
  }
  ```

  **修正案**:

  `SpaceIdentifier` フィールドを削除し、テンプレート内の参照を `data.Space.Identifier` に統一する。

  ```go
  // 修正後のコード
  type ShowData struct {
      Topic       viewmodel.TopicForShow
      Space       viewmodel.Space
      PinnedPages []viewmodel.CardLinkPage
      Pages       []viewmodel.CardLinkPage
      Pagination  viewmodel.Pagination
  }
  ```

  テンプレート内の変更:

  ```
  // paginationPath メソッド
  - return fmt.Sprintf("/s/%s/topics/%d?page=%d", d.SpaceIdentifier, ...)
  + return fmt.Sprintf("/s/%s/topics/%d?page=%d", d.Space.Identifier, ...)

  // CardLinkPage 呼び出し
  - @components.CardLinkPage(page, data.SpaceIdentifier)
  + @components.CardLinkPage(page, data.Space.Identifier)

  // URL生成
  - templates.NewPagePath(data.SpaceIdentifier.String(), ...)
  + templates.NewPagePath(data.Space.Identifier.String(), ...)

  - templates.TopicSettingsPath(data.SpaceIdentifier.String(), ...)
  + templates.TopicSettingsPath(data.Space.Identifier.String(), ...)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `SpaceIdentifier` フィールドを削除し、`data.Space.Identifier` に統一する
  - [ ] 現状のまま（理由を回答欄に記入）
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

タスク 3-1（トピック詳細画面用の ViewModel とテンプレート作成）の実装は作業計画書の要件をおおむね満たしている。`TopicForShow` ViewModel、`show.templ` テンプレート、i18n 翻訳キーの追加は設計通りに実装されている。

良かった点:

- `TopicForShow` ViewModel の設計が簡潔で、権限情報（`CanUpdate`、`CanCreatePage`）を適切に分離している
- テストが正常系・異常系ともにテーブル駆動テストで書かれており、テストガイドに準拠している
- i18n 翻訳が日英両方で追加されており、命名規則（`topic_show_*`）もガイドラインに沿っている
- 既存コンポーネント（`TopNav`、`CardLinkPage`、`PaginationNav`、`MainTitle`）を適切に再利用している
- 空状態のコンポーネントも作成されており、ページ作成権限に応じた表示分岐がある

問題点は `ShowData` の `SpaceIdentifier` フィールドの冗長性のみであり、軽微な指摘である。
