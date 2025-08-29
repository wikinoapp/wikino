# 権限管理の改善

## 現在の権限まわりの処理

### アーキテクチャ概要

権限管理は主に以下のクラスで構成されています：

1. **`SpaceMemberPolicy`** - 権限チェックのメインクラス
   - すべての権限判定ロジックを集約
   - 各種アクション（ページ作成、更新、削除など）の可否を判定
   - ユーザーとスペースメンバーの関連性を検証

2. **`SpaceMemberRole`** - ロール定義
   - `Owner` - スペースの管理者権限
   - `Member` - 通常メンバー権限
   - 各ロールが持つ権限（permissions）を定義

3. **`SpaceMemberPermission`** - 権限種別の定義
   - `CreateTopic` - トピック作成権限
   - `CreatePage` - ページ作成権限
   - `CreateDraftPage` - ドラフトページ作成権限
   - `ExportSpace` - スペースエクスポート権限
   - `UpdateSpace` - スペース更新権限
   - `UpdateTopic` - トピック更新権限

### 権限チェックのフロー

1. **コントローラーでの権限チェック**

   ```ruby
   space_member_policy = SpaceMemberPolicy.new(
     user_record: current_user_record,
     space_member_record: current_space_member_record
   )

   unless space_member_policy.can_update_space?(space_record:)
     # 権限がない場合の処理
   end
   ```

2. **ロールベースの権限付与**
   - `Owner`ロール：すべての権限を保有
   - `Member`ロール：基本的な操作権限のみ（スペース管理権限なし）

3. **権限チェックメソッド**
   - `joined_space?` - スペースに参加しているか
   - `can_update_space?` - スペース更新権限
   - `can_create_topic?` - トピック作成権限
   - `can_update_topic?` - トピック更新権限
   - `can_create_page?` - ページ作成権限
   - `can_update_page?` - ページ更新権限
   - `can_show_page?` - ページ閲覧権限（公開トピックは誰でも閲覧可）
   - `can_trash_page?` - ページ削除権限
   - `can_export_space?` - エクスポート権限
   - `can_upload_attachment?` - ファイルアップロード権限
   - `can_view_attachment?` - ファイル閲覧権限
   - `can_delete_attachment?` - ファイル削除権限
   - `can_manage_attachments?` - ファイル管理権限

### 特徴的な実装

1. **公開トピックの扱い**
   - 公開トピック内のページは非メンバーでも閲覧可能
   - 公開トピックの添付ファイルも非メンバーが閲覧可能

2. **権限の継承関係**
   - スペースメンバーであることが基本条件
   - 特定の権限（UpdateSpaceなど）が追加の操作を可能にする

3. **添付ファイルの権限**
   - アップロード者本人または管理者のみ削除可能
   - 公開ページ参照のファイルは誰でも閲覧可能

## 現在の課題

- 権限チェックが各コントローラーに散在

- ロールと権限の関係がハードコーディング

- 細かい権限制御には対応していない（例：読み取り専用メンバー）

- `SpaceMemberRole#permissions` と `SpaceMemberPolicy` の責務が重複
  - `SpaceMemberRole`で定義した権限と実際のチェックロジックが一致しない
  - 例：`CreateTopic`権限は定義されているが、`can_create_topic?`では権限をチェックしていない
  - 一部のメソッドは`permissions.include?`を使い、一部は独自ロジックのみで判定
  - 権限の宣言的定義と実装が乖離しており、メンテナンス性に課題

- 権限（Permission）とビジネスルールが混在している
  - UpdateTopicは「能力」を表す
  - 「トピックに参加している」は「状態」を表す
  - これらが AND 条件で結合されている

- ロールの特権を表現できない
  - 「Ownerは全トピックを編集可能」のようなルールが書きづらい
  - 各メソッドに個別にロールチェックを追加する必要がある

- SpaceとTopicの2層構造の権限管理が複雑
  - SpaceMemberRecordとTopicMemberRecordの2つの権限状態が存在
  - Space権限とTopic権限の優先順位が不明確
  - 権限の継承関係（Space Owner→Topic全権限）が暗黙的

## 要件

### ロールの権限を明確に定義する

- 各ロール（Owner、Member等）が持つ権限を一箇所で明確に定義したい
- 新しいロールや権限を追加する際の変更箇所を最小限にしたい
- 権限の定義と実際のチェックロジックを一致させたい

#### 達成条件

- ロールの権限定義を見れば、そのロールで何ができるかが明確にわかる
- ロール特有の特権（Ownerは全トピック編集可能など）を表現できる

### SpaceとTopicの2層構造の権限管理

- WikinoはGitHubのようにSpace（Organization）とTopic（Repository）の2層構造
- SpaceMemberRecordとTopicMemberRecordという2つの権限レイヤーが存在
- Space権限とTopic権限の優先順位を明確にしたい
- 権限の継承関係を明示的に表現したい
- Public/Privateの可視性制御が必要

#### 達成条件

- Space OwnerがTopic全体で持つ特権が明確
- Topic固有の権限（Topicモデレーター等）を追加可能
- Space権限とTopic権限の組み合わせによる権限判定が理解しやすい
- 新しい階層（Sub-topicなど）の追加が容易
- PublicトピックとPrivateトピックの権限制御が明確

### 権限チェックの一元化と保守性向上

- 権限チェックロジックが各コントローラーに散在している問題を解決したい
- 権限（Permission）とビジネスルール（トピック参加状態など）を明確に分離したい
- ロールごとにPolicyクラスを分離し、単一責任の原則を守りたい

#### 達成条件

- 権限チェックロジックがPolicy層に集約される
- 新しいロール追加時に既存コードへの影響が最小限
- ロール特有のバグ修正が他のロールに影響しない
- テストが書きやすく、メンテナンスしやすい構造

### 段階的な移行とリスク軽減

- 既存システムからの移行を安全に行いたい
- 現在の権限システムとの互換性を保ちながら段階的に移行したい
- GitHubモデルのベストプラクティスを取り入れたい

#### 達成条件

- 既存の`SpaceMemberPolicy`と新システムの並行稼働が可能
- Factoryパターンによりインターフェースの変更なし
- 将来的な権限レベルの細分化（Read/Write/Admin等）に対応可能
- マイグレーション時の権限変換ルールが明確

## 修正案

### ロールごとにPolicyクラスを分離する

現在の`SpaceMemberPolicy`が複数のロールの権限チェックを担当していることが複雑性の原因となっています。
これを解決するため、ロールごとに独立したPolicyクラスを作成します。

#### クラス構造

```
app/policies/
├── application_policy.rb         # 基底クラス
├── base_member_policy.rb         # メンバー共通の基底クラス
├── owner_policy.rb               # Ownerロール専用
├── member_policy.rb              # Memberロール専用
├── guest_policy.rb               # 非メンバー（ゲスト）用
└── space_member_policy_factory.rb # Policyクラスの生成
```

#### 実装例

**1. 基底クラス（共通ロジック）**

```ruby
# app/policies/base_member_policy.rb
class BaseMemberPolicy < ApplicationPolicy
  def initialize(user_record:, space_member_record:)
    @user_record = user_record
    @space_member_record = space_member_record
  end

  def joined_space?
    !space_member_record.nil?
  end

  def in_same_space?(space_id)
    space_member_record&.space_id == space_id
  end

  def active?
    space_member_record&.active?
  end
end
```

**2. Ownerロール専用Policy**

```ruby
# app/policies/owner_policy.rb
class OwnerPolicy < BaseMemberPolicy
  # Ownerは全トピックを編集可能
  def can_update_topic?(topic_record:)
    in_same_space?(topic_record.space_id)
  end

  # Ownerはスペース設定を変更可能
  def can_update_space?(space_record:)
    in_same_space?(space_record.id)
  end

  # Ownerは全ファイルを削除可能
  def can_delete_attachment?(attachment_record:)
    active? && in_same_space?(attachment_record.space_id)
  end

  # Ownerはファイル管理画面にアクセス可能
  def can_manage_attachments?(space_record:)
    active? && in_same_space?(space_record.id)
  end

  def can_export_space?(space_record:)
    in_same_space?(space_record.id)
  end
end
```

**3. Memberロール専用Policy**

```ruby
# app/policies/member_policy.rb
class MemberPolicy < BaseMemberPolicy
  # Memberは参加しているトピックのみ編集可能
  def can_update_topic?(topic_record:)
    in_same_space?(topic_record.space_id) &&
      space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  # Memberはスペース設定を変更不可
  def can_update_space?(space_record:)
    false
  end

  # Memberは自分がアップロードしたファイルのみ削除可能
  def can_delete_attachment?(attachment_record:)
    active? &&
      in_same_space?(attachment_record.space_id) &&
      space_member_record!.id == attachment_record.attached_space_member_id
  end

  # Memberはファイル管理画面にアクセス不可
  def can_manage_attachments?(space_record:)
    false
  end

  def can_export_space?(space_record:)
    false
  end
end
```

**4. ゲスト（非メンバー）用Policy**

```ruby
# app/policies/guest_policy.rb
class GuestPolicy < ApplicationPolicy
  def initialize(user_record:)
    @user_record = user_record
  end

  def can_show_page?(page_record:)
    # 公開トピックのページのみ閲覧可能
    page_record.topic_record!.visibility_public?
  end

  def can_view_attachment?(attachment_record:)
    # 公開ページで使用されているファイルのみ閲覧可能
    attachment_record.all_referencing_pages_public?
  end

  # その他の操作は全て不可
  def can_update_topic?(topic_record:)
    false
  end

  def can_update_space?(space_record:)
    false
  end
end
```

**5. Factoryパターンで適切なPolicyを生成**

```ruby
# app/policies/space_member_policy_factory.rb
class SpaceMemberPolicyFactory
  def self.build(user_record:, space_member_record: nil)
    # 非メンバーの場合
    return GuestPolicy.new(user_record:) if space_member_record.nil?

    # ロールに応じたPolicyを返す
    case space_member_record.role
    when "owner"
      OwnerPolicy.new(user_record:, space_member_record:)
    when "member"
      MemberPolicy.new(user_record:, space_member_record:)
    else
      raise ArgumentError, "Unknown role: #{space_member_record.role}"
    end
  end
end
```

#### コントローラーでの使用方法

```ruby
# 現在の実装
space_member_policy = SpaceMemberPolicy.new(
  user_record: current_user_record,
  space_member_record: current_space_member_record
)

# 新しい実装
space_member_policy = SpaceMemberPolicyFactory.build(
  user_record: current_user_record,
  space_member_record: current_space_member_record
)

# 使用方法は同じ
unless space_member_policy.can_update_space?(space_record:)
  # 権限がない場合の処理
end
```

### メリット

1. **単一責任の原則**
   - 各ロールの権限ロジックが独立したクラスに分離
   - ロール特有の振る舞いがそのクラス内に集約

2. **オープン・クローズドの原則**
   - 新しいロール追加時は新しいPolicyクラスを追加するだけ
   - 既存のロールのコードを変更する必要なし

3. **可読性の向上**
   - `OwnerPolicy`を見ればOwnerができることが一目瞭然
   - 条件分岐が減り、コードがシンプルに

4. **テスタビリティの向上**
   - ロールごとに独立してテスト可能
   - モックやスタブが簡単

5. **保守性の向上**
   - ロール特有のバグ修正が他のロールに影響しない
   - 権限の追加・変更が容易

### 段階的な移行方法

1. **Phase 1**: 新しいPolicyクラスを作成し、既存の`SpaceMemberPolicy`と並行稼働
2. **Phase 2**: コントローラーを順次新しいFactoryパターンに移行
3. **Phase 3**: 全コントローラー移行後、旧`SpaceMemberPolicy`を削除

### 懸念事項と対策

1. **共通ロジックの重複**
   - `BaseMemberPolicy`に共通メソッドを定義して解決

2. **Policyクラスの増加**
   - ロールごとに明確に分離されるため、むしろ管理しやすくなる

3. **既存コードへの影響**
   - Factoryパターンにより、インターフェースは変わらないため影響最小限

### SpaceとTopicの2層構造の権限管理

Space（ワークスペース）とTopic（チャンネル）の2層構造の権限を適切に管理するための設計案です。

#### 権限の階層構造

```
権限の優先順位：
1. Space Owner     → Space内の全権限（全Topic含む）
2. Topic Admin     → Topic内の全権限
3. Topic Member    → Topic内の基本操作権限
4. Space Member    → Space内の基本操作権限（Topic未参加）
5. Guest          → 公開コンテンツのみ閲覧
```

#### クラス構造

```
app/policies/
├── spaces/
│   ├── space_owner_policy.rb
│   ├── space_member_policy.rb
│   └── space_guest_policy.rb
├── topics/
│   ├── topic_admin_policy.rb
│   ├── topic_member_policy.rb
│   └── topic_guest_policy.rb
├── permission_resolver.rb       # 権限の優先順位を解決
└── policy_factory.rb           # 適切なPolicyを生成
```

#### 実装例

**1. 権限リゾルバー（優先順位の解決）**

```ruby
# app/policies/permission_resolver.rb
class PermissionResolver
  def initialize(user:, space:, topic: nil)
    @user = user
    @space = space
    @topic = topic
    @space_member = user&.space_members&.find_by(space:)
    @topic_member = topic ? user&.topic_members&.find_by(topic:) : nil
  end

  def resolve
    # 1. Space Ownerが最優先
    if @space_member&.owner?
      return SpaceOwnerPolicy.new(
        space_member: @space_member,
        topic_member: @topic_member
      )
    end

    # 2. Topic権限をチェック
    if @topic && @topic_member
      return build_topic_policy
    end

    # 3. Space権限をチェック
    if @space_member
      return SpaceMemberPolicy.new(space_member: @space_member)
    end

    # 4. ゲスト権限
    GuestPolicy.new(user: @user)
  end

  private

  def build_topic_policy
    case @topic_member.role
    when "admin"
      TopicAdminPolicy.new(
        topic_member: @topic_member,
        space_member: @space_member
      )
    when "member"
      TopicMemberPolicy.new(
        topic_member: @topic_member,
        space_member: @space_member
      )
    else
      TopicGuestPolicy.new(space_member: @space_member)
    end
  end
end
```

**2. Space Ownerポリシー（最高権限）**

```ruby
# app/policies/spaces/space_owner_policy.rb
class SpaceOwnerPolicy < BasePolicy
  def initialize(space_member:, topic_member: nil)
    @space_member = space_member
    @topic_member = topic_member
  end

  # Space Ownerは全トピックで全権限
  def can_update_topic?(topic:)
    topic.space_id == @space_member.space_id
  end

  def can_manage_topic_members?(topic:)
    topic.space_id == @space_member.space_id
  end

  def can_delete_any_page?(page:)
    page.space_id == @space_member.space_id
  end

  # Space管理権限
  def can_manage_space?
    true
  end

  def can_invite_space_members?
    true
  end
end
```

**3. Topic Memberポリシー（Topic参加者）**

```ruby
# app/policies/topics/topic_member_policy.rb
class TopicMemberPolicy < BasePolicy
  def initialize(topic_member:, space_member:)
    @topic_member = topic_member
    @space_member = space_member
  end

  # 参加しているTopicのページのみ編集可能
  def can_update_page?(page:)
    page.topic_id == @topic_member.topic_id &&
      @topic_member.active?
  end

  # Topic設定は変更不可（Adminのみ）
  def can_update_topic?(topic:)
    false
  end

  # 他のメンバーを招待不可（Adminのみ）
  def can_invite_to_topic?(topic:)
    false
  end

  # Topicから離脱は可能
  def can_leave_topic?
    true
  end
end
```

**4. 複合権限チェック**

```ruby
# app/policies/composite_policy.rb
class CompositePolicy
  def initialize(policies:)
    @policies = Array(policies)
  end

  # いずれかのPolicyで許可されていればtrue
  def can_view_page?(page:)
    @policies.any? { |policy| policy.can_view_page?(page:) }
  end

  # 全てのPolicyで許可されている場合のみtrue
  def can_delete_page?(page:)
    @policies.all? { |policy| policy.can_delete_page?(page:) }
  end
end
```

#### コントローラーでの使用例

```ruby
class Pages::UpdateController
  def call
    # 権限を解決
    policy = PermissionResolver.new(
      user: current_user,
      space: @page.space,
      topic: @page.topic
    ).resolve

    # 権限チェック
    unless policy.can_update_page?(page: @page)
      raise NotAuthorizedError
    end

    # ページ更新処理
    @page.update!(page_params)
  end
end
```

#### Topic固有のロール追加例

```ruby
# app/models/topic_member_role.rb
class TopicMemberRole < T::Enum
  enums do
    Admin = new("admin")        # Topic管理者
    Moderator = new("moderator") # モデレーター（新規）
    Member = new("member")       # 一般メンバー
    ReadOnly = new("read_only")  # 読み取り専用（新規）
  end
end

# app/policies/topics/topic_moderator_policy.rb
class TopicModeratorPolicy < BasePolicy
  # モデレーター固有の権限
  def can_pin_page?(page:)
    page.topic_id == @topic_member.topic_id
  end

  def can_delete_others_page?(page:)
    page.topic_id == @topic_member.topic_id
  end

  def can_mute_member?(member:)
    member.topic_id == @topic_member.topic_id
  end
end
```

### メリット

1. **権限の優先順位が明確**
   - Space Owner → Topic Admin → Topic Member の階層が明示的
   - 上位権限の特権が保証される

2. **柔軟な権限設定**
   - Topic固有のロール（モデレーター等）を簡単に追加可能
   - プライベートTopic、読み取り専用Topicなどの実装が容易

3. **関心の分離**
   - Space権限とTopic権限が独立して管理される
   - 各レイヤーの権限ロジックが分離

4. **拡張性**
   - 新しい階層（Sub-topic、Thread等）の追加が簡単
   - 既存の権限構造に影響を与えずに拡張可能

5. **テスタビリティ**
   - 各権限レイヤーを独立してテスト可能
   - 権限の組み合わせテストも容易

### GitHubモデルとの比較と実装案

WikinoはSlackよりもGitHubに近い権限モデルを採用しています。

#### モデル対応表

| Wikino            | GitHub                  | 説明                                     |
| ----------------- | ----------------------- | ---------------------------------------- |
| Space             | Organization            | 最上位の組織単位                         |
| Topic             | Repository              | プロジェクト単位、Public/Private設定可能 |
| Page              | Issue/PR/Wiki           | 個別のコンテンツ                         |
| SpaceMemberRecord | Organization Member     | 組織レベルのメンバーシップ               |
| TopicMemberRecord | Repository Collaborator | リポジトリレベルのアクセス権             |

#### GitHubライクな権限レベル

**1. Topic（Repository）の可視性**

```ruby
# app/models/topic_visibility.rb
class TopicVisibility < T::Enum
  enums do
    Public = new("public")      # 誰でも閲覧可能（GitHubのPublic Repo）
    Internal = new("internal")  # ログインユーザーのみ閲覧可能
    Private = new("private")    # メンバーのみアクセス可能（GitHubのPrivate Repo）
  end
end
```

**2. 権限レベルの細分化（GitHub風）**

```ruby
# app/models/topic_permission_level.rb
class TopicPermissionLevel < T::Enum
  enums do
    Admin = new("admin")       # フルアクセス（Settings変更可能）
    Maintain = new("maintain") # マージ、デプロイ権限相当
    Write = new("write")       # ページ作成・編集権限
    Triage = new("triage")     # ラベル付け、アサイン権限
    Read = new("read")         # 読み取り専用
  end

  sig { returns(T::Boolean) }
  def can_write?
    [Admin, Maintain, Write].include?(self)
  end

  sig { returns(T::Boolean) }
  def can_manage?
    [Admin, Maintain].include?(self)
  end
end
```

**3. GitHubライクな権限リゾルバー**

```ruby
# app/policies/github_style_permission_resolver.rb
class GithubStylePermissionResolver
  def initialize(user:, space:, topic: nil)
    @user = user
    @space = space
    @topic = topic
    @space_member = user&.space_members&.find_by(space:)
    @topic_member = topic ? user&.topic_members&.find_by(topic:) : nil
  end

  def resolve
    # Publicトピックの特別処理（GitHubのPublic Repo相当）
    if @topic&.public? && !authenticated?
      return PublicTopicGuestPolicy.new(topic: @topic)
    end

    # Organization Owner（GitHub Org Owner相当）
    if @space_member&.owner?
      return OrganizationOwnerPolicy.new(
        space_member: @space_member,
        topic: @topic
      )
    end

    # Private Topicで非Collaboratorはアクセス不可
    if @topic&.private? && !@topic_member
      return NoAccessPolicy.new
    end

    # Repository権限（GitHub Collaborator相当）
    if @topic_member
      return build_repository_policy
    end

    # Internal Topic（組織内部公開）
    if @topic&.internal? && authenticated?
      return InternalTopicViewerPolicy.new(user: @user, topic: @topic)
    end

    # Space Memberのデフォルト権限
    if @space_member
      return OrganizationMemberPolicy.new(space_member: @space_member)
    end

    # 未認証ユーザー
    AnonymousPolicy.new
  end

  private

  def build_repository_policy
    case @topic_member.permission_level
    when TopicPermissionLevel::Admin
      RepositoryAdminPolicy.new(topic_member: @topic_member)
    when TopicPermissionLevel::Maintain
      RepositoryMaintainerPolicy.new(topic_member: @topic_member)
    when TopicPermissionLevel::Write
      RepositoryWriterPolicy.new(topic_member: @topic_member)
    when TopicPermissionLevel::Triage
      RepositoryTriagerPolicy.new(topic_member: @topic_member)
    when TopicPermissionLevel::Read
      RepositoryReaderPolicy.new(topic_member: @topic_member)
    end
  end

  def authenticated?
    @user.present?
  end
end
```

**4. GitHub風のPolicy実装例**

```ruby
# app/policies/repository_admin_policy.rb
class RepositoryAdminPolicy < BasePolicy
  # GitHubのRepo Admin権限
  def can_manage_settings?
    true
  end

  def can_manage_collaborators?
    true
  end

  def can_delete_repository?
    true
  end

  def can_change_visibility?
    true  # Public/Private切り替え
  end

  def can_create_protected_branch?
    true
  end
end

# app/policies/public_topic_guest_policy.rb
class PublicTopicGuestPolicy < BasePolicy
  # GitHubのPublic Repoを未認証で見る場合
  def can_view_pages?
    true
  end

  def can_clone_repository?
    true  # Read-onlyクローン
  end

  def can_create_issue?
    @topic.allow_public_issues?  # 設定次第
  end

  def can_fork?
    false  # 認証が必要
  end

  def can_star?
    false  # 認証が必要
  end
end

# app/policies/organization_owner_policy.rb
class OrganizationOwnerPolicy < BasePolicy
  # GitHub Organization Ownerの権限
  def can_manage_all_repositories?
    true
  end

  def can_manage_billing?
    true
  end

  def can_manage_teams?
    true
  end

  def can_transfer_repository?(repository:)
    repository.space_id == @space_member.space_id
  end

  def can_create_private_repository?
    true
  end
end
```

**5. GitHub Teams相当の実装（将来拡張）**

```ruby
# app/models/space_team.rb
class SpaceTeam < ApplicationRecord
  belongs_to :space
  has_many :team_members
  has_many :team_topic_permissions

  # GitHub Teamsのような権限グループ
  enum :permission_level, {
    member: 0,
    maintainer: 1,
    admin: 2
  }
end

# app/policies/team_based_policy.rb
class TeamBasedPolicy < BasePolicy
  def initialize(user:, space:, topic:)
    @user = user
    @space = space
    @topic = topic
    @teams = user.space_teams.where(space: space)
  end

  def can_access_topic?
    # チーム単位でのトピックアクセス権限
    @teams.any? do |team|
      team.team_topic_permissions.exists?(topic: @topic)
    end
  end
end
```

#### WikinoにおけるPublic/Privateトピックの仕様

**重要**: WikinoのPrivateトピックはGitHubのPrivate Repositoryとは異なる動作をします。

**1. Publicトピック**

```ruby
# 誰でも閲覧可能（ログイン不要）
topic.visibility_public?
  → 非ログインユーザーでもページ閲覧可能
  → 添付ファイルも閲覧可能
  → 編集は不可（Spaceメンバーである必要がある）
```

**2. Privateトピック（Wikino独自仕様）**

```ruby
# Spaceメンバーなら誰でも閲覧可能
topic.visibility_private?
  → Space外のユーザーはアクセス不可
  → Spaceメンバーなら自動的にアクセス可能（招待不要）
  → TopicMemberRecordは編集権限の制御に使用
```

**GitHubとの違い**

| 項目         | GitHub Private Repo                | Wikino Private Topic                  |
| ------------ | ---------------------------------- | ------------------------------------- |
| アクセス制御 | Collaboratorへの明示的な招待が必要 | Spaceメンバーなら自動的にアクセス可能 |
| 権限管理     | Repository単位で個別管理           | Space単位で一括管理                   |
| 閲覧権限     | Collaboratorのみ                   | Spaceメンバー全員                     |
| 編集権限     | Collaboratorの権限レベルに依存     | TopicMemberRecordで制御               |

**実装上の注意点**

```ruby
# GitHubスタイル（WikinoではNG）
def can_view_private_topic?(topic:, user:)
  topic.collaborators.include?(user)  # 個別招待が必要
end

# Wikinoスタイル（正しい実装）
def can_view_private_topic?(topic:, user:)
  user.space_members.exists?(space: topic.space)  # Spaceメンバーなら自動的にOK
end
```

**TopicMemberRecordの役割**

- **GitHubでは**: Collaboratorとしての基本的なアクセス権を付与
- **Wikinoでは**: 主に編集権限の制御（閲覧はSpaceMemberで判定）

```ruby
# Wikinoの権限判定フロー
def can_update_topic?(topic:, user:)
  # 1. Space Ownerなら無条件でOK
  return true if user.space_owner?(topic.space)

  # 2. TopicMemberRecordで編集権限をチェック
  topic_member = user.topic_members.find_by(topic:)
  topic_member&.can_edit?  # 参加していて、かつ編集権限がある場合のみ
end
```

**Internal追加の必要性について**

現在のWikinoの仕様では：

- **Public**: 社外公開（誰でも閲覧可能）
- **Private**: Space内共有（≒社内共有）

この2つで基本的なユースケースはカバーできるため、Internalの追加は必須ではありません。
将来的に複数Space運用や、より細かい権限制御が必要になった場合に検討すべき拡張機能と位置づけられます。

#### GitHubモデル採用のメリット

1. **成熟した権限モデル**
   - GitHubの権限モデルは長年の実績があり、多くの開発者に馴染みがある
   - Public/Private/Internalの3段階の可視性は実用的

2. **細かい権限制御**
   - Read/Triage/Write/Maintain/Adminの5段階は多くのユースケースをカバー
   - 各権限レベルの責務が明確

3. **外部コラボレーション対応**
   - PublicトピックでのIssue作成やPull Request（将来実装）が可能
   - Fork機能（将来実装）による派生プロジェクトの作成

4. **エンタープライズ対応**
   - Teams機能による大規模組織での権限管理
   - SAML/SSOとの統合が容易（将来実装）

#### 実装における注意点

1. **段階的な実装**
   - まずはBasic権限（Owner/Member）から始める
   - 必要に応じて細分化された権限レベルを追加

2. **既存データとの互換性**
   - 現在のOwner→Admin、Member→Writeへのマッピング
   - マイグレーション時の権限変換ルール策定

3. **UI/UXの考慮**
   - GitHubライクな権限設定画面の実装
   - 権限レベルの説明とヘルプの充実

## 修正案を踏まえた修正方針 (決定方針)

### 採用する設計方針

#### 1. ロールベースのPolicyクラス分離とFactoryパターンの採用

現在の`SpaceMemberPolicy`が肥大化している問題を解決するため、ロールごとに独立したPolicyクラスに分離する設計を採用します。
これにより単一責任の原則を守り、保守性を向上させます。

**実装方針:**

- 基底クラス`BaseMemberPolicy`に共通ロジックを集約
- `OwnerPolicy`、`MemberPolicy`、`GuestPolicy`をロール別に実装
- `SpaceMemberPolicyFactory`でインターフェースの互換性を維持
- 既存コントローラーへの影響を最小化

#### 2. Space-Topic 2層構造の権限管理の明確化

WikinoのSpace（Organization相当）とTopic（Repository相当）の2層構造における権限の優先順位と継承関係を明確にします。

**権限階層の定義:**

1. Space Owner → Space内の全権限（全Topic含む）
2. Topic権限 → TopicMemberRecordによる編集権限制御
3. Space Member → Privateトピックの閲覧権限（Wikino独自仕様）
4. Guest → Publicトピックのみ閲覧可能

**重要な仕様決定:**

- **PrivateトピックはSpaceメンバー全員が閲覧可能**（GitHubと異なる）
- TopicMemberRecordは主に編集権限の制御に使用
- Space Ownerは全トピックで特権を持つ

#### 3. 権限とビジネスルールの分離

現在混在している権限（Permission）とビジネスルール（参加状態など）を明確に分離します。

**分離方針:**

- 権限: ロールに紐づく能力（UpdateSpace、CreateTopicなど）
- ビジネスルール: 状態や条件（トピック参加、アクティブ状態など）
- Policyクラス内で両者を組み合わせて最終的な判定を行う

### 段階的な移行計画

#### Phase 1: 基盤整備（互換性維持）

1. **新しいPolicyクラスの作成**
   - `app/policies/base_member_policy.rb`
   - `app/policies/owner_policy.rb`
   - `app/policies/member_policy.rb`
   - `app/policies/guest_policy.rb`
   - `app/policies/space_member_policy_factory.rb`

2. **既存`SpaceMemberPolicy`のリファクタリング**
   - 新しいFactoryを通じて適切なPolicyに処理を委譲
   - 既存のインターフェースは維持

3. **テストの整備**
   - 各Policyクラスの単体テスト作成
   - 既存テストが通ることを確認

#### Phase 2: 権限モデルの拡張

1. **権限定義の明確化**
   - `SpaceMemberPermission`を実際の権限チェックと一致させる
   - ロール特有の特権を明示的に定義

2. **Topic権限の整理**
   - TopicMemberRecordの役割を編集権限制御に特化
   - Space権限との関係を明確化

3. **権限リゾルバーの導入**
   - Space権限とTopic権限の優先順位を解決
   - 複合的な権限チェックを一元管理

#### Phase 3: コントローラーの移行

1. **段階的なコントローラー更新**
   - 重要度の低いコントローラーから順次移行
   - Factoryパターンを通じた新Policy利用

2. **権限チェックの標準化**
   - コントローラー内の権限チェックパターンを統一
   - 共通のbefore_actionやconcernの活用

#### Phase 4: 最適化と削除

1. **旧コードの削除**
   - 全コントローラー移行後、旧`SpaceMemberPolicy`を削除
   - 不要になった中間層のコードを整理

2. **パフォーマンス最適化**
   - N+1問題の解消
   - 権限チェックのキャッシュ機構導入

### 実装上の重要な考慮事項

#### 1. Wikino固有の仕様への対応

- **PrivateトピックはGitHubと異なり、Spaceメンバー全員が閲覧可能**
- TopicMemberRecordは編集権限の制御が主目的
- Space Ownerの特権は明示的に実装

#### 2. 既存システムとの互換性

- Factoryパターンにより既存のインターフェースを維持
- 段階的移行により本番環境での安全な展開が可能
- 既存のテストスイートを活用した品質保証

#### 3. 将来の拡張性

- 新しいロール（ReadOnly、Moderatorなど）の追加が容易
- GitHubライクな権限レベル（Read/Write/Admin）への移行パスを確保
- Teamsのような権限グループ機能の追加余地

### 成功指標

1. **コードの保守性向上**
   - 各ロールの権限が独立したクラスで管理される
   - 新しいロール追加時の変更箇所が最小限

2. **権限の明確性**
   - Space-Topic間の権限継承が明示的
   - ロールと権限の対応が一目瞭然

3. **システムの安定性**
   - 既存機能への影響なし
   - テストカバレッジの維持・向上

## タスクリスト

### Phase 1: 基盤整備

- [x] 基底Policyクラスの作成
  - `BaseMemberPolicy`の実装（共通メソッド: joined_space?, in_same_space?, active?）
  - `ApplicationPolicy`の作成（全Policyの基底クラス）

- [ ] ロール別Policyクラスの実装
  - `OwnerPolicy`の実装（全権限を持つ）
  - `MemberPolicy`の実装（基本操作権限のみ）
  - `GuestPolicy`の実装（公開コンテンツのみ閲覧）

- [ ] Factoryパターンの実装
  - `SpaceMemberPolicyFactory`の作成
  - ロールに応じた適切なPolicyインスタンスの返却

- [ ] テストの作成
  - 各Policyクラスの単体テスト
  - Factoryのテスト
  - 既存テストの動作確認

### Phase 2: 権限モデルの拡張

- [ ] 権限定義の整理
  - `SpaceMemberPermission`と実際の権限チェックメソッドの整合性確保
  - ロール特有の特権の明文化

- [ ] Topic権限の明確化
  - TopicMemberRecordの役割を編集権限に特化
  - PrivateトピックのWikino独自仕様の文書化

- [ ] 権限リゾルバーの実装
  - `PermissionResolver`クラスの作成
  - Space Owner > Topic権限 > Space Member > Guestの優先順位実装

### Phase 3: コントローラーの段階的移行

- [ ] 移行対象コントローラーの優先順位付け
  - リスクの低いコントローラーから開始
  - 重要な機能は後回し

- [ ] コントローラーの更新
  - SpaceMemberPolicyFactory経由での新Policy利用
  - 権限チェックパターンの統一

- [ ] 動作確認とロールバック準備
  - Feature flagの活用検討
  - 旧実装への切り戻し手順の準備

### Phase 4: 最適化とクリーンアップ

- [ ] 旧コードの削除
  - 旧`SpaceMemberPolicy`の削除
  - 不要な中間層コードの整理

- [ ] パフォーマンス最適化
  - 権限チェックのメモ化
  - データベースクエリの最適化

- [ ] ドキュメント更新
  - 権限システムの設計文書作成
  - 開発者向けガイドラインの更新
