# 編集提案機能

## 概要

編集提案機能は、Wikinoのトピック内でページの作成や既存ページの編集を提案できる機能です。
GitHubのPull Requestsのような形で、公開トピックであれば誰でも編集を提案でき、承認後に反映される仕組みを提供します。

## 要件

### 基本要件

- 編集提案は1つのトピック内のページに限定される
- 公開トピックであれば、スペース非参加者でも編集提案を作成可能
- 編集提案作成者が退会しても編集提案は保持される

### ステータス管理

編集提案には以下の4つのステータスを設ける：

1. **下書き** - 作成中の編集提案
2. **オープン** - レビュー待ちの編集提案
3. **反映済み** - トピックに反映された編集提案
4. **クローズ** - 反映されずに閉じられた編集提案

### UI要件

#### トピックページ

- トピックページ上部に「ページ」「編集提案」のタブを表示
- GitHubのCode/Pull requestsタブと同様のUI

#### 編集提案一覧ページ

- ユーザーが作成した編集提案をリスト表示
- ステータスでフィルタリング可能

#### 編集提案詳細ページ

- 「会話」「編集したページ」の2つのタブ
- デフォルトは「会話」タブがアクティブ
- 「会話」タブ：編集提案の概要とコメント表示
- 「編集したページ」タブ：変更差分の表示
- 「反映する」ボタン（権限がある場合）

#### ページ編集画面

- 「保存する」ボタンの横に「編集提案する...」ボタンを配置
- クリック時にモーダル表示：
  - 「新しい編集提案を作成する」（デフォルト選択）
    - タイトル入力フィールド
    - 概要入力フィールド
  - 「既存の編集提案に加える」
    - オープン/下書きの編集提案一覧から選択

## タスクリスト

### データベース設計

- [ ] 編集提案テーブル（edit_suggestions）の作成
  - id, topic_record_id, created_by_user_record_id, title, description, status, created_at, updated_at
- [ ] 編集提案ページテーブル（edit_suggestion_pages）の作成
  - id, edit_suggestion_record_id, page_record_id, action (create/update), content_before, content_after
- [ ] 編集提案コメントテーブル（edit_suggestion_comments）の作成
  - id, edit_suggestion_record_id, user_record_id, content, created_at, updated_at

### モデル層

- [ ] EditSuggestionRecordクラスの作成
- [ ] EditSuggestionPageRecordクラスの作成
- [ ] EditSuggestionCommentRecordクラスの作成
- [ ] EditSuggestionモデルクラスの作成
- [ ] EditSuggestionRepositoryクラスの作成

### サービス層

- [ ] EditSuggestions::CreateServiceの実装
- [ ] EditSuggestions::UpdateServiceの実装
- [ ] EditSuggestions::ApplyServiceの実装（編集提案の反映）
- [ ] EditSuggestions::CloseServiceの実装
- [ ] EditSuggestionPages::AddServiceの実装
- [ ] EditSuggestionComments::CreateServiceの実装

### コントローラー層

- [ ] EditSuggestions::IndexControllerの実装
- [ ] EditSuggestions::ShowControllerの実装
- [ ] EditSuggestions::CreateControllerの実装
- [ ] EditSuggestions::UpdateControllerの実装
- [ ] EditSuggestions::ApplyControllerの実装
- [ ] EditSuggestionPages::CreateControllerの実装
- [ ] EditSuggestionComments::CreateControllerの実装

### ビュー・コンポーネント層

- [ ] Topics::TabsComponentの作成（ページ/編集提案タブ）
- [ ] EditSuggestions::ListComponentの作成
- [ ] EditSuggestions::DetailComponentの作成
- [ ] EditSuggestions::TabsComponentの作成（会話/編集したページタブ）
- [ ] EditSuggestions::DiffViewComponentの作成
- [ ] EditSuggestions::ConversationComponentの作成
- [ ] EditSuggestions::CreateModalComponentの作成

### フロントエンド（Stimulus）

- [ ] edit-suggestion-modal-controllerの実装
- [ ] edit-suggestion-tabs-controllerの実装
- [ ] diff-view-controllerの実装

### ポリシー層

- [ ] EditSuggestionPolicyの実装
  - 作成権限の確認
  - 反映権限の確認
  - 閲覧権限の確認

### ルーティング

- [ ] 編集提案一覧：GET /s/:space_identifier/topics/:topic_number/edit-suggestions
- [ ] 編集提案詳細：GET /s/:space_identifier/topics/:topic_number/edit-suggestions/:id
- [ ] 編集提案作成：POST /s/:space_identifier/topics/:topic_number/edit-suggestions
- [ ] 編集提案更新：PATCH /s/:space_identifier/topics/:topic_number/edit-suggestions/:id
- [ ] 編集提案反映：POST /s/:space_identifier/topics/:topic_number/edit-suggestions/:id/apply
- [ ] コメント追加：POST /s/:space_identifier/topics/:topic_number/edit-suggestions/:id/comments

### テスト

- [ ] モデルのユニットテスト
- [ ] サービスのユニットテスト
- [ ] コントローラーのリクエストスペック
- [ ] システムテスト（E2Eテスト）
- [ ] ポリシーのテスト

### その他

- [ ] 既存のページ編集画面への「編集提案する...」ボタン追加
- [ ] 通知機能との連携（編集提案へのコメント、反映時など）
- [ ] アクティビティログへの記録
- [ ] I18n対応（日本語・英語）
