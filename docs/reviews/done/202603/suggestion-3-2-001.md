# コードレビュー: suggestion-3-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-17                             |
| 対象ブランチ               | suggestion-3-2                         |
| ベースブランチ             | suggestion-3-1                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 17 ファイル（自動生成 2 ファイル含む） |
| 変更行数（実装）           | +768 / -15 行                          |
| 変更行数（テスト）         | +30 / -15 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/draft_pages.sql`
- [ ] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [ ] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/query/draft_pages.sql.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/new_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_new.go`
- [ ] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/create.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/security-guide.md#認可（Authorization）]**: `Create` ハンドラーで `output == nil` のケース（スペースメンバーでない、またはトピックが見つからない場合）は 404 を返しているが、これは認可チェックとして適切に機能している。ただし、`GetSuggestionNewUsecase` が `nil` を返すケースが「スペースが見つからない」「スペースメンバーでない」「トピックが見つからない」「非公開トピックへのアクセス権がない」の4パターンあり、いずれも `nil` で表現されている。現時点では問題ないが、将来的にエラー種別を区別してユーザーに適切なメッセージを表示する場合は UseCase の出力を分ける必要がある。現行は既存パターン（`new.go` と同じ）に従っているため問題なし。

  ただし1点、`Create` ハンドラーで `GetSuggestionNewUsecase` を呼び出す際、バリデーションエラー時のフォーム再表示用にデータを取得しているが、**`output` の取得前に `r.ParseForm()` を呼んでいる点は適切**。この順序であれば問題ない。

  **対応方針**: 問題なし。確認事項のみ。
  - [x] 了解、現状のまま

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/suggestion/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#メソッド命名規則]**: `New` ハンドラーと `renderNewForm` ヘルパーのコード分割は適切。`renderNewForm` が `Create` からも再利用されており、良い設計。

  ただし、`renderNewForm` のシグネチャで `selectedDraftIDs []string` がドメイン ID 型（`model.DraftPageID`）ではなく `string` スライスである点が気になる。これは `r.URL.Query()["draft_page_ids"]` や `r.Form["draft_page_ids"]` から直接取得した値をそのまま渡すためと理解できるが、テンプレート側で `string(dp.ID)` との比較に使われており、型の一貫性の観点から確認したい。

  **修正案**: テンプレート側の `isSelected` 関数で `string` のまま比較しているため、現状の設計で動作上は問題なし。ViewModel 層でドメイン型への変換を行わない方がシンプルなため、現状維持で良いと判断。

  **対応方針**:
  - [x] 現状のまま
  - [ ] `selectedDraftIDs` を `[]model.DraftPageID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/viewmodel/suggestion.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ビューモデル（View Model）](/workspace/go/docs/architecture-guide.md) - ViewModel設計方針

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ビューモデル]**: `DraftPageForSuggestionNew` 構造体の `title` フィールドが小文字で始まっており非公開フィールドになっている。ViewModel の `DisplayTitle` メソッド経由でのみアクセスする設計のため機能的には問題ないが、Go の慣習では構造体のフィールドはエクスポートする（大文字で始める）のが一般的。同ファイル内の `SuggestionForList` では `Title string` と公開フィールドにしており、同一ファイル内で一貫性がない。

  ただし、`title` を非公開にしている意図が「`DisplayTitle` メソッド経由でのみアクセスさせたい（空文字列の場合に『無題』を返すロジックを強制したい）」であれば、これは意図的な設計判断であり問題ない。

  **修正案**: 意図的な設計であれば現状のまま。一貫性を優先する場合は `Title` に変更。

  **対応方針**:
  - [x] `title` を `Title` に変更して一貫性を保つ
  - [ ] 現状のまま（`DisplayTitle` 経由のアクセスを強制する意図）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **NilPointerのリスク**: `NewDraftPagesForSuggestionNew` の82行目で `d.Page.Number` にアクセスしているが、`d.Page` が `nil` の場合にパニックが発生する可能性がある。`ListByMemberAndTopic` リポジトリメソッドは常に `Page` フィールドを設定しているため実運用では問題ないが、防御的なチェックがない。

  **修正案A**: `d.Page` の nil チェックを追加する

  ```go
  var pageNumber int32
  if d.Page != nil {
      pageNumber = int32(d.Page.Number)
  }
  items[i] = DraftPageForSuggestionNew{
      ID:         d.ID,
      title:      title,
      PageNumber: pageNumber,
  }
  ```

  **修正案B**: リポジトリが必ず `Page` を設定する前提で現状のまま。

  **対応方針**:
  - [x] 案Aの通り nil チェックを追加する
  - [ ] 現状のまま（リポジトリが必ず Page を設定する前提）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書（タスク 3-2）との整合性を確認した結果:

| 要件                                      | 実装状況 | 備考                                              |
| ----------------------------------------- | -------- | ------------------------------------------------- |
| `new.go` に `New` メソッドを実装          | ✅       | GET /s/{space}/topics/{topic}/suggestions/new     |
| `create.go` に `Create` メソッドを実装    | ✅       | POST /s/{space}/topics/{topic}/suggestions        |
| `get_suggestion_new.go` に UseCase を作成 | ✅       | トピック内の下書きページ一覧取得                  |
| `new.templ` にテンプレートを作成          | ✅       | チェックボックス、タイトル・概要入力              |
| `cmd/server/main.go` にルーティング登録   | ✅       | 認証済みユーザー専用ルートに登録                  |
| 翻訳ファイルにメッセージ追加              | ✅       | ja.toml, en.toml に 11 メッセージ追加             |
| テスト追加                                | ⚠️       | 既存テストの更新のみ。New/Create のテストが未追加 |

### 設計との乖離

- **テスト不足**: 作業計画書では「テスト 2 ファイル、約 200 行」を想定しているが、既存テスト（`index_test.go`）のリファクタリング（`setupHandler` ヘルパーの更新）のみが含まれている。`New` ハンドラーと `Create` ハンドラーのテストが未追加。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

編集提案作成のハンドラー（`New`, `Create`）、UseCase（`GetSuggestionNewUsecase`）、ViewModel（`DraftPageForSuggestionNew`）、テンプレート（`new.templ`）、ルーティング登録、翻訳メッセージの追加が適切に実装されている。アーキテクチャガイドラインに従った 3 層アーキテクチャの構成、ハンドラーガイドの命名規則、セキュリティガイドラインの CSRF トークン対応、国際化ガイドの翻訳キー命名規則など、各ガイドラインへの準拠が確認できた。

`renderNewForm` ヘルパーの共有設計（`New` と `Create` で再利用）や、`ListByMemberAndTopic` クエリでの `suggestion_page_id IS NULL` フィルタ（通常の下書きのみ表示）など、仕様要件を正確に反映した実装がなされている。

ただし、**`New` と `Create` ハンドラーのテストが未追加**であり、作業計画書で想定されているテストカバレッジに達していない。正常系（編集提案の作成成功、フォーム表示）と異常系（バリデーションエラー、権限なし）のテストを追加する必要がある。
