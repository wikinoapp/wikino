# 編集提案機能 - ブランチ edit-suggestion-4 の進捗状況

## ブランチ情報
- ブランチ名: `edit-suggestion-4`
- ベースブランチ: `edit-suggestion`
- 作業内容: 編集提案作成機能（ページ編集画面から）の実装

## 完了済みタスク（edit-suggestionブランチとの差分）

### 1. 基本コンポーネント
- **BaseUi::DialogComponent**: Basecoat UIベースのダイアログコンポーネント
- **dialog-controller**: Stimulusコントローラー（ダイアログのopen/close制御）

### 2. 編集提案作成機能
- **EditSuggestions::CreateService**: 新規編集提案作成サービス
- **EditSuggestionPages::AddService**: 既存編集提案へのページ追加サービス
- **EditSuggestions::CreateController**: 編集提案作成コントローラー
  - POST /s/:space_identifier/topics/:topic_number/edit_suggestions
  - 新規作成と既存への追加の両方に対応（現在の実装）
- **EditSuggestions::CreateForm**: 編集提案作成フォーム
  - タイトル、概要、ページタイトル、ページ本文
  - 既存編集提案選択のサポート
- **EditSuggestions::CreateModalComponent**: 編集提案作成モーダル
  - 「新しい編集提案を作成する」（デフォルト）
  - 「既存の編集提案に加える」（オープン/下書きから選択）

### 3. 既存画面の修正
- **Pages::EditController**: 編集提案ダイアログ用のデータ準備
- **Pages::EditView**: 「編集提案する...」ボタンとダイアログの追加

### 4. テスト
- EditSuggestions::CreateServiceのユニットテスト
- EditSuggestionPages::AddServiceのユニットテスト
- EditSuggestions::Createのシステムテスト

## 今後の改善予定

### Turbo Frame対応への移行
現在はダイアログ内で通常のフォーム送信を行っているが、より良いUXのためにTurbo Frameを使用した実装への移行を検討中：

#### 新規作成フロー
1. **EditSuggestions::NewController**
   - GET /s/:space_identifier/topics/:topic_number/edit_suggestions/new
   - Turbo Frameで新規編集提案フォームを返す

2. **EditSuggestions::CreateController（再実装）**
   - POST /s/:space_identifier/topics/:topic_number/edit_suggestions
   - 新規編集提案の作成専用エンドポイントとして再実装
   - Turbo Stream対応でエラー表示を改善

#### 既存に追加フロー
3. **EditSuggestionPages::NewController**
   - GET /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/pages/new
   - Turbo Frameで既存編集提案へのページ追加フォームを返す

4. **EditSuggestionPages::CreateController**
   - POST /s/:space_identifier/topics/:topic_number/edit_suggestions/:id/pages
   - 既存編集提案へのページ追加専用エンドポイント
   - Turbo Stream対応でエラー表示を改善

#### UI実装
5. **ダイアログ内でのタブ切り替え**
   - 「新規作成」「既存に追加」のタブを通常のリンクとして実装
   - タブクリック時にTurbo Frameで適切なフォームを動的に読み込み

## 注意事項
- 現在の実装は動作しており、テストも通過している
- 現在のEditSuggestions::CreateControllerは両方のケースを処理しているが、将来的には分離予定
- Turbo Frame対応は将来的な改善として位置付けられている
- 基本機能は完成しているため、次のタスク（編集提案詳細画面など）に進むことも可能