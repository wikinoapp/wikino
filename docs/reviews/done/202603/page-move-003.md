# コードレビュー: page-move

## レビュー情報

| 項目                       | 内容                            |
| -------------------------- | ------------------------------- |
| レビュー日                 | 2026-03-05                      |
| 対象ブランチ               | page-move                       |
| ベースブランチ             | develop                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-move.md |
| 変更ファイル数             | 44 ファイル                     |
| 変更行数（実装）           | +1,097 / -10 行                 |
| 変更行数（テスト）         | +962 / -0 行                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版開発ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/page_move/handler.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/handler/page_move/validator.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/page_move/new_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/usecase/move_page.go`
- [x] `go/internal/viewmodel/page.go`
- [x] `go/internal/viewmodel/topic.go`
- [x] `go/internal/viewmodel/sidebar.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/components/footer.templ`
- [x] `go/internal/templates/components/footer_templ.go`
- [x] `go/internal/templates/components/main_title.templ`
- [x] `go/internal/templates/components/main_title_templ.go`
- [x] `rails/app/components/dropdowns/page_actions_component.html.erb`
- [x] `rails/config/locales/verbs.en.yml`
- [x] `rails/config/locales/verbs.ja.yml`
- [x] `rails/config/routes.rb`

### テストファイル

- [x] `go/internal/handler/page_move/main_test.go`
- [x] `go/internal/handler/page_move/new_test.go`
- [x] `go/internal/handler/page_move/create_test.go`
- [x] `go/internal/handler/page_move/validator_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/usecase/move_page_test.go`

### 設定・その他

- [x] `Dockerfile.dev`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/plans/1_doing/page-move.md`
- [x] `docs/reviews/page-move-001.md`
- [x] `docs/reviews/page-move-002.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page_move/handler.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md#依存性注入のガイドライン](/workspace/go/docs/handler-guide.md) - Handler構造体のフィールド数

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#依存性注入のガイドライン]**: Handler構造体のフィールドが9個で、ガイドラインの「8個を超えたらリソース分割を検討」に該当する

  ```go
  type Handler struct {
      cfg             *config.Config
      flashMgr        *session.FlashManager
      spaceRepo       *repository.SpaceRepository
      spaceMemberRepo *repository.SpaceMemberRepository
      pageRepo        *repository.PageRepository
      draftPageRepo   *repository.DraftPageRepository    // sidebarContentのみで使用
      topicRepo       *repository.TopicRepository
      topicMemberRepo *repository.TopicMemberRepository
      movePageUC      *usecase.MovePageUsecase
  }
  ```

  ただし、`draftPageRepo` は `sidebarContent` ヘルパーのみで使用されており、既存の `page/handler.go` でも同じパターン（9フィールド）を採用している。プロジェクト全体の一貫性を考えると、現状は許容範囲とも言える。

  **修正案**:

  サイドバーコンテンツの取得を別の構造体やヘルパーに切り出す（例: `SidebarHelper` を Handler に注入する）ことでフィールド数を削減可能。ただし、既存の `page/handler.go` と同じパターンなので、変更するなら両方を一度に変更すべき。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 将来的にサイドバーヘルパーの切り出しを検討する（現時点では対応不要）
  - [x] 既存の `page/handler.go` と合わせて `SidebarHelper` を切り出す
  - [ ] 現状のまま（既存パターンとの一貫性を優先）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page_move/create.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション結果のハンドリング

**問題点・改善提案**:

- **バリデーション結果に `Err` フィールドがなく、システムエラーと `FormErrors` が混在している**: `CreateValidatorResult` は `DestTopic` と `FormErrors` のみを持つが、validator.go の実装では DB エラーが発生した場合に `formErrors.AddGlobal(i18n.T(ctx, "validation_system_error"))` でフォームエラーとして返している（71行目、88行目、102行目）。

  バリデーションガイドの例では `Err error` フィールドを持つ Result 構造体が推奨されている。現在の実装では、DB 接続エラー等のシステムエラーがフォーム再表示のフィールドエラーとして表示され、ユーザーに「もう一度やり直してください」的なメッセージが出る。

  ただし、これは動作上問題があるわけではなく、ユーザー体験の観点での改善案。DB エラーの場合に 500 を返すか、フォーム再表示でグローバルエラーを表示するかは設計判断。

  **修正案**:

  `CreateValidatorResult` に `Err error` フィールドを追加し、DB エラー時はハンドラー側で 500 レスポンスを返す。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `Err error` フィールドを追加し、システムエラーとフォームエラーを分離する
  - [ ] 現状のまま（フォーム再表示でグローバルエラーを表示するのが意図的）
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

過去2回のレビュー（page-move-001, page-move-002）で指摘された主要な問題が適切に修正されている:

- `MovePageData` のプリミティブフィールドが `viewmodel.PageForMove` に改善された
- `availableTopicsForMove` のN+1クエリが解消され、権限チェックがリスト取得の段階で暗黙的に行われるよう最適化された
- `strconv.ParseInt` によるオーバーフロー安全なパースに修正された
- `PageNamePageMove` 定数が追加されサイドバーの表示が適切に区別された
- `create_test.go` が追加され、正常系（リダイレクト確認）、バリデーションエラー（422レスポンス確認）、未認証（リダイレクト確認）のテストが網羅された

**良い点**:

- 3層アーキテクチャの依存関係ルールが正しく守られている
- SQLクエリの `MovePageToTopic` に `space_id` スコープが含まれ、セキュリティガイドラインに準拠
- TopicPolicy の各実装（owner/admin/member/guest）で `CanCreatePage` が作業計画書の権限設計通りに実装されている
- バリデーションが形式チェック→状態チェックの順序で段階的に実行され、早期リターンパターンが活用されている
- I18n対応が日英両方で完備され、命名規則 `page_move_{detail}` に統一されている
- templ テンプレートが構造体ベースのデータ受け渡しパターンに従い、`viewmodel.PageForMove` を使用している
- テストが正常系・異常系ともに網羅的（New, Create, Validator, UseCase, Policy）
- UseCase が WithTx パターンを正しく使用してトランザクション管理している
- Rails 側の変更が最小限で適切（ドロップダウンへの移動リンク追加、ルーティング、翻訳）
- 作業計画書の全タスク（フェーズ1-3）が完了し、設計通りに実装されている

**軽微な指摘**:

- Handler構造体のフィールド数が9個でガイドラインの閾値（8個）を1つ超えているが、既存パターン（`page/handler.go`）との一貫性があり、実質的に問題なし
- バリデーション結果に `Err` フィールドがなくシステムエラーがフォームエラーに混在するが、動作上の問題はなし
