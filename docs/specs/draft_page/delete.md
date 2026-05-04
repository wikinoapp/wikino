# 下書き削除 仕様書

<!--
このテンプレートの使い方:
1. 操作対象のモデルに対応するディレクトリを `docs/specs/` 配下に作成（例: `docs/specs/page/`）
2. このファイルをそのディレクトリにコピー（例: cp docs/specs/template.md docs/specs/page/create.md）
3. [機能名] などのプレースホルダーを実際の内容に置き換え
4. 各セクションのガイドラインに従って記述
5. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**ファイルの配置ルール**:
- 仕様書は操作対象のモデル（名詞）ごとにディレクトリを分け、機能（動詞）をファイル名にする
  - 例: `docs/specs/user/sign-up.md`、`docs/specs/page/create.md`
- モデルに分類しにくい横断的な機能は、その機能自体を名詞としてディレクトリにする
  - 例: `docs/specs/search/full-text.md`
- モデルの定義・状態遷移・他モデルとの関係を記述する場合は `overview.md` を作成する
  - `overview.md` はモデルの静的な性質（「これは何か」）を書く場所
  - 操作に紐づく仕様（バリデーション、権限など）は各機能の仕様書に書く
- 詳細は [@docs/README.md](/workspace/docs/README.md) を参照

**仕様書の性質**:
- 仕様書は「現在のシステムの状態」を記述するドキュメントです
- 実装が完了したら、仕様書を最新の状態に更新してください
- 過去の状態はGit履歴で参照できるため、仕様書には常に現在の状態のみを記述します

**作業計画書との関係**:
- 新しい機能の場合: `docs/plans/` の作業計画書に概要・要件・設計を記述し、タスク完了後にこの仕様書を作成します
- 既存機能の変更の場合: `docs/plans/` の作業計画書に変更内容を記述し、タスク完了後にこの仕様書を更新します

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 概要

<!--
ガイドライン:
- この機能が現在「どのように動いているか」を簡潔に説明
- なぜこの仕組みになっているかの背景も記述
- 2-3段落程度で簡潔に
-->

下書き一覧画面 (`GET /drafts`) の各行に表示される三点リーダーアイコンのドロップダウンから、ユーザーが自分の DraftPage を明示的に削除できる機能。削除すると DraftPage と紐づく DraftPageRevision が DB から物理削除され、フラッシュメッセージとともに `/drafts` へリダイレクトされる。Go 版でのみ提供しており、Rails 版および Go 版のページ編集画面には削除操作を持たない。

Rails 版では編集画面の「キャンセル」リンクがページ表示に戻るのみで、DraftPage は公開時にしか削除されなかった。Go 版では `/drafts` を起点とした明示的な削除導線を追加し、不要な下書きをユーザー自身でクリーンアップできるようにしている。

**目的**:

- ユーザーが `/drafts` から不要な下書きを直接片付けられるようにする
- 下書きの操作権限を「編集」と「削除」で分離し、AI など外部エージェントに編集権限のみを付与する運用 (`draft_page:write` のみ与えて `draft_page:delete` は与えない) を可能にする

**背景**:

- 削除導線を `/drafts` のみに集約することで、編集画面 (自動保存があり UI 構造が異なる) の設計検討を別タスクに分離している
- 権限スコープを編集 (`draft_page:write`) と削除 (`draft_page:delete`) で分けることで、Wikino の認可モデル (スコープ含意展開) を維持したまま AI 運用に必要な権限分離を実現している

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### 削除操作

- ユーザーは `/drafts` 画面の各行に表示される三点リーダーアイコンをクリックしてドロップダウンを開き、「削除する」項目から自分の DraftPage を削除できる
- 「削除する」をクリックするとブラウザの `window.confirm` で確認ダイアログ (「この下書きを削除しますか？」) が表示され、OK で確定するとフォームが送信される
- 確認ダイアログでキャンセルした場合、フォーム送信は行われない
- 削除が成功すると、フラッシュメッセージ「下書きを削除しました」を表示しつつ `/drafts` にリダイレクトされる (操作元の一覧画面に戻り、続けて他の下書きを整理しやすくする)
- 削除によって DraftPage と紐づく DraftPageRevision が DB から物理削除される (公開済みページには影響しない)

### 削除対象

- 削除できるのは「呼び出しユーザー本人が所有する DraftPage」のみ
  - DraftPage は `space_member_id` でスペースメンバーに紐づいており、UseCase は本人の DraftPage しか取得しない
  - admin が他メンバーの下書きを操作する経路は本仕様の対象外 (将来必要になった時点で別 UseCase として実装する想定)
- 削除対象の DraftPage が存在しない (既に削除済み等) 場合は 404 を返す

### 権限

権限はスコープベースの `Authorizer` インターフェース (`MemberPolicy` / `GuestPolicy`) で判定される。

- `draft_page:delete` スコープを持つスペースメンバーのみが削除操作を実行できる
- `space:admin` スコープを持つメンバーは、`allResourceScopes` のスコープ含意展開によって自動的に `draft_page:delete` を保持するため削除できる
- `draft_page:write` (編集権限) は `draft_page:delete` を含意展開しない。編集権限のみを付与した運用 (例: AI に編集だけ任せて削除はさせない) が可能
- 未ログインユーザーは削除できず、`GET /sign_in` へリダイレクトされる
- スペースメンバーではないログインユーザー、および `draft_page:delete` を持たないメンバーが削除操作を試みた場合は 404 を返す (リソース存在の漏洩を防ぐため Forbidden も Not Found として扱う)

### セキュリティ

- 削除フォームは POST + `_method=DELETE` の Method Override パターンで送信され、CSRF トークンを伴う (CSRF ミドルウェアでトークン不一致なら 403 で拒否される)
- 確認ダイアログのテキストは削除フォーム (`<form>`) の `data-confirm` 属性に i18n 翻訳済みテキストをセットし、`hx-on:submit` ハンドラ内で `this.dataset.confirm` 経由で取り出す方式を採用している。インラインの JavaScript 文字列にユーザー向け文言を直接埋め込むことで生じうるエスケープ事故 (翻訳文字列内のクォートや特殊文字によるインライン JS の構文崩れ) を回避するのが目的

## 設計

<!--
ガイドライン:
- 現在の技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど)
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### エンドポイント

| メソッド | パス                                                   | ハンドラー                  | 説明           |
| -------- | ------------------------------------------------------ | --------------------------- | -------------- |
| DELETE   | `/s/{space_identifier}/pages/{page_number}/draft_page` | `draft_page.Handler.Delete` | DraftPage 削除 |

HTML フォームからは POST + `_method=DELETE` の Method Override パターンで呼び出す。同一パスの GET (`Show`) / PATCH (`Update`) は別仕様 ([ページ編集 仕様書](../page/edit.md)) で扱われる。

### 処理フロー

```
[Handler.Delete]
  ├─ 未認証なら GET /sign_in へ 302 リダイレクト
  ├─ URL パラメータ (space_identifier, page_number) をパース
  └─ DeleteDraftPageUsecase.Execute を呼び出す
      ↓
[DeleteDraftPageUsecase.Execute]
  ├─ 1. データ取得 (fetchPageAccessData)
  │     └─ space / space_member / page / topic / topic_member を取得
  ├─ 2. space_member が存在しなければ Forbidden を返す
  ├─ 3. DraftPageRepository.FindByPageAndMember で本人の下書きを取得
  │     └─ 取得できなければ ResourceNotFound を返す (本人の下書きしか触れない)
  ├─ 4. Authorizer.CanDeleteDraftPage() で認可チェック
  │     └─ false なら Forbidden を返す
  └─ 5. トランザクション内で永続化
        ├─ DraftPageRevisionRepository.DeleteByDraftPageID (revision を先に削除)
        └─ DraftPageRepository.Delete (FK 制約のため revision の後)
      ↓
[Handler.Delete]
  ├─ 成功時: フラッシュメッセージをセットし /drafts へ 303 (See Other) リダイレクト
  ├─ ResourceNotFound / Forbidden: handler.NotFound (404)
  └─ その他のエラー: 500 (詳細はログにのみ記録)
```

### 認可・所有者チェックの責務分担

所有者チェックを Policy ではなく UseCase 側のクエリで担保している。

| 関心事                                              | 担当                 | 実装                                                                           |
| --------------------------------------------------- | -------------------- | ------------------------------------------------------------------------------ |
| `draft_page:delete` スコープを持っているか          | Policy               | `Authorizer.CanDeleteDraftPage() bool`                                         |
| 削除対象の DraftPage が呼び出しユーザー本人のものか | UseCase / Repository | `DraftPageRepository.FindByPageAndMember(ctx, pageID, spaceMemberID, spaceID)` |

`MemberPolicy.CanDeleteDraftPage()` は `effectiveScopes[ScopeDraftPageDelete]` をそのまま返すだけのシンプルな判定で、所有者ID等は受け取らない。所有者の限定は `FindByPageAndMember` がスペースメンバーIDで絞り込むことで実質的に達成され、本人の DraftPage が見つからなければ `ResourceNotFound` (Handler 側で 404) になる。`space:admin` は `allResourceScopes` のスコープ含意展開で `draft_page:delete` を持つが、`FindByPageAndMember` の絞り込みにより本人の下書きしか削除できない。

### スコープ

`space:admin` のスコープ含意展開を行う `allResourceScopes()` (`internal/policy/scope.go`) に `draft_page:delete` が含まれているため、`space:admin` を持つメンバーには自動的に `draft_page:delete` が付与される。一方、`draft_page:write` の含意展開には `draft_page:delete` を含めていないため、編集権限のみのメンバーは削除できない。

| Go 定数                      | Rails 定数                 | 値                  | 含意展開元                          |
| ---------------------------- | -------------------------- | ------------------- | ----------------------------------- |
| `model.ScopeDraftPageDelete` | `Scope::DRAFT_PAGE_DELETE` | `draft_page:delete` | `space:admin` (`allResourceScopes`) |

Rails 版にはスコープ定数のみ同期しており、Policy メソッドおよび UI は実装していない (`/drafts` の削除操作は Go 版でのみ提供されるため)。

### エラーレスポンスの方針

| エラー種別                                  | レスポンス                     |
| ------------------------------------------- | ------------------------------ |
| 未認証                                      | `GET /sign_in` へ 302 Found    |
| 削除対象が見つからない (`ResourceNotFound`) | 404                            |
| 認可不足 (`Forbidden`)                      | 404 (リソース存在の漏洩を防ぐ) |
| サーバー内部エラー                          | 500 (詳細はログにのみ記録)     |
| CSRF トークン不一致                         | 403 (CSRF ミドルウェアが返す)  |

未認証時のレスポンスは、フォーム由来の DELETE リクエストであるため `/sign_in` へリダイレクトする。Update (PATCH) は JS 経由で 401 を返すが、Delete はフォーム送信に合わせている。

### コード設計

#### Handler

- `internal/handler/draft_page/delete.go` の `Handler.Delete`
- 既存標準ファイル名の `delete.go` を使用 (HTTP ハンドラーガイドラインに準拠)
- 依存 (Delete アクションが利用するもの): `flashMgr *session.FlashManager`, `deleteDraftPageUC *usecase.DeleteDraftPageUsecase`
  - 同一 `Handler` 構造体は Show/Update も担当するため他の依存 (`getPageDetailUC` 等) も保持しているが、それらは [ページ編集 仕様書](../page/edit.md) を参照

#### UseCase

- `internal/usecase/delete_draft_page.go` の `DeleteDraftPageUsecase`
- 依存: `db *sql.DB`, `spaceRepo`, `spaceMemberRepo`, `draftPageRepo`, `draftPageRevisionRepo`, `pageRepo`, `topicRepo`, `topicMemberRepo`
- データ取得は既存の `fetchPageAccessData` ヘルパーを再利用 (他の DraftPage 系 UseCase と同じパターン)
- `DraftPageRepository.Delete` は `PublishPageUsecase` でも公開時の下書き削除に使用しており、ID と `space_id` で削除する純粋な永続化処理

#### Policy

- `Authorizer` インターフェース (`internal/policy/authorizer.go`) に `CanDeleteDraftPage() bool` を追加
- `MemberPolicy.CanDeleteDraftPage`: `effectiveScopes[ScopeDraftPageDelete]` を返す
- `GuestPolicy.CanDeleteDraftPage`: 常に `false`

#### テンプレート

- `internal/templates/pages/draft_page/index.templ` の `draftRowActions` コンポーネントで実装
- 各行のドロップダウン ID は `draft-actions-{space_identifier}-{page_number}` で一意化
- フォーム送信は `hx-on:submit="if (!confirm(this.dataset.confirm)) { event.preventDefault(); return false; }"` で確認ダイアログを挟む (既存の `suggestion_change/index.templ` と同じパターン)
- 「削除する」ボタンは `text-error` クラスで赤系統に着色し、ゴミ箱アイコン (`trash-regular`) を併記する

#### i18n キー

| キー                                  | 日本語                     | 用途                                                |
| ------------------------------------- | -------------------------- | --------------------------------------------------- |
| `draft_page_index_actions_aria_label` | 下書きのアクション         | アクション列・ドロップダウン開閉ボタンの aria-label |
| `draft_page_delete_menu_item`         | 削除する                   | ドロップダウン内の削除メニュー項目                  |
| `draft_page_delete_confirm`           | この下書きを削除しますか？ | `window.confirm` で表示する確認ダイアログ           |
| `flash_draft_page_deleted`            | 下書きを削除しました       | 削除成功時のフラッシュメッセージ                    |

### データベース永続化

DraftPage は `draft_pages` テーブルで管理されており、`draft_page_revisions.draft_page_id` には FK 制約がある (`ON DELETE CASCADE` ではない)。そのため、削除はトランザクション内で以下の順で実行する。

1. `DraftPageRevisionRepository.DeleteByDraftPageID(ctx, draftPageID, spaceID)` で revision を全削除
2. `DraftPageRepository.Delete(ctx, draftPageID, spaceID)` で draft_pages の行を削除

`PublishPageUsecase` で公開時に下書きを削除する処理と同じ順序を採用している。

### ファイル構成

```
go/internal/
├── handler/draft_page/
│   ├── handler.go            # 依存に deleteDraftPageUC, flashMgr を追加
│   ├── delete.go             # Handler.Delete (新規)
│   └── delete_test.go        # Delete のテスト
├── usecase/
│   ├── delete_draft_page.go        # DeleteDraftPageUsecase (新規)
│   └── delete_draft_page_test.go   # UseCase テスト
├── policy/
│   ├── authorizer.go         # CanDeleteDraftPage を追加
│   ├── member_policy.go      # 実装
│   ├── guest_policy.go       # 常に false
│   └── scope.go              # allResourceScopes に ScopeDraftPageDelete を追加
├── model/
│   └── scope.go              # ScopeDraftPageDelete を追加
├── templates/pages/draft_page/
│   └── index.templ           # アクション列・ドロップダウン・削除フォームを追加
└── i18n/locales/
    ├── ja.toml               # 4 キー追加
    └── en.toml               # 4 キー追加

rails/app/
├── models/scope.rb           # DRAFT_PAGE_DELETE 定数を追加 (Go と同期)
└── policies/scope_expander.rb # ALL_RESOURCE_SCOPES に追加
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### 編集画面 (`/s/{space}/pages/{n}/edit`) にも削除ボタンを追加する

編集画面は既に Go 版で常時処理されているため技術的には追加可能だが、削除導線を `/drafts` の三点リーダー UI に集約する方針を採った。編集画面は自動保存など UI 構造が異なり、削除ボタンの置き場所と挙動の検討が別軸で必要なため、必要になった時点で別タスクで追加する。

### Rails 版にも同等の削除機能を実装する

`/drafts` 画面は Go 版で提供しており、Rails 版は段階的に Go 版へ移行中。Rails 版への横展開は不要と判断した。Rails 版にはスコープ定数 (`Scope::DRAFT_PAGE_DELETE`) のみ同期し、Policy メソッドおよび UI は実装していない。

### フィーチャーフラグで制御する

追加機能で互換性リスクが小さく、コード量も 1 PR に収まる規模のため、段階公開の必要性が低い。

### 既存の `CanUpdateDraftPage` を流用する (削除を `draft_page:write` で許可する)

削除を編集と同じ `draft_page:write` スコープで許可すれば要件は満たせるが、AI など外部エージェントに下書きを操作させる際に「編集はさせるが削除はさせない」運用ができなくなる。スコープを別建て (`draft_page:delete`) にし、専用 Policy メソッドを追加することで権限を分離した。`draft_page:write` の含意展開には `draft_page:delete` を含めていない。

### 「破棄 / `discard`」という用語で統一する

当初は「下書きは破棄する」というニュアンスを優先する案もあったが、他のリソース (`topic:delete`, `space:delete`, `attachment:delete` など) と表現が揃わず、内部用語に例外を作ってしまう。`page:trash` はソフトデリート専用の例外で、DraftPage はハードデリートのため `delete` が正しい。i18n の文言レベルでも、既存の `suggestion_change_remove_page_button` =「削除する」と整合する。

### 共通コンポーネント `components/post.templ` の `postDropdown` を拡張してフォームアクションをサポートする

`Post` コンポーネントは投稿 (編集提案・コメント) の表示用に設計されており、責務が異なる。下書き行のドロップダウンは `index.templ` 内にインラインで実装し、共通コンポーネント化はしていない。

### 専用の `DeleteDraftPageValidator` を作成する

フォーム入力値はパス変数 (`space_identifier`, `page_number`) のみで、形式バリデーションが不要。状態バリデーション (存在確認・所有者確認) は UseCase 内の認可チェックと `FindByPageAndMember` の絞り込みで実施しているため、専用 Validator は作成していない。

### 削除成功後に公開ページへリダイレクトする

`/drafts` 画面からの操作なので、操作元の一覧画面に戻るほうが自然で、続けて他の下書きを整理しやすい。

### Policy で所有者チェックを行う (`CanDeleteDraftPage(isOwner bool)`)

当初の作業計画では `MemberPolicy.CanDeleteDraftPage(isOwner bool)` で「`draft_page:delete` を保有 + (所有者 OR `space:admin`)」を判定する案を検討した。実装にあたっては、所有者の限定を Policy ではなく UseCase 側のクエリ (`DraftPageRepository.FindByPageAndMember` でスペースメンバーIDで絞り込み) に集約し、Policy はスコープ判定のみを担う設計に変更した。`space:admin` は `allResourceScopes` のスコープ含意展開で `draft_page:delete` を取得する。これにより Policy のシグネチャがシンプルになり、所有者の限定が「呼び出しユーザー本人のドラフトしか取得しない」というデータアクセス段階で自然に達成される。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページ編集 仕様書](../page/edit.md) - 同一パス上の `Show` (GET) / `Update` (PATCH) や DraftPage / DraftPageRevision のデータモデルを記述
- [`internal/handler/draft_page/delete.go`](/workspace/go/internal/handler/draft_page/delete.go) - Delete ハンドラーの実装
- [`internal/usecase/delete_draft_page.go`](/workspace/go/internal/usecase/delete_draft_page.go) - DeleteDraftPageUsecase の実装
- [`internal/policy/scope.go`](/workspace/go/internal/policy/scope.go) - スコープ含意展開 (`allResourceScopes`)
- [`internal/templates/pages/draft_page/index.templ`](/workspace/go/internal/templates/pages/draft_page/index.templ) - 一覧画面のドロップダウン UI
