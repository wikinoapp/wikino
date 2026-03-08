# コードレビュー: go-fix

## レビュー情報

| 項目                   | 内容                                                         |
| ---------------------- | ------------------------------------------------------------ |
| レビュー日             | 2026-02-08                                                   |
| 対象ブランチ           | go-fix                                                       |
| ベースブランチ         | go                                                           |
| 設計書（指定があれば） | なし                                                         |
| 変更ファイル数         | 85 ファイル                                                  |
| 変更行数（実装）       | +3071 / -1385 行（テンプレート生成ファイル、Dockerfile含む） |
| 変更行数（テスト）     | テスト変更は主にsign_inハンドラーテストの依存追加            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/new.go`
- [x] `go/internal/middleware/csrf.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_password_reset.go`
- [x] `go/internal/templates/helper.go`
- [x] `go/internal/templates/emails/password_reset/data.go`
- [x] `go/internal/templates/emails/password_reset/ja_html.templ`
- [x] `go/internal/templates/emails/password_reset/en_html.templ`
- [x] `go/internal/templates/emails/password_reset/ja_text.templ`
- [x] `go/internal/templates/emails/password_reset/en_text.templ`
- [x] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/components/footer.templ`
- [x] `go/internal/templates/components/form_errors.templ`
- [x] `go/internal/templates/components/head.templ`
- [x] `go/internal/templates/layouts/plain.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/welcome/show.templ`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/password/reset_sent.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/web/style.css`
- [x] `rails/app/controllers/controller_concerns/authenticatable.rb`
- [x] `rails/config/locales/messages.ja.yml`
- [x] `rails/db/queue_structure.sql`

### テストファイル

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/handler/welcome/show_test.go`
- [x] `go/internal/middleware/csrf_test.go`

### 設定・その他

- [x] `CLAUDE.md`
- [x] `Dockerfile.dev`
- [x] `docker-compose.yml`
- [x] `go/Dockerfile.dev`（削除）
- [x] `rails/Dockerfile.dev`（削除）
- [x] `go/db/schema.sql`
- [x] `go/mise.toml`
- [x] `go/package.json`
- [x] `go/pnpm-lock.yaml`
- [x] `mise.toml`
- [x] `rails/CLAUDE.md`
- [x] `rails/mise.toml`
- [x] `rails/spec/requests/welcome/show_spec.rb`（削除）
- [x] `rails/spec/support/capybara.rb`
- [x] `rails/spec/system/welcome/show_spec.rb`（削除）
- [x] `docs/reviews/template.md`
- [x] `docs/reviews/done/202602/go-fix-202602080706.md`
- [x] `docs/designs/3_done/202602/unified-dev-container.md`

### 自動生成ファイル（レビュー対象外）

- [x] `go/internal/templates/components/flash_templ.go`
- [x] `go/internal/templates/components/footer_templ.go`
- [x] `go/internal/templates/components/form_errors_templ.go`
- [x] `go/internal/templates/components/head_templ.go`
- [x] `go/internal/templates/emails/password_reset/en_html_templ.go`
- [x] `go/internal/templates/emails/password_reset/en_text_templ.go`
- [x] `go/internal/templates/emails/password_reset/ja_html_templ.go`
- [x] `go/internal/templates/emails/password_reset/ja_text_templ.go`
- [x] `go/internal/templates/layouts/plain_templ.go`
- [x] `go/internal/templates/pages/account/new_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/edit_templ.go`
- [x] `go/internal/templates/pages/password/edit_templ.go`
- [x] `go/internal/templates/pages/password/reset_templ.go`
- [x] `go/internal/templates/pages/password/reset_sent_templ.go`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new_templ.go`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`
- [x] `go/internal/templates/pages/welcome/show_templ.go`

### 静的ファイル（レビュー対象外）

- [x] `go/static/images/welcome/feature_1.png`
- [x] `go/static/images/welcome/feature_2.png`
- [x] `go/static/images/welcome/feature_3.png`
- [x] `go/static/images/welcome/feature_4.png`
- [x] `go/static/images/welcome/hero.png`
- [x] `rails/app/assets/images/welcome/image_1.png`（削除）
- [x] `rails/app/assets/images/welcome/image_2.png`（削除）
- [x] `rails/app/assets/images/welcome/image_3.png`（削除）
- [x] `rails/app/assets/images/welcome/image_4.png`（削除）

## ファイルごとのレビュー結果

### `go/internal/templates/components/head.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド

**問題点・改善提案**:

- **コメントアウトされたコードが残っている**: ダークモード検出のJavaScriptコードがコメントアウトされて残っている。`TODO` コメントは付いているが、コメントアウトされたコード自体が出力HTMLに含まれてしまう

  ```templ
  // TODO: ダークモード対応時にコメントアウトを解除する
  <script>
      // (() => {
      // 	try {
      // 		if (matchMedia("(prefers-color-scheme: dark)").matches) {
      // 			document.documentElement.classList.add("dark");
      // 		}
      // 	} catch (_) {}
      // })();
  </script>
  ```

  **修正案**: 空の`<script>`タグを出力しないよう、`<script>`タグごと削除するか、templ側でコメント化する

  ```templ
  // TODO: ダークモード対応時に以下のスクリプトを追加する
  // <script>
  //     (() => { ... })();
  // </script>
  ```

  **対応方針**:
  - [ ] `<script>`タグごと削除してtemplコメントのみ残す
  - [ ] 現状のまま維持する（ダークモード対応が近いため）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  script要素ごとコメントアウトして、ソースコードにscript要素とコメントアウトしたJSが残らないようにしてください
  ```

### `go/internal/worker/send_password_reset.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ワーカー](/workspace/go/CLAUDE.md) - Worker の実装パターン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **`cfg`フィールドが未使用**: `SendPasswordResetWorker`構造体に`cfg *config.Config`フィールドがあるが、`Work`メソッドでも`renderEmailTemplates`メソッドでも使用されていない。既存の`SendEmailConfirmationWorker`にも同様のフィールドがあり一貫性はあるが、現時点で不要なフィールドは削除すべきか確認が必要

  **対応方針**:
  - [x] `SendPasswordResetWorker`と`SendEmailConfirmationWorker`の両方から`cfg`フィールドを削除する
  - [ ] 現状のまま維持する（将来的に使用する可能性があるため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/sign_in/new.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **パスワードリセットリンクのパス変更**: `/password_reset` から `/password/reset` に変更されている。この変更がルーティング定義と一致しているか確認が必要

  ```templ
  // 変更前
  <a href="/password_reset" class="ml-auto text-sm link" tabindex="4">
  // 変更後
  <a href="/password/reset" class="ml-auto text-sm link" tabindex="4">
  ```

  **対応方針**:
  - [x] ルーティング定義と一致していることを確認済み
  - [ ] ルーティング定義の修正が必要
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 総合評価

**評価**: Comment

**総評**:

全体的に品質の高い変更です。主な変更内容は以下の通りです:

**良かった点**:

1. **開発コンテナの統合**: Go版とRails版のDockerfileを1つに統合し、mise.tomlでツールバージョンを管理する方針は開発体験の向上に貢献しています
2. **ミドルウェア順序の修正**: リバースプロキシミドルウェアをMethod Override/CSRFミドルウェアより前に配置することで、リクエストボディ消費の問題を正しく解決しています。コメントによる説明も適切です
3. **CSRFスキップパス**: Rails版からのログアウトリクエスト対応として完全一致マッチングを使用しており、セキュリティ的に適切です
4. **パスワードリセットWorker**: 既存の`SendEmailConfirmationWorker`と一貫したパターンで実装されています
5. **二要素認証チェック**: サインイン処理に二要素認証チェックを適切に追加しています
6. **UI改善**: テンプレートのカード化、フッター改善、アイコン名のリネーム（`-regular`サフィックス追加）は一貫性があります
7. **翻訳メッセージの品質向上**: 文末の句点統一、文言の改善が適切に行われています
8. **テストの更新**: 新しい依存関係（`userTwoFactorAuthRepo`）の追加がすべてのテストに反映されています

**軽微な指摘**:

1. `head.templ`でコメントアウトされたJavaScriptがHTMLに出力される点は対応を確認したい
2. `SendPasswordResetWorker`の未使用`cfg`フィールドについて確認が必要
3. パスワードリセットリンクのパス変更がルーティングと一致しているか確認が必要
