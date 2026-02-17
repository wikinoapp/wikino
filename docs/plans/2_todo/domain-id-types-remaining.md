# 残りモデルへのドメインID型の適用 作業計画書

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

- なし（内部リファクタリングのため仕様書は不要）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

`page-edit-go-migration` の作業計画書（タスク 2a-1）で、`internal/model/id.go` にドメインID型（SpaceID, TopicID, PageID, SpaceMemberID, TopicMemberID, DraftPageID）を導入し、対応するモデル・リポジトリ・ポリシー・テストヘルパーを更新した。

しかし、まだ以下のモデルのIDフィールドが `string` のまま残っている：

- User, PageEditor, PageRevision, Attachment, PageAttachmentReference, EmailConfirmation, PasswordResetToken, UserPassword, UserSession, UserTwoFactorAuth

これらのモデルにもドメインID型を適用し、IDの取り違えをコンパイル時に検出できるようにする。特に `UserID` は複数のモデル（SpaceMember, PasswordResetToken, UserPassword, UserSession, UserTwoFactorAuth）で外部キーとして参照されるため、導入効果が高い。

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

- すべてのモデルのIDフィールドに専用のドメインID型を使用する
- 他モデルへの参照（UserIDなど）にもドメインID型を使用する
- 既存のテストがすべて通ること
- ランタイムの挙動に変更がないこと（型安全性の強化のみ）

## 実装ガイドラインの参照

<!--
**重要**: 設計を行う前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計の段階でガイドラインに準拠していることを確認してください。
-->

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約（ドメインID型のセクションを参照）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

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

### 追加するドメインID型

`internal/model/id.go` に以下の型を追加する：

| 型名 | 対象モデル | String()メソッド |
|------|-----------|-----------------|
| `UserID` | User | あり |
| `PageEditorID` | PageEditor | あり |
| `PageRevisionID` | PageRevision | あり |
| `AttachmentID` | Attachment | あり |
| `PageAttachmentReferenceID` | PageAttachmentReference | あり |
| `EmailConfirmationID` | EmailConfirmation | あり |
| `PasswordResetTokenID` | PasswordResetToken | あり |
| `UserPasswordID` | UserPassword | あり |
| `UserSessionID` | UserSession | あり |
| `UserTwoFactorAuthID` | UserTwoFactorAuth | あり |

### 変更対象のモデルフィールド

**自身のIDフィールド**（`string` → 専用型）:

| モデル | フィールド | 変更 |
|--------|-----------|------|
| User | ID | `string` → `UserID` |
| PageEditor | ID | `string` → `PageEditorID` |
| PageRevision | ID | `string` → `PageRevisionID` |
| Attachment | ID | `string` → `AttachmentID` |
| PageAttachmentReference | ID | `string` → `PageAttachmentReferenceID` |
| EmailConfirmation | ID | `string` → `EmailConfirmationID` |
| PasswordResetToken | ID | `string` → `PasswordResetTokenID` |
| UserPassword | ID | `string` → `UserPasswordID` |
| UserSession | ID | `string` → `UserSessionID` |
| UserTwoFactorAuth | ID | `string` → `UserTwoFactorAuthID` |

**外部キーフィールド**（`string` → 専用型）:

| モデル | フィールド | 変更 |
|--------|-----------|------|
| SpaceMember | UserID | `string` → `UserID` |
| PasswordResetToken | UserID | `string` → `UserID` |
| UserPassword | UserID | `string` → `UserID` |
| UserSession | UserID | `string` → `UserID` |
| UserTwoFactorAuth | UserID | `string` → `UserID` |
| Page | FeaturedImageAttachmentID | `*string` → `*AttachmentID` |
| PageAttachmentReference | AttachmentID | `string` → `AttachmentID` |

### 変更パターン

既存の導入パターン（SpaceID, PageID 等）に従う：

**id.go への型追加**:

```go
type UserID string
func (id UserID) String() string { return string(id) }
```

**モデルのIDフィールド変更**:

```go
// 変更前
type User struct {
    ID string
}

// 変更後
type User struct {
    ID UserID
}
```

**リポジトリの toModel() での変換**:

```go
// 変更前
ID: row.ID,

// 変更後
ID: model.UserID(row.ID),
```

**テストビルダーの Build() 戻り値変更**:

```go
// 変更前
func (b *UserBuilder) Build() string {

// 変更後
func (b *UserBuilder) Build() model.UserID {
```

### タスク分割の方針

変更の影響範囲が広いため、以下の基準でタスクを分割する：

1. **UserID の導入**: 最も影響範囲が広い（User自身 + 5つのモデルの外部キー + Usecase + Handler）。まずUserIDを導入し、関連するすべてのモデル・リポジトリ・ユースケース・テストを一括で更新する。
2. **スペース内リソースのID型**: PageEditorID, PageRevisionID, AttachmentID, PageAttachmentReferenceID を導入。スペース内のリソースに閉じた変更。
3. **認証関連のID型**: EmailConfirmationID, PasswordResetTokenID, UserPasswordID, UserSessionID, UserTwoFactorAuthID を導入。認証フローに関連するモデルに閉じた変更。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### 全モデルを1タスクで一括変更

10個のID型を一度に追加すると変更ファイル数が30以上になり、PRが大きくなりすぎるため不採用。影響範囲の関連性に基づいて3タスクに分割した。

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

### フェーズ 1: UserID の導入

UserID は最も参照箇所が多いため、最初に導入する。User モデル自身と、UserID を外部キーとして持つ 5 つのモデル（SpaceMember, PasswordResetToken, UserPassword, UserSession, UserTwoFactorAuth）、およびそれらを参照する Usecase・Handler を更新する。

- [ ] **1-1**: [Go] UserID ドメインID型の定義とモデル・リポジトリへの適用
  - `internal/model/id.go` に `UserID` 型と `String()` メソッドを追加
  - `internal/model/user.go` の `ID` フィールドを `UserID` に変更
  - `internal/model/space_member.go` の `UserID` フィールドを `model.UserID` に変更
  - `internal/model/password_reset_token.go` の `ID` を `PasswordResetTokenID`、`UserID` を `model.UserID` に変更
  - `internal/model/user_password.go` の `ID` を `UserPasswordID`、`UserID` を `model.UserID` に変更
  - `internal/model/user_session.go` の `ID` を `UserSessionID`、`UserID` を `model.UserID` に変更
  - `internal/model/user_two_factor_auth.go` の `ID` を `UserTwoFactorAuthID`、`UserID` を `model.UserID` に変更
  - `internal/model/id.go` に `PasswordResetTokenID`, `UserPasswordID`, `UserSessionID`, `UserTwoFactorAuthID` 型も追加
  - 対応するリポジトリの `toModel()` で `model.UserID(row.UserID)` 等の変換を追加
  - `internal/testutil/user_builder.go` の `Build()` 戻り値を `model.UserID` に変更
  - `internal/testutil/password_reset_token_builder.go` の `Build()` 戻り値を `model.PasswordResetTokenID` に変更
  - 想定ファイル数: 実装 ~15 ファイル, テスト ~10 ファイル
  - 想定行数: 実装 ~150 行, テスト ~100 行
  - 依存: なし

- [ ] **1-2**: [Go] UserID に伴う Usecase・Handler の更新
  - `internal/usecase/` 配下で `UserID string` を `model.UserID` に変更（create_account.go, create_user_session.go, create_password_reset_token.go, mark_email_as_confirmed.go, update_password_reset.go, consume_recovery_code.go 等）
  - `internal/handler/` 配下で User.ID を string として扱っている箇所を更新
  - 想定ファイル数: 実装 ~10 ファイル, テスト ~8 ファイル
  - 想定行数: 実装 ~100 行, テスト ~80 行
  - 依存: 1-1

### フェーズ 2: スペース内リソースの ID 型

スペース内のリソースに閉じた変更。PageEditor, PageRevision, Attachment, PageAttachmentReference のID型を導入する。

- [ ] **2-1**: [Go] スペース内リソースのドメインID型の定義とモデル・リポジトリへの適用
  - `internal/model/id.go` に `PageEditorID`, `PageRevisionID`, `AttachmentID`, `PageAttachmentReferenceID` 型を追加
  - `internal/model/page_editor.go` の `ID` を `PageEditorID` に変更
  - `internal/model/page_revision.go` の `ID` を `PageRevisionID` に変更
  - `internal/model/attachment.go` の `ID` を `AttachmentID` に変更
  - `internal/model/page_attachment_reference.go` の `ID` を `PageAttachmentReferenceID`、`AttachmentID` を `model.AttachmentID` に変更
  - `internal/model/page.go` の `FeaturedImageAttachmentID` を `*AttachmentID` に変更
  - 対応するリポジトリの `toModel()` を更新
  - `internal/testutil/page_revision_builder.go` の `Build()` 戻り値を `model.PageRevisionID` に変更
  - テスト内の型不一致を修正
  - 想定ファイル数: 実装 ~10 ファイル, テスト ~5 ファイル
  - 想定行数: 実装 ~100 行, テスト ~50 行
  - 依存: なし（フェーズ 1 と並行可能）

### フェーズ 3: 認証関連の ID 型

EmailConfirmation のID型を導入する。

- [ ] **3-1**: [Go] EmailConfirmationID ドメインID型の定義とモデル・リポジトリへの適用
  - `internal/model/id.go` に `EmailConfirmationID` 型を追加
  - `internal/model/email_confirmation.go` の `ID` を `EmailConfirmationID` に変更
  - 対応するリポジトリの `toModel()` を更新
  - `internal/testutil/email_confirmation_builder.go` の `Build()` 戻り値を `model.EmailConfirmationID` に変更
  - `internal/usecase/` 配下で `EmailConfirmationID string` を参照している箇所を更新
  - テスト内の型不一致を修正
  - 想定ファイル数: 実装 ~5 ファイル, テスト ~5 ファイル
  - 想定行数: 実装 ~50 行, テスト ~50 行
  - 依存: なし（フェーズ 1, 2 と並行可能）
