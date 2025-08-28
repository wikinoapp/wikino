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

## 要件

### (要件1)

- (要件について)
- (箇条書きで)
- (書く)

#### 達成条件

- (要件の達成条件について)
- (箇条書きで)
- (書く)

## タスクリスト

- [ ] (タスク1)
  - (サブタスクの詳細)

- [ ] (タスク2)

- [ ] (タスク3)
