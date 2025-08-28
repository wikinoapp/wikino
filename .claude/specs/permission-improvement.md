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

## 要件

### ロールの権限を明確に定義する

- 各ロール（Owner、Member等）が持つ権限を一箇所で明確に定義したい
- 新しいロールや権限を追加する際の変更箇所を最小限にしたい
- 権限の定義と実際のチェックロジックを一致させたい

#### 達成条件

- ロールの権限定義を見れば、そのロールで何ができるかが明確にわかる
- ロール特有の特権（Ownerは全トピック編集可能など）を表現できる

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

## タスクリスト

- [ ] (タスク1)
  - (サブタスクの詳細)

- [ ] (タスク2)

- [ ] (タスク3)
