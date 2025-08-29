# Topic権限の設計仕様

## 概要

WikinoのTopic権限は、Space（ワークスペース）とTopic（チャンネル）の2層構造で管理されています。
この文書では、Topic権限の仕様と実装方針を明確化します。

## 権限階層の優先順位

```
1. Space Owner     → Space内の全権限（全Topic含む）
2. Topic Member    → TopicMemberRecordで編集権限を制御
3. Space Member    → Privateトピックの閲覧権限
4. Guest          → Publicトピックのみ閲覧可能
```

## TopicMemberRecordの役割

### 設計方針

TopicMemberRecordは**編集権限の制御**に特化したレコードとして位置付けられます。

- **主な役割**: トピック内でのページ作成・編集権限の管理
- **ロール定義**:
  - `Admin`: トピック管理者（将来的な拡張用）
  - `Member`: トピックメンバー（編集権限あり）

### 権限判定フロー

```ruby
# MemberPolicyでの実装例
def can_update_topic?(topic_record:)
  # Space Memberであることが前提条件
  in_same_space?(space_record_id: topic_record.space_id) &&
    # TopicMemberRecordが存在する（= 編集権限がある）
    joined_topic?(topic_record_id: topic_record.id)
end
```

## Wikino独自仕様：PrivateトピックのアクセスルKール

### GitHubとの違い

| 項目         | GitHub Private Repo                | Wikino Private Topic                  |
| ------------ | ---------------------------------- | ------------------------------------- |
| アクセス制御 | Collaboratorへの明示的な招待が必要 | Spaceメンバーなら自動的にアクセス可能 |
| 権限管理     | Repository単位で個別管理           | Space単位で一括管理                   |
| 閲覧権限     | Collaboratorのみ                   | Spaceメンバー全員                     |
| 編集権限     | Collaboratorの権限レベルに依存     | TopicMemberRecordで制御               |

### 実装上の重要ポイント

1. **Privateトピックの閲覧権限**

   ```ruby
   # MemberPolicyでの実装
   def can_show_page?(page_record:)
     # SpaceMemberであれば、Privateトピックのページも閲覧可能
     active? && in_same_space?(space_record_id: page_record.space_id)
   end
   ```

2. **Privateトピックの編集権限**
   ```ruby
   def can_update_page?(page_record:)
     # 編集にはTopicMemberRecordが必要
     active? && joined_topic?(topic_record_id: page_record.topic_id)
   end
   ```

## Topic Visibilityの定義

```ruby
# app/models/topic_visibility.rb
class TopicVisibility < T::Enum
  enums do
    Public = new("public")    # 誰でも閲覧可能
    Private = new("private")   # Spaceメンバーのみアクセス可能
  end
end
```

### Publicトピック

- 非ログインユーザーでも閲覧可能
- 添付ファイルも閲覧可能
- 編集にはSpaceMember + TopicMemberが必要

### Privateトピック

- Space外のユーザーはアクセス不可
- **Spaceメンバーなら自動的に閲覧可能**（招待不要）
- 編集にはTopicMemberRecordが必要

## 権限チェックの実装パターン

### 1. Space Ownerの特権

```ruby
# OwnerPolicy
def can_update_topic?(topic_record:)
  # Space Ownerは無条件で全トピックを編集可能
  in_same_space?(space_record_id: topic_record.space_id)
end
```

### 2. Space Memberの権限

```ruby
# MemberPolicy
def can_update_topic?(topic_record:)
  # TopicMemberRecordが必要
  in_same_space?(space_record_id: topic_record.space_id) &&
    joined_topic?(topic_record_id: topic_record.id)
end

def can_show_page?(page_record:)
  # 閲覧はSpaceMemberなら可能
  active? && in_same_space?(space_record_id: page_record.space_id)
end
```

### 3. Guestの権限

```ruby
# GuestPolicy
def can_show_page?(page_record:)
  # Publicトピックのページのみ閲覧可能
  page_record.topic_record!.visibility_public?
end
```

## 今後の拡張性

### 将来的に追加可能な機能

1. **Topic固有のロール**
   - Moderator: コンテンツ管理権限
   - ReadOnly: 読み取り専用メンバー

2. **権限レベルの細分化**
   - Read: 閲覧のみ
   - Write: ページ作成・編集
   - Admin: トピック設定変更

3. **Teams機能**
   - Space内でのグループ単位の権限管理
   - 複数トピックへの一括権限付与

## まとめ

WikinoのTopic権限システムは、GitHubモデルを参考にしつつ、独自の仕様を採用しています：

1. **TopicMemberRecordは編集権限の制御に特化**
2. **PrivateトピックはSpaceメンバー全員が閲覧可能**（GitHubと異なる）
3. **Space Ownerは全トピックで特権を持つ**

この設計により、シンプルながら柔軟な権限管理を実現しています。
