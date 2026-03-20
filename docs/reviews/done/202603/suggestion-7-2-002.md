# コードレビュー: suggestion-7-2

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-19                           |
| 対象ブランチ               | suggestion-7-2                       |
| ベースブランチ             | suggestion-7-1c                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md     |
| 変更ファイル数             | 19 ファイル（生成ファイル 1 を含む） |
| 変更行数（実装）           | +210 行                              |
| 変更行数（テスト）         | +510 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、依存関係ルール
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ（CSRF、認証・認可）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - ログ出力
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [ ] `go/internal/handler/suggestion_apply/create.go`
- [x] `go/internal/handler/suggestion_apply/handler.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [ ] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/path.go`

### テストファイル

- [x] `go/internal/handler/suggestion_apply/create_test.go`
- [x] `go/internal/handler/suggestion_apply/main_test.go`
- [x] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `go/internal/templates/pages/suggestion/show_templ.go` （自動生成）
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-2-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_apply/create.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 認可チェックは Handler で実行
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認証・認可
- [作業計画書 ステータス変更のべき等性](/workspace/docs/plans/1_doing/suggestion.md)

**問題点・改善提案**:

- **[作業計画書#ステータス]**: ハンドラーで編集提案のステータスが「オープン」であることを検証していない。べき等性チェック（反映済みの場合は成功リダイレクト）は実装されているが、「下書き」や「クローズ」ステータスの編集提案に対する反映リクエストがそのまま UseCase に渡される。UI ではオープンステータスの場合のみボタンが表示されるが、POST リクエストを直接送信する場合に不正なステータス遷移が起きる可能性がある。

  ```go
  // 現在のコード（73-78行目）
  // べき等性: 既に反映済みの場合は何もせず成功リダイレクト
  if detailOutput.Suggestion.Status == model.SuggestionStatusApplied {
      h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_apply_success"))
      http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
      return
  }
  ```

  **修正案**:

  ```go
  // べき等性: 既に反映済みの場合は何もせず成功リダイレクト
  if detailOutput.Suggestion.Status == model.SuggestionStatusApplied {
      h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_apply_success"))
      http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
      return
  }

  // オープンステータスでなければ反映不可
  if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
      h.flashMgr.SetError(w, i18n.T(ctx, "suggestion_apply_error"))
      http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
      return
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りステータスチェックを追加する
  - [ ] UseCase 側で既にステータスチェックしているため不要（下の回答欄に根拠を記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion/show.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#レイヤー間の依存関係](/workspace/go/docs/architecture-guide.md) - Templates は Model に直接依存しない

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#レイヤー間の依存関係]**: テンプレート内で `model.SuggestionStatusOpen` を参照してステータス判定を行っている（75 行目）。アーキテクチャガイドでは「Templates は Model への直接依存は禁止」とされている。既存テンプレート（`suggestion_status_badge.templ` 等）で `model` を使用している前例はあるが、この箇所は Handler 側で `CanApply` に統合可能であるため、依存を追加せずに済む。

  ```templ
  // 現在のコード（show.templ 75行目）
  if data.CanApply && data.Suggestion.Status == model.SuggestionStatusOpen {
  ```

  **修正案**:

  Handler（`show.go`）で `canApply` の計算にステータスチェックを含め、テンプレートからは `model` の import を削除する。

  `go/internal/handler/suggestion/show.go` の修正:

  ```go
  // 反映権限をチェック（オープンステータスかつ権限がある場合のみ）
  var canApply bool
  if output.SpaceMember != nil && output.Suggestion.Status == model.SuggestionStatusOpen {
      topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
      canApply = topicPolicy.CanApplySuggestion(output.Suggestion)
  }
  ```

  `go/internal/templates/pages/suggestion/show.templ` の修正:

  ```templ
  if data.CanApply {
      <div class="px-4">
          @showApplyForm(data)
      </div>
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り Handler 側にステータスチェックを移動し、テンプレートの `model` import を削除する
  - [ ] 既存パターン（`suggestion_status_badge.templ`）に合わせて現状維持
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

タスク 7-2（編集提案反映のハンドラーと Policy）の実装として、作業計画書に記載された要件を適切にカバーしている。

**良い点**:

- 既存の `suggestion_comment` ハンドラーと一貫したパターンで実装されている
- Policy の設計が適切: スペースオーナーは `spaceID` ベース、トピック Admin は `topicID` ベースで権限判定
- べき等性（反映済みの編集提案に対する反映リクエスト）が仕様通りに実装されている
- テストが充実しており、認証・認可の各パターン（未ログイン、非メンバー、一般メンバー、オーナー、べき等性）をカバー
- CSRF トークンが適切に処理されている

**指摘事項**:

- **要修正 1 件**: Draft/Closed ステータスの編集提案に対する反映リクエストのガード
- **要修正 1 件**: テンプレートの `model` 依存を Handler 側に移動

いずれも軽微な修正であり、対応後はマージ可能。
