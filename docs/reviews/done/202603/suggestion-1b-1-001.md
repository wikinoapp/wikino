# コードレビュー: suggestion-1b-1

## レビュー情報

| 項目                       | 内容                                    |
| -------------------------- | --------------------------------------- |
| レビュー日                 | 2026-03-16                              |
| 対象ブランチ               | suggestion-1b-1                         |
| ベースブランチ             | error-pages-4-1                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/go-error-pages.md    |
| 作業計画書（実際に参照）   | docs/plans/1_doing/suggestion.md        |
| 変更ファイル数             | 5 ファイル                              |
| 変更行数（実装）           | +50 / -0 行（マイグレーション・モデル） |
| 変更行数（テスト）         | +0 / -0 行                              |

> **Note**: 指定された作業計画書 `docs/plans/1_doing/go-error-pages.md` は存在しなかったため、変更内容に対応する `docs/plans/1_doing/suggestion.md` のタスク 1b-1 を参照してレビューを行った。

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ドメインID型、モデル定義
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - DBマイグレーション、カラム定義ガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コメントのガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260316032157_create_suggestion_pages.sql`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/suggestion_page.go`

### 設定・その他

- [x] `go/db/schema.sql`（自動生成）
- [x] `docs/plans/1_doing/suggestion.md`（チェックボックス更新のみ）

## ファイルごとのレビュー結果

### `go/db/migrations/20260316032157_create_suggestion_pages.sql`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/development-guide.md#カラム定義のガイドライン](/workspace/go/docs/development-guide.md) - タイムスタンプ型
- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDの存在
- 作業計画書のタスク 1b-1 仕様との整合性

**問題点・改善提案**:

- **[設計確認] `latest_revision_id` に FK 制約がない**: `latest_revision_id UUID` は `suggestion_page_revisions` テーブルへの参照を意図しているが、FK 制約が設定されていない。これは `suggestion_page_revisions` テーブルが未作成（タスク 1c-1 で作成予定）のためと理解できるが、タスク 1c-1 のマイグレーションで `ALTER TABLE suggestion_pages ADD CONSTRAINT ... FOREIGN KEY (latest_revision_id) REFERENCES suggestion_page_revisions(id)` を追加する必要がある。

  現在のタスク 1c-1 の記述にはこの FK 追加が明記されていない。

  **修正案**:

  作業計画書のタスク 1c-1 に `suggestion_pages.latest_revision_id` への FK 制約追加を明記する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] タスク 1c-1 のマイグレーションで FK 制約を追加する（作業計画書に追記）
  - [ ] FK 制約は不要（データ整合性はアプリケーション層で担保する）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[設計確認] ユニークインデックス `(suggestion_id, page_id)` と NULL の扱い**: `page_id` は nullable（新規ページ作成時は NULL）であり、PostgreSQL のユニークインデックスでは NULL は互いに distinct として扱われる。つまり、同一 `suggestion_id` に対して `page_id = NULL` の行が複数作成可能になる。

  作業計画書の設計では「1つの編集提案に複数のページの変更をまとめる」とあるため、新規ページ作成を複数含められるのは正しい動作と思われるが、意図通りか確認したい。

  **修正案**:

  意図通りであれば修正不要。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 意図通り（1つの編集提案に複数の新規ページを含められる）
  - [ ] 想定外（新規ページは1つの編集提案に1つのみ。別途対応が必要）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  想定外でした。ページは事前に作成されているはずなので、 `suggestion_pages.page_id` はNOT NULLにしてください。
  ```

### `go/internal/model/id.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ドメインID型](/workspace/go/docs/architecture-guide.md) - ドメインID型の使用
- 作業計画書のタスク 1b-1 / 1c-1 との整合性

**問題点・改善提案**:

- **[タスク範囲] `SuggestionPageRevisionID` の先行追加**: タスク 1b-1 では `SuggestionPageID` の追加のみが記載されており、`SuggestionPageRevisionID` はタスク 1c-1 で追加予定。しかし `SuggestionPage` モデルの `LatestRevisionID` フィールドで `*SuggestionPageRevisionID` を使用するため、コンパイルを通すにはここで追加する必要がある。

  実用的な判断として問題ないが、確認のため記録する。

  **修正案**:

  修正不要。作業計画書のタスク 1c-1 から `SuggestionPageRevisionID` の追加を削除するか、「1b-1 で追加済み」と注記する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現状のまま（実用的な判断として許容）
  - [ ] 作業計画書を更新して 1c-1 から該当記述を削除
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1b-1（suggestion_pages テーブルのマイグレーションとモデル定義）の実装として、作業計画書の仕様に沿った正確な実装が行われている。

**良い点**:

- マイグレーションが既存の `suggestions` テーブルのパターン（ULID主キー、TIMESTAMP WITH TIME ZONE、FK制約、スペースIDインデックス）に一貫して従っている
- ドメインID型の使用が適切（`SuggestionPageID`, `SuggestionPageRevisionID`）
- モデルの nullable フィールドがポインタ型で正しく表現されている
- `migrate:down` でインデックスとテーブルの削除順序が正しい

**確認が必要な点**:

- `latest_revision_id` の FK 制約をタスク 1c-1 で追加する予定を明確にすること
- ユニークインデックスにおける NULL の扱いが意図通りか
- `SuggestionPageRevisionID` の先行追加について作業計画書との整合性
