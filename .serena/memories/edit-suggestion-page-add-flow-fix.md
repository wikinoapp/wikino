# 編集提案ページ追加フローの修正

## 修正内容
2025年1月24日に「既存に追加」フローの実装を修正しました。

### 問題点
- タブクリック時点で編集提案IDが必要だったため、エンドポイントが機能しなかった
- `/edit_suggestions/:id/pages/new` の形式では、タブ表示時にIDが確定していない

### 解決策
1. エンドポイントをIDなしに変更
   - GET `/edit_suggestion_pages/new`
   - POST `/edit_suggestion_pages`

2. セレクトボックスによる編集提案選択
   - 既存の編集提案をセレクトボックスで表示
   - ユーザーが選択してからページを追加

3. タブUIの簡素化
   - 「新規作成」「既存に追加」の2つのタブのみ
   - 個別の編集提案タブは表示しない

## 修正したファイル
- config/routes.rb
- app/controllers/edit_suggestion_pages/new_controller.rb
- app/controllers/edit_suggestion_pages/create_controller.rb
- app/forms/edit_suggestion_pages/create_form.rb
- app/views/edit_suggestion_pages/new_view.rb
- app/views/edit_suggestion_pages/new_view.html.erb
- app/components/edit_suggestions/form_modal_component.rb
- app/components/edit_suggestions/form_modal_component.html.erb
- config/locales/forms.ja.yml
- config/locales/forms.en.yml