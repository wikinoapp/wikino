# コードレビュー: go-topic-3-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-08                                    |
| 対象ブランチ               | go-topic-3-1                                  |
| ベースブランチ             | go-topic                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md |
| 変更ファイル数             | 10 ファイル（うち自動生成 1、ドキュメント 2） |
| 変更行数（実装）           | +216 行                                       |
| 変更行数（テスト）         | +87 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/viewmodel/topic.go`
- [x] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/viewmodel/topic_test.go`

### 設定・その他

- [x] `go/internal/templates/pages/topic/show_templ.go`（自動生成）
- [x] `docs/plans/1_doing/topic-show-go-migration.md`（タスクチェック更新）
- [x] `docs/reviews/done/202603/go-topic-3-1-001.md`（前回レビュー）

## ファイルごとのレビュー結果

すべてのファイルが問題なく、ガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書のタスク 3-1 の要件と実装を照合:

| 要件                                     | 状態 |
| ---------------------------------------- | ---- |
| `TopicForShow` ViewModel の追加          | ✅   |
| トピック詳細画面テンプレートの作成       | ✅   |
| 既存の `CardLinkPage` コンポーネント活用 | ✅   |
| 既存の `TopNav` コンポーネント活用       | ✅   |
| 空状態コンポーネントの作成               | ✅   |
| i18n 翻訳キーの追加                      | ✅   |

**設計からの変更点**（意図的な改善）:

- 作業計画書では `TopicForShow` に `Space SpaceForBreadcrumb` フィールドを含める設計だったが、`ShowData` で `Space` を別フィールドとして渡す方式に変更されている（コミット 09e6faf3 で冗長なフィールドを削除）。これは `EditPageData` など他のテンプレートデータ構造と一貫したパターンであり、適切な設計改善。
- i18n キー名に `topic_show_` プレフィックスを使用（計画書では `topic_` プレフィックス）。i18n ガイドの命名規則 `{page_name}_{detail}` に従っており、こちらの方が適切。
- 計画書に明記されていなかった `topic_show_new_page`、`topic_show_settings` の翻訳キーも追加されている。テンプレートのアクションボタンに必要なキーであり、妥当。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-1（トピック詳細画面用の ViewModel とテンプレート作成）の実装として、すべての要件が正しく実装されている。

**良い点**:

- templ テンプレートガイドに従い、構造体ベースのデータ受け渡しパターンを正しく使用している（`ShowData` 構造体、`ctx` を明示的に渡さない）
- 既存コンポーネント（`TopNav`、`CardLinkPage`、`PaginationNav`、`MainTitle`）の再利用が適切で、パンくずリストの構成も `page/edit.templ` と同じパターン
- `TopicForShow` ViewModel の設計がアーキテクチャガイドに準拠しており、画面の要件に応じた適切なフィールド構成
- テストがテーブル駆動テストで網羅的に書かれており、公開/非公開トピック、権限の有無のケースをカバー
- i18n の翻訳キーが日英両方で追加されており、命名規則も `{page_name}_{detail}` パターンに従っている
- `paginationPath` メソッドを `ShowData` に定義することでテンプレート内のロジックをシンプルに保っている
