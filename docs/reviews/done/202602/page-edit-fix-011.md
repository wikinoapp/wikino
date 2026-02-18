# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                                  |
| ---------------------------- | ----------------------------------------------------- |
| レビュー日                   | 2026-02-18                                            |
| 対象ブランチ                 | page-edit-fix                                         |
| ベースブランチ               | page-edit                                             |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md          |
| 変更ファイル数               | 31 ファイル（自動生成・ドキュメント除く）              |
| 変更行数（実装）             | +614 / -44 行（テスト・自動生成・ドキュメント除く）   |
| 変更行数（テスト）           | +588 / -0 行                                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/topics.sql`
- [ ] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/welcome/show.go`
- [ ] `go/internal/i18n/locales/en.toml`
- [ ] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/components/top_nav.templ`
- [ ] `go/internal/templates/helper.go`
- [x] `go/internal/templates/icons_custom.go`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [ ] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/password/reset_sent.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/welcome/show.templ`
- [ ] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/space.go`

### テストファイル

- [ ] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/page/main_test.go`
- [x] `go/internal/viewmodel/space_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/3_done/202602/icon-file-separation.md`

## ファイルごとのレビュー結果

### `go/internal/i18n/locales/ja.toml` / `go/internal/i18n/locales/en.toml`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 命名規則、description必須

**問題点・改善提案**:

- **[@go/docs/i18n-guide.md#descriptionを必ず記述]**: `page_edit_*` の7つのキーに `description` フィールドがない

  同じ差分内で追加された `top_nav_breadcrumb_label`、`breadcrumb_home`、`space_layout_breadcrumb_label` には `description` が記述されており、一貫性がない。

  **修正案**:

  ```toml
  # ja.toml
  [page_edit_title]
  description = "ページ編集画面のタイトル"
  other = "ページを編集"

  [page_edit_title_label]
  description = "ページタイトル入力フィールドのラベル"
  other = "タイトル"
  # ... 他のpage_edit_*キーも同様
  ```

  **対応方針**:

  - [x] 全ての `page_edit_*` キーに `description` を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/i18n-guide.md#命名規則]**: `space_layout_breadcrumb_label` キーが定義されているが、コードベース内で使用されていない

  `grep` で検索した結果、`space_layout_breadcrumb_label` はどのGoファイルにもテンプレートファイルにも参照されていない。未使用のi18nキーは混乱の元になるため、使用予定がなければ削除すべき。

  **修正案**: 未使用の `space_layout_breadcrumb_label` を `ja.toml` と `en.toml` の両方から削除する。

  **対応方針**:

  - [x] 未使用キーを削除する
  - [ ] 今後のスペースレイアウトで使用するため残す（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page/edit.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#HTTPハンドラー](/workspace/go/CLAUDE.md) - ハンドラーの命名・構造
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認証・認可、スペースIDスコープ
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール

**問題点・改善提案**:

- **トピックnilケースのサイレント処理**: 90-104行目で `topic` が `nil` の場合、パンくずリストのトピック部分が空になるだけで、エラーログも出力されない。ページは必ずトピックに属するため、`topic == nil` はデータ不整合を示すはず。

  ```go
  // 現在のコード (line 97-104)
  var topicName string
  var topicPath templates.Path
  var topicIconName templates.IconName
  if topic != nil {
      topicName = topic.Name
      topicPath = templates.TopicPath(space.Identifier, topic.Number)
      topicIconName = templates.TopicVisibilityIconName(topic.Visibility)
  }
  ```

  **修正案**:

  ```go
  if topic == nil {
      slog.ErrorContext(ctx, "ページのトピックが見つかりません", "page_id", pg.ID, "topic_id", pg.TopicID)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
  }

  topicName := topic.Name
  topicPath := templates.TopicPath(space.Identifier, topic.Number)
  topicIconName := templates.TopicVisibilityIconName(topic.Visibility)
  ```

  **対応方針**:

  - [x] 修正案の通り、nilの場合は500エラーを返す
  - [ ] 警告ログを出しつつ、空のパンくずで表示を続行する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/helper.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール（Templates → ViewModel → Model）

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#依存関係]**: `TopicVisibilityIconName` 関数がtemplatesパッケージから `model` パッケージに直接依存している

  アーキテクチャガイドでは「Templates は ViewModel に依存できますが、Model に直接依存することは禁止」と明記されている。現在この関数はハンドラーから呼び出されており、テンプレート内では使用されていないが、`templates` パッケージに配置されているため `model` への依存が生じている。

  ```go
  // helper.go (line 40-45)
  func TopicVisibilityIconName(v model.TopicVisibility) IconName {
      if v == model.TopicVisibilityPublic {
          return "globe-regular"
      }
      return "lock-regular"
  }
  ```

  **修正案**: `viewmodel` パッケージに移動する。`viewmodel` は `model` に依存可能であり、ハンドラーは `viewmodel` に依存可能。

  ```go
  // internal/viewmodel/topic.go
  func TopicVisibilityIconName(v model.TopicVisibility) templates.IconName {
      if v == model.TopicVisibilityPublic {
          return "globe-regular"
      }
      return "lock-regular"
  }
  ```

  **対応方針**:

  - [ ] `viewmodel` パッケージに移動する
  - [ ] 現状のまま（templates層の`model`依存を軽微な例外として許容する）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  modelからデータベースにアクセスされることは無いので、
  templates層がmodelに依存しても良いのではと思い始めていますが、
  それはそれでじゃあmodelとviewmodelはどう使い分けるのか？といった話にもなりそうで、どうしようかなという感じです。
  どう思いますか？
  ```

### `go/internal/templates/pages/page/edit.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策

**問題点・改善提案**:

- **URL構築が`fmt.Sprintf`で散在している**: 46行目と95行目でURLを `fmt.Sprintf` で直接構築している。`path.go` に `PagePath` や `GoPageEditPath` などのヘルパーが定義されていないため、URLパターンが分散している。

  ```templ
  // line 46: Go版へのフォーム送信（/goプレフィックス付き）
  action={ templ.SafeURL(fmt.Sprintf("/go/s/%s/pages/%d", data.SpaceIdentifier, data.PageNumber)) }

  // line 95: Rails版のページ表示へのキャンセルリンク（/goプレフィックスなし）
  href={ templ.SafeURL(fmt.Sprintf("/s/%s/pages/%d", data.SpaceIdentifier, data.PageNumber)) }
  ```

  **修正案**: `path.go` に以下のヘルパーを追加し、テンプレート内で使用する。

  ```go
  // internal/templates/path.go
  func PagePath(spaceIdentifier string, pageNumber int32) Path {
      return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
  }

  func GoPagePath(spaceIdentifier string, pageNumber int32) Path {
      return Path(fmt.Sprintf("/go/s/%s/pages/%d", spaceIdentifier, pageNumber))
  }
  ```

  **対応方針**:

  - [x] `path.go` にヘルパー関数を追加し、テンプレートで使用する
  - [ ] 現時点では `fmt.Sprintf` のまま（今後 update.go 等でも使う際にまとめて対応）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/path.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#既存コードとの一貫性](/workspace/go/CLAUDE.md) - 一貫性

**問題点・改善提案**:

- **`PagePath` ヘルパーの欠如**: `SpacePath`、`TopicPath`、`HomePath` が定義されているが、`PagePath` が未定義。`edit.templ` で `fmt.Sprintf` を直接使用している（上記 `edit.templ` の問題点と同じ）。

  上記の `edit.templ` の対応方針に含まれるため、ここでは重複して記載しない。

### `go/internal/handler/page/edit_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テストのベストプラクティス

**問題点・改善提案**:

- **`TestEdit` でロケールが明示的に設定されていない**: `TestEdit` は `Accept-Language: ja` ヘッダーを設定しているが、i18nミドルウェアを経由しないため、ロケールがコンテキストに正しくセットされない可能性がある。`TestEdit_EnglishLocale` では `i18n.SetLocale(ctx, i18n.LangEn)` を明示的に呼び出しているのに対し、`TestEdit` では呼び出していない。これにより、日本語テキストの検証（148-155行目）が信頼できない。

  **修正案**:

  ```go
  // TestEdit 内で明示的にロケールを設定
  ctx := context.Background()
  ctx = i18n.SetLocale(ctx, i18n.LangJa)
  req := httptest.NewRequest("GET", url, nil).WithContext(ctx)
  ```

  **対応方針**:

  - [x] `i18n.SetLocale` を明示的に設定する
  - [ ] デフォルトロケールがjaなので不要（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **`TestEdit_AutofocusTitle` がautofocus属性を検証していない**: テスト名は「AutofocusTitle」だが、実際のアサーションでは `id="page_title"` の存在確認のみで、`autofocus` 属性の有無を検証していない。

  **修正案**: autofocus属性の存在を検証するアサーションを追加する。

  ```go
  // autofocus属性の検証を追加
  if !strings.Contains(body, "autofocus") {
      t.Error("autofocus attribute not found in page_title input")
  }
  ```

  **対応方針**:

  - [x] autofocus属性の検証を追加する
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

全体として高品質な実装。3層アーキテクチャの依存関係ルールが守られており、セキュリティ対策（CSRF、認証・認可、スペースIDスコープ）も適切。テストカバレッジは正常系・異常系の両方を網羅しており、TestMainパターン、`t.Parallel()`、ビルダーパターンなどのプロジェクト規約に従っている。

主な指摘事項は以下の通り:

- **必須対応**: i18nキーの `description` フィールド追加（ガイドライン準拠）
- **要確認**: `templates` パッケージの `model` 直接依存（アーキテクチャルール）、トピックnilケースの処理、テストのロケール設定
- **推奨**: URLパスヘルパーの追加による一貫性向上
