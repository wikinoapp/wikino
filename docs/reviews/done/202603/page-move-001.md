# コードレビュー: page-move

## レビュー情報

| 項目                       | 内容                            |
| -------------------------- | ------------------------------- |
| レビュー日                 | 2026-03-05                      |
| 対象ブランチ               | page-move                       |
| ベースブランチ             | develop                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-move.md |
| 変更ファイル数             | 32 ファイル                     |
| 変更行数（実装）           | +1224 / -5 行                   |
| 変更行数（テスト）         | +715 / -0 行                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/page_move/handler.go`
- [ ] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/handler/page_move/validator.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`
- [ ] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/page_move/new_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/move_page.go`
- [x] `go/internal/viewmodel/topic.go`
- [x] `rails/app/components/dropdowns/page_actions_component.html.erb`
- [x] `rails/config/locales/verbs.en.yml`
- [x] `rails/config/locales/verbs.ja.yml`
- [x] `rails/config/routes.rb`

### テストファイル

- [x] `go/internal/handler/page_move/main_test.go`
- [x] `go/internal/handler/page_move/new_test.go`
- [x] `go/internal/handler/page_move/validator_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/usecase/move_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/plans/1_doing/page-move.md`

## ファイルごとのレビュー結果

### `go/internal/templates/pages/page_move/new.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートデータ構造体と ViewModel の関係

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレートデータ構造体とViewModelの関係]**: `MovePageData` に `PageTitle string` と `PageNumber int32` がプリミティブ値として展開されている

  テンプレートガイドには以下の記述がある:

  > モデルのフィールドを個別のプリミティブ値として展開せず、ViewModel を構成要素として使用する。
  > ❌ モデルのフィールドを個別に並べない: `Title string`, `Body string`, `PageNumber int32`

  ```go
  // 現在のコード
  type MovePageData struct {
      CSRFToken       string
      FormErrors      *session.FormErrors
      PageTitle       string           // プリミティブ
      PageNumber      int32            // プリミティブ
      Space           viewmodel.Space
      CurrentTopic    viewmodel.Topic
      AvailableTopics []viewmodel.TopicForSelect
  }
  ```

  **修正案**:

  `viewmodel.Page` を使用するか、ページ移動画面用の ViewModel（例: `viewmodel.PageForMove`）を作成し、ViewModel として渡す。

  ```go
  type MovePageData struct {
      CSRFToken       string
      FormErrors      *session.FormErrors
      Page            viewmodel.PageForMove  // ViewModelを使用
      Space           viewmodel.Space
      CurrentTopic    viewmodel.Topic
      AvailableTopics []viewmodel.TopicForSelect
  }
  ```

  ただし、既存の `viewmodel.Page` が編集画面用で多くのフィールドを持っている場合、ページ移動画面で必要なのは `Title` と `Number` のみなので、専用の ViewModel を作成するのが適切。テンプレートガイドの「表示項目が同じであれば再利用しても構いません」にも留意。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] ページ移動用の ViewModel（`viewmodel.PageForMove`）を作成する
  - [ ] 現状のプリミティブフィールドのまま維持する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page_move/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - レイヤー間の依存関係

**問題点・改善提案**:

- **N+1 クエリの可能性**: `availableTopicsForMove` メソッド（186-226行目）で、トピック一覧をループして各トピックごとに `topicMemberRepo.FindBySpaceMemberAndTopic` を呼び出している。トピック数が多い場合にN+1クエリが発生する。

  ```go
  for _, t := range topics {
      // ...
      tm, err := h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, t.ID)
      // ...
  }
  ```

  **修正案**:

  スペースオーナーの場合は全トピックに `CanCreatePage` が真なので、`FindBySpaceMemberAndTopic` の呼び出しを省略し、現在のトピックの除外のみ行う。スペースオーナーでない場合は `ListJoinedBySpaceMember` が所属トピックのみを返すため、メンバーとしての `CanCreatePage` は常に真になるはずで、同様にクエリ不要。

  つまり、権限チェックはリスト取得の段階で暗黙的に満たされており、追加のクエリは不要な可能性がある。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] スペースオーナーの場合はループ内の権限チェックを省略する
  - [ ] 一括取得クエリ（`FindBySpaceMemberAndTopics`）を追加する
  - [ ] 現状のまま（トピック数は少ないため問題ない、理由を回答欄に記入）
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

ページ移動機能が作業計画書の設計通りに実装されている。アーキテクチャガイドライン（3層アーキテクチャ、ハンドラーガイドの命名規則、セキュリティガイド）に概ね準拠しており、品質は高い。

良い点:

- Policy パターンで権限チェックが適切に実装されている（`CanCreatePage` の追加と各ポリシーの実装）
- バリデーションが形式チェックと状態チェックを段階的に実行しており、ガイドラインに沿っている
- セキュリティ面で `space_id` によるクエリスコープが SQL レベルで適用されている（`MovePageToTopic` クエリ）
- CSRF トークンが適切にフォームに含まれている
- テストが正常系・異常系ともに網羅的に記述されている
- I18n 対応が ja/en ともに追加されている

軽微な指摘:

- `MovePageData` のプリミティブフィールドがテンプレートガイドの推奨パターンと異なる（影響は小さい）
- `availableTopicsForMove` のN+1クエリは現時点で問題にならない可能性が高いが、認識しておく価値がある
