# Handler → Repository 依存関係の廃止 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
   例: cp docs/plans/template.md docs/plans/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**作業計画書の性質**:
- 作業計画書は「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- [アーキテクチャガイド](/workspace/go/docs/architecture-guide.md) - 変更対象

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go 版 Wikino の Handler（Presentation 層）から Repository（Domain/Infrastructure 層）への直接依存を廃止し、すべてのデータアクセスを UseCase（Application 層）経由に統一するアーキテクチャリファクタリングを行う。

**背景**:

現在のアーキテクチャでは、Handler から Repository を直接呼び出すことが許可されている（読み取り専用の場合）。しかし、この設計には以下の問題がある：

- Repository の書き込みメソッド（Insert/Update/Delete）が Handler から呼び出される可能性があり、UseCase でトランザクション管理するという原則が規約（convention）でしか守られない
- Handler → UseCase と Handler → Repository の 2 つの依存パスが存在し、「どちらを使うべきか」の判断コストが発生する
- 実際に `user_session/delete.go` で Repository の Delete メソッドが直接呼び出されている違反箇所が存在する

**変更後のアーキテクチャ**:

Handler は UseCase のみに依存し、Repository には依存しない。依存グラフが「Handler → UseCase → Repository」の一方向に統一される。

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

<!--
「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
箇条書きで簡潔に
-->

- Handler の自身のメソッド（Show, Create, Index 等）は Repository を直接呼び出さず、UseCase を経由してデータアクセスする
- 参照系の処理にも UseCase を使用する（読み取り専用 UseCase の導入）
- 複数の Repository 呼び出しを 1 つの UseCase に集約し、Handler をシンプルに保つ
- Validator を `internal/validator/` パッケージに移動し、Handler パッケージから repository の import を完全に排除する
- depguard で Handler → Repository の依存をリント時に禁止する
- 既存の機能に変更を加えない（リファクタリングのみ）

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

- **保守性**: 依存関係のルールが明快で、新しい機能を追加する際に迷わない

## 実装ガイドラインの参照

<!--
**重要**: 作業計画書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
作業計画書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - UI設計（画面構成、ユーザーフローなど）
  - セキュリティ設計（認証・認可、トークン管理など）
  - コード設計（パッケージ構成、主要な構造体など）

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### 依存関係の変更

**変更前**:

```
Handler → UseCase（書き込み処理）
Handler → Repository（読み取り処理）
Handler → Validator → Repository（状態バリデーション）
```

**変更後**:

```
Handler → UseCase（すべてのデータアクセス）
Handler → Validator（状態バリデーション）
Validator → Repository（Validator は独立パッケージ internal/validator/ に配置）
```

### UseCase の役割の拡張

現在の UseCase は「トランザクションを伴う永続化処理」のみを担当しているが、参照系の処理も担当するように役割を拡張する。

| 種類                     | 責務                               | トランザクション        |
| ------------------------ | ---------------------------------- | ----------------------- |
| 書き込み UseCase（既存） | 永続化処理、ビジネスロジック       | あり（WithTx パターン） |
| 読み取り UseCase（新規） | データ取得、複数 Repository の集約 | なし                    |

### 読み取り UseCase の設計パターン

読み取り UseCase は、Handler が必要とするデータ取得ロジックを集約する。

```go
// internal/usecase/get_topic_detail.go
type GetTopicDetailUsecase struct {
    spaceRepo       *repository.SpaceRepository
    spaceMemberRepo *repository.SpaceMemberRepository
    topicRepo       *repository.TopicRepository
    topicMemberRepo *repository.TopicMemberRepository
    pageRepo        *repository.PageRepository
}

type GetTopicDetailInput struct {
    SpaceIdentifier model.SpaceIdentifier
    TopicNumber     int32
    UserID          *model.UserID
    Page            int32
}

type GetTopicDetailOutput struct {
    Space       *model.Space
    SpaceMember *model.SpaceMember
    Topic       *model.Topic
    TopicMember *model.TopicMember
    PinnedPages []*model.Page
    Pages       []*model.Page
    Pagination  *repository.PaginationResult
}

func (uc *GetTopicDetailUsecase) Execute(ctx context.Context, input GetTopicDetailInput) (*GetTopicDetailOutput, error) {
    space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
    // ...
}
```

### Validator パッケージの分離

Validator を `internal/handler/` から `internal/validator/` に移動し、独立パッケージとする。これにより depguard で Handler → Repository の依存を完全に禁止できるようになる。

**移動の設計**:

```
変更前:
internal/handler/sign_in/
├── handler.go
├── validator.go          # Repository を import
├── validator_test.go
├── new.go
└── create.go

変更後:
internal/handler/sign_in/
├── handler.go
├── new.go
└── create.go
internal/validator/
├── sign_in.go            # Repository を import
└── sign_in_test.go
```

**命名規則**:

| 変更前（handler パッケージ内）   | 変更後（validator パッケージ）         |
| -------------------------------- | -------------------------------------- |
| `handler/sign_in/validator.go`   | `validator/sign_in.go`                 |
| `sign_in.CreateValidator`        | `validator.SignInCreateValidator`      |
| `sign_in.NewCreateValidator()`   | `validator.NewSignInCreateValidator()` |
| `handler/page_move/validator.go` | `validator/page_move.go`               |
| `page_move.CreateValidator`      | `validator.PageMoveCreateValidator`    |

**構築パターンの変更**:

Validator は `main.go` で構築し、Handler のコンストラクタに渡す。Handler は repository パッケージを一切 import しない。

```go
// 変更前: Handler が Repository を受け取り、内部で Validator を構築
func NewHandler(cfg *config.Config, userRepo *repository.UserRepository, ...) *Handler {
    return &Handler{
        validator: NewCreateValidator(userRepo),
    }
}

// 変更後: main.go で Validator を構築し、Handler に渡す
// main.go
signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo)
signInHandler := sign_in.NewHandler(cfg, sessionMgr, signInValidator, createSessionUC)
```

**Validator パッケージ分離のメリット**:

- **depguard で完全に強制可能**: Handler パッケージから repository パッケージへの依存をリントで禁止できる
- **再利用性の向上**: Handler だけでなく Worker からもバリデーションを呼び出せるようになる
- **依存方向の明確化**: Handler → Validator → Repository という依存チェーンが明確になる

### 影響範囲の分析

現在のハンドラーを以下の 3 グループに分類する：

**グループ A: Handler 自身が Repository を呼び出している（読み取り UseCase の作成が必要）**

| ハンドラー                        | 使用している Repository                                 | 必要な UseCase                |
| --------------------------------- | ------------------------------------------------------- | ----------------------------- |
| `topic/show`                      | Space, SpaceMember, Topic, TopicMember, Page            | `GetTopicDetailUsecase`       |
| `page/show`                       | Space, SpaceMember, Page, DraftPage, Topic, TopicMember | `GetPageDetailUsecase`        |
| `page_location/show`              | Space, SpaceMember, Page                                | `GetPageLocationUsecase`      |
| `page_backlinks/show`             | Space, SpaceMember, Page                                | `GetPageBacklinksUsecase`     |
| `page_backlink_list/show`         | Space, SpaceMember, Page                                | `GetPageBacklinkListUsecase`  |
| `page_link_list/show`             | Space, SpaceMember, Page                                | `GetPageLinkListUsecase`      |
| `draft_page/show,new,edit`        | Space, SpaceMember, Page, Topic, TopicMember, DraftPage | 複数の UseCase                |
| `draft_page/create,update,delete` | Space, SpaceMember（書き込みは既存 UseCase）            | 読み取り部分を UseCase に     |
| `draft_page_index/index`          | Space, SpaceMember, DraftPage                           | `ListDraftPagesUsecase`       |
| `draft_page_revision/show`        | DraftPage, Page, DraftPageRevision                      | `GetDraftPageRevisionUsecase` |
| `page_move/new,create`            | Space, SpaceMember, Page, Topic, TopicMember            | 読み取り部分を UseCase に     |

**グループ B: Handler が Repository を Validator 経由でのみ使用（Validator の `internal/validator/` への移動）**

| ハンドラー                    | 変更内容                                                                            |
| ----------------------------- | ----------------------------------------------------------------------------------- |
| `sign_in`                     | `validator.go` を `internal/validator/sign_in.go` に移動、Handler を UseCase のみに |
| `account`                     | `validator.go` を `internal/validator/account.go` に移動                            |
| `email_confirmation`          | `validator.go` を `internal/validator/email_confirmation.go` に移動                 |
| `password`                    | `validator.go` を `internal/validator/password.go` に移動                           |
| `password_reset`              | `validator.go` を `internal/validator/password_reset.go` に移動                     |
| `sign_in_two_factor`          | `validator.go` を `internal/validator/sign_in_two_factor.go` に移動                 |
| `sign_in_two_factor_recovery` | `validator.go` を `internal/validator/sign_in_two_factor_recovery.go` に移動        |

**グループ C: アーキテクチャ違反の修正が必要**

| ハンドラー            | 違反内容                            | 修正                              |
| --------------------- | ----------------------------------- | --------------------------------- |
| `user_session/delete` | Repository の Delete を直接呼び出し | `DeleteUserSessionUsecase` を作成 |

### depguard による強制

Validator を `internal/validator/` に移動することで、depguard で Handler → Repository の依存を完全に禁止できる。

```yaml
# .golangci.yml
handler-layer:
  files:
    - "**/internal/handler/**"
  deny:
    - pkg: github.com/wikinoapp/wikino/go/internal/query
      desc: "HandlerはQueryに直接依存できません。UseCaseを経由してください。"
    - pkg: github.com/wikinoapp/wikino/go/internal/repository
      desc: "HandlerはRepositoryに直接依存できません。UseCaseを経由してください。"

usecase-layer:
  files:
    - "**/internal/usecase/**"
  deny:
    - pkg: github.com/wikinoapp/wikino/go/internal/policy
      desc: "UseCaseはPolicyに直接依存できません。認可チェックはHandlerで行ってください。"
```

これにより、Handler パッケージ内で repository を import しようとするとリントエラーになる。

### 認可チェック（Policy）の配置

認可チェック（`policy.TopicPolicy` など）は **Presentation 層（Handler）** で行い、UseCase は Policy に依存しない。

**方針**:

- UseCase はデータ取得と永続化に専念し、認可判断に必要なデータ（`TopicMember` など）を Output に含めて返す
- Handler が UseCase の Output を使って Policy チェックを実行する
- depguard で UseCase → Policy の依存を禁止する

**理由**:

- Handler が唯一の Entry Point であるため、Handler で認可すれば漏れがない
- UseCase の責務が明確になる（データ取得 / ビジネスロジックに集中）
- 認可ロジックが Handler に集約されて見通しが良い

**対象 UseCase**:

| UseCase                       | 現在の Policy 依存                           | 変更内容                                                     |
| ----------------------------- | -------------------------------------------- | ------------------------------------------------------------ |
| `GetSaveDraftPageDataUsecase` | `topicPolicy.CanUpdatePage` を内部で呼び出し | Output に `TopicMember` を追加し、Handler で Policy チェック |
| `GetPageMoveDataUsecase`      | `topicPolicy.CanUpdatePage` を内部で呼び出し | Output に `TopicMember` を追加し、Handler で Policy チェック |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### A. Handler → Repository を引き続き許可する（現状維持）

Handler が Repository を直接呼び出すことを許可し、規約とコードレビューで書き込み呼び出しを防止する方針。

**不採用の理由**:

- 規約だけでは書き込みメソッドの呼び出しを防止できず、実際に違反（`user_session/delete.go`）が発生している
- Handler の依存先が UseCase と Repository の 2 つに分散し、ルールが複雑になる
- 依存グラフが一方向に統一されず、アーキテクチャの見通しが悪い

### B. Repository を ReadRepository と WriteRepository に分離する

Repository をインターフェースで分離し、Handler には ReadRepository のみ注入する方針。

**不採用の理由**:

- インターフェースの管理コストが増える
- Go のプラグマティックな哲学に反する（過度な抽象化）
- UseCase 経由に統一するほうがルールとしてシンプル

### C. Validator を handler パッケージ内に残す

Validator を `internal/handler/` 内に配置したまま、depguard による強制は諦めて構造的な規約とコードレビューで対応する方針。

**不採用の理由**:

- depguard で強制できないと、`user_session/delete.go` のような違反が再発する可能性がある
- 「Handler パッケージは repository を import しない」というルールを完全に強制できるメリットが大きい
- Validator を独立パッケージにすることで、将来的に Worker からもバリデーションを再利用できる

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- **フェーズ番号は半角英数字とハイフンのみで表記**してください（ブランチ名に使用するため）
  - 例: フェーズ 1, フェーズ 2, フェーズ 5a（フェーズ 5 と 6 の間に追加する場合）
  - NG: フェーズ 5.5（ドットは使用不可）
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）

プラットフォームプレフィックス:
- Go版またはRails版の修正を行うタスクには、タスク名の先頭にプラットフォームを示すプレフィックスを付けてください
- フォーマット: **フェーズ番号-タスク番号**: [Go] タスク名 または **フェーズ番号-タスク番号**: [Rails] タスク名
- Go版とRails版の両方を修正する場合は、別々のタスクに分けてください
- 例:
  - `- [ ] **1-1**: [Go] マイグレーション作成`
  - `- [ ] **1-2**: [Rails] モデルへのコールバック追加`
-->

### フェーズ 1: アーキテクチャドキュメントの更新

- [x] **1-1**: [Go] architecture-guide.md と CLAUDE.md の更新
  - `architecture-guide.md`: Handler → Repository の依存を禁止するルールを追加
  - `architecture-guide.md`: UseCase の役割を「書き込み + 読み取り」に拡張する記述を追加
  - `architecture-guide.md`: Validator パッケージの分離について記述を追加
  - `architecture-guide.md`: 読み取り UseCase の設計パターンとコード例を追加
  - `CLAUDE.md`: 重要な設計原則セクションを更新
  - `validation-guide.md`: Validator の配置先を `internal/validator/` に変更する記述を追加
  - `handler-guide.md`: ハンドラーディレクトリから `validator.go` を削除する旨を更新
  - **想定ファイル数**: 約 4 ファイル（実装 4 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

### フェーズ 2: Validator パッケージの分離と depguard 更新

- [x] **2-1**: [Go] `internal/validator/` パッケージの作成と認証系 Validator の移動（sign_in, sign_in_two_factor, sign_in_two_factor_recovery）
  - `internal/validator/` パッケージを新規作成
  - `handler/sign_in/validator.go` → `validator/sign_in.go` に移動・リネーム
  - `handler/sign_in_two_factor/validator.go` → `validator/sign_in_two_factor.go` に移動・リネーム
  - `handler/sign_in_two_factor_recovery/validator.go` → `validator/sign_in_two_factor_recovery.go` に移動・リネーム
  - 対応するテストファイルも移動
  - Handler の `NewHandler` を更新し、Validator を外部から受け取るように変更
  - `main.go` で Validator を構築し Handler に渡すように変更
  - **想定ファイル数**: 約 12 ファイル（実装 9 + テスト 3）
  - **想定行数**: 約 200 行（実装 150 行 + テスト 50 行）

- [x] **2-2**: [Go] 認証系 Validator の移動（account, email_confirmation, password, password_reset）
  - 2-1 と同様のパターンで各 Validator を `internal/validator/` に移動
  - Handler の更新と `main.go` の更新
  - テストの移動・更新
  - **想定ファイル数**: 約 14 ファイル（実装 10 + テスト 4）
  - **想定行数**: 約 250 行（実装 180 行 + テスト 70 行）

- [x] **2-3**: [Go] コンテンツ系 Validator の移動（page, page_move）
  - `handler/page/validator.go` → `validator/page.go` に移動・リネーム
  - `handler/page_move/validator.go` → `validator/page_move.go` に移動・リネーム
  - 対応するテストファイルも移動
  - Handler の更新と `main.go` の更新
  - `.golangci.yml` の validator-layer ファイルパターンを `**/internal/validator/**` に更新
  - **想定ファイル数**: 約 10 ファイル（実装 7 + テスト 3）
  - **想定行数**: 約 200 行（実装 140 行 + テスト 60 行）

### フェーズ 3: アーキテクチャ違反の修正

- [x] **3-1**: [Go] DeleteUserSessionUsecase の作成と user_session ハンドラーの修正
  - `internal/usecase/delete_user_session.go` を作成
  - `user_session/handler.go` から Repository フィールドを削除し UseCase フィールドに変更
  - `user_session/delete.go` を UseCase 経由に修正
  - `main.go` のルーティング登録を更新
  - テスト追加
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 100 行（実装 60 行 + テスト 40 行）

### フェーズ 4: トピック系ハンドラーの UseCase 化

- [x] **4-1**: [Go] GetTopicDetailUsecase の作成と topic ハンドラーの修正
  - `internal/usecase/get_topic_detail.go` を作成（スペース取得、メンバー確認、トピック取得、ページ取得を集約）
  - `topic/handler.go` から Repository フィールドを削除し UseCase フィールドに変更
  - `topic/show.go` を UseCase 経由に修正
  - `main.go` の更新
  - UseCase テストとハンドラーテストの更新
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 5: ページ系ハンドラーの UseCase 化

- [x] **5-1**: [Go] GetPageDetailUsecase の作成と page ハンドラーの修正
  - `internal/usecase/get_page_detail.go` を作成
  - `page/handler.go` から Repository フィールドを削除し UseCase フィールドに変更
  - `page/show.go` を UseCase 経由に修正
  - `main.go` の更新
  - テスト追加・更新
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

- [x] **5-2**: [Go] ページ補助ハンドラーの UseCase 化（page_location, page_backlinks, page_backlink_list, page_link_list）
  - 各ハンドラー用の読み取り UseCase を作成
  - 各ハンドラーの Handler 構造体から Repository フィールドを削除
  - `main.go` の更新
  - テスト追加・更新
  - **想定ファイル数**: 約 14 ファイル（実装 10 + テスト 4）
  - **想定行数**: 約 300 行（実装 200 行 + テスト 100 行）

- [x] **5-3**: [Go] page_move ハンドラーの UseCase 化
  - 読み取り部分の UseCase を作成
  - Handler 構造体から Repository フィールドを削除
  - `main.go` の更新
  - テスト追加・更新
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 200 行（実装 120 行 + テスト 80 行）

### フェーズ 6: 下書きページ系ハンドラーの UseCase 化

- [x] **6-1**: [Go] draft_page ハンドラーの参照系 UseCase 化（show, new, edit）
  - 下書きページ表示・新規作成フォーム・編集フォーム用の読み取り UseCase を作成
  - Handler 構造体から Repository フィールドを削除（書き込み UseCase は既存のものを使用）
  - `main.go` の更新
  - テスト追加・更新
  - **想定ファイル数**: 約 8 ファイル（実装 5 + テスト 3）
  - **想定行数**: 約 300 行（実装 180 行 + テスト 120 行）

- [x] **6-2**: [Go] draft_page ハンドラーの create, update, delete の読み取り部分を UseCase 化
  - 書き込みハンドラー内の読み取り処理（スペース取得、権限チェック等）を UseCase に移動
  - 既存の書き込み UseCase との連携を整理
  - テスト更新
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 200 行（実装 120 行 + テスト 80 行）

- [x] **6-3**: [Go] draft_page_index, draft_page_revision ハンドラーの UseCase 化
  - 下書き一覧とリビジョン表示用の読み取り UseCase を作成
  - Handler 構造体から Repository フィールドを削除
  - `main.go` の更新
  - テスト追加・更新
  - **想定ファイル数**: 約 8 ファイル（実装 5 + テスト 3）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 7: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **7-0**: [Go] UseCase から Policy 依存を除去し、認可チェックを Handler に移動
  - `GetSaveDraftPageDataUsecase`: Policy チェックを削除、Output に `TopicMember` を追加
  - `GetPageMoveDataUsecase`: Policy チェックを削除、Output に `TopicMember` を追加
  - `draft_page/update.go`: UseCase の Output から TopicMember を受け取り、Handler で Policy チェックを実行
  - `page_move/new.go`, `page_move/create.go`: 同様に Handler で Policy チェックを実行
  - テスト更新
  - **想定ファイル数**: 約 8 ファイル（実装 5 + テスト 3）
  - **想定行数**: 約 100 行（実装 60 行 + テスト 40 行）

- [x] **7-1**: [Go] `.golangci.yml` に Handler → Repository 禁止と UseCase → Policy 禁止の depguard ルールを追加
  - すべての Handler から Repository の直接 import が除去された後に追加する
  - すべての UseCase から Policy の直接 import が除去された後に追加する
  - `make lint` で Handler → Repository、UseCase → Policy の依存違反がないことを確認
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

- [x] **7-2**: アーキテクチャガイドの最終確認
  - フェーズ 1 で更新した `architecture-guide.md` の内容が実装結果と一致しているか確認
  - 実装中に変更した設計判断があれば反映する
  - 作業計画書の採用しなかった方針をアーキテクチャガイドに転記する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **既存 UseCase の読み取り UseCase への分離**: 既存の書き込み UseCase 内に読み取り処理が含まれている場合、今回は分離しない。書き込み UseCase は現状のまま維持する
- **Validator の UseCase 経由化**: Validator を UseCase 経由で Repository にアクセスする方式への変更は行わない。Validator → Repository の直接依存は許容する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Go 版アーキテクチャガイド](/workspace/go/docs/architecture-guide.md) - 現行のアーキテクチャ定義
