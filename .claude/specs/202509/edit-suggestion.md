# 編集提案機能

## 概要

編集提案機能は、Wikinoのトピック内でページの作成や既存ページの編集を提案できる機能です。
GitHubのPull Requestsのような形で、スペースメンバーが編集を提案し、承認後に反映される仕組みを提供します。

## 要件

### 基本要件

- 編集提案は1つのトピック内のページに限定される
- 編集提案の作成はスペースメンバーのみ可能（公開トピックでもスペース参加が必要）
- 編集提案作成者がスペースから退会しても編集提案は保持される

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
- GitHubのようにオープン/クローズで絞り込み可能
  - オープン表示：下書き・オープンステータスの編集提案
  - クローズ表示：反映済み・クローズステータスの編集提案

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

### 1. データベース基盤の構築

- [ ] マイグレーションファイルの作成
  - 編集提案テーブル (`edit_suggestions`)
    - id, space_id, topic_id, created_user_id, title, description, status, applied_at, created_at, updated_at
  - 編集提案ページテーブル (`edit_suggestion_pages`)
    - id, space_id, edit_suggestion_id, page_id, title_before, title_after, body_before, body_after
  - 編集提案コメントテーブル (`edit_suggestion_comments`)
    - id, space_id, edit_suggestion_id, created_user_id, body, body_html, created_at, updated_at
- [ ] レコードクラスの作成
  - EditSuggestionRecord
  - EditSuggestionPageRecord
  - EditSuggestionCommentRecord
- [ ] モデル・リポジトリの作成
  - EditSuggestionモデル
  - EditSuggestionRepository
- [ ] ポリシーの実装
  - SpaceMemberPolicy、TopicMemberPolicyなどの既存ポリシーに編集提案関連の権限を追加
- [ ] テスト作成
  - レコードのFactoryBot定義
  - モデルのユニットテスト
  - ポリシーのテスト

### 2. トピックページへのタブ追加

- [ ] Topics::TabsComponentの作成
- [ ] Topics::ShowControllerの修正（タブ表示対応）
- [ ] ルーティングの追加
  - GET /s/:space_identifier/topics/:topic_number/edit_suggestions
- [ ] テスト作成
  - コンポーネントのテスト
  - システムテスト（タブ表示確認）

### 3. 編集提案一覧画面

- [ ] EditSuggestions::IndexControllerの実装
- [ ] EditSuggestions::IndexViewの実装
- [ ] EditSuggestions::ListComponentの作成
- [ ] オープン/クローズのフィルタリング機能
- [ ] テスト作成
  - コントローラーのリクエストスペック
  - システムテスト（一覧表示・フィルタリング）

### 4. 編集提案作成機能（ページ編集画面から）

- [ ] EditSuggestions::CreateServiceの実装
- [ ] EditSuggestions::CreateControllerの実装
- [ ] EditSuggestions::CreateFormの実装
- [ ] EditSuggestions::CreateModalComponentの作成
- [ ] edit-suggestion-modal-controllerの実装（Stimulus）
- [ ] Pages::EditControllerの修正（モーダル追加）
- [ ] ルーティングの追加
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（モーダル表示・作成）

### 5. 編集提案詳細画面（会話タブ）

- [ ] EditSuggestions::ShowControllerの実装
- [ ] EditSuggestions::ShowViewの実装
- [ ] EditSuggestions::DetailComponentの作成
- [ ] EditSuggestions::TabsComponentの作成
- [ ] EditSuggestions::ConversationComponentの作成
- [ ] edit-suggestion-tabs-controllerの実装（Stimulus）
- [ ] ルーティングの追加
  - GET /s/:space_identifier/topics/:topic_number/edit_suggestions/:id
- [ ] テスト作成
  - コントローラーのリクエストスペック
  - システムテスト（詳細表示・タブ切り替え）

### 6. 編集提案詳細画面（編集したページタブ）

- [ ] EditSuggestions::DiffViewComponentの作成
- [ ] diff-view-controllerの実装（Stimulus）
- [ ] 差分表示のスタイリング
- [ ] テスト作成
  - コンポーネントのテスト
  - システムテスト（差分表示）

### 7. 編集提案へのコメント機能

- [ ] EditSuggestionComments::CreateServiceの実装
- [ ] EditSuggestionComments::CreateControllerの実装
- [ ] EditSuggestionComments::CreateFormの実装
- [ ] コメント表示コンポーネントの作成
- [ ] ルーティングの追加
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/comments
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（コメント投稿・表示）

### 8. 編集提案のステータス変更機能

- [ ] EditSuggestions::OpenServiceの実装
- [ ] EditSuggestions::ConvertToDraftServiceの実装
- [ ] EditSuggestions::CloseServiceの実装
- [ ] EditSuggestionOpens::CreateControllerの実装
- [ ] EditSuggestionDrafts::CreateControllerの実装
- [ ] EditSuggestionClosures::CreateControllerの実装
- [ ] ステータス変更ボタンの追加
- [ ] ルーティングの追加
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/open
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/draft
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/close
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（ステータス変更）

### 9. 編集提案の反映機能

- [ ] EditSuggestions::ApplyServiceの実装
- [ ] EditSuggestionApplications::CreateControllerの実装
- [ ] 反映確認モーダルの作成
- [ ] ルーティングの追加
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/apply
- [ ] テスト作成
  - サービスのユニットテスト（反映処理）
  - システムテスト（反映確認・実行）

### 10. 編集提案の編集機能

- [ ] EditSuggestions::UpdateServiceの実装
- [ ] EditSuggestions::EditControllerの実装
- [ ] EditSuggestions::UpdateControllerの実装
- [ ] EditSuggestions::EditFormの実装
- [ ] 編集画面の作成
- [ ] ルーティングの追加
  - GET /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/edit
  - PATCH /s/:space_identifier/topics/:topic_number/edit_suggestions/:id
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（編集画面表示・更新）

### 11. 既存の編集提案へのページ追加機能

- [ ] EditSuggestionPages::AddServiceの実装
- [ ] EditSuggestionPages::CreateControllerの実装
- [ ] 既存編集提案選択UIの実装
- [ ] ルーティングの追加
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/pages
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（ページ追加）

### 12. 編集提案からのページ削除機能

- [ ] EditSuggestionPages::RemoveServiceの実装
- [ ] EditSuggestionPages::DestroyControllerの実装
- [ ] 削除確認UIの実装
- [ ] ルーティングの追加
  - DELETE /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/pages/:page_id
- [ ] テスト作成
  - サービスのユニットテスト
  - システムテスト（ページ削除）

### 13. その他の機能

- [ ] 通知機能との連携
  - コメント通知
  - ステータス変更通知
  - 反映通知
- [ ] アクティビティログへの記録
- [ ] I18n対応（日本語・英語）
