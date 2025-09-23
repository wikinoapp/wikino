# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "EditSuggestions::New", type: :system do
  it "編集提案の新規作成フォームを表示すること" do
    space_record = FactoryBot.create(:space_record)
    user_record = FactoryBot.create(:user_record, :activated, password: "passw0rd")
    FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)

    sign_in(user_record:)

    # Turbo Frameでフォームを取得
    visit new_edit_suggestion_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number,
      page_title: "テストページ",
      page_body: "テスト内容"
    )

    # フォームが表示されることを確認
    within("turbo-frame#edit-suggestion-form") do
      expect(page).to have_field("edit_suggestions_create_form[title]")
      expect(page).to have_field("edit_suggestions_create_form[description]")
      expect(page).to have_field("edit_suggestions_create_form[page_title]", type: :hidden, with: "テストページ")
      expect(page).to have_field("edit_suggestions_create_form[page_body]", type: :hidden, with: "テスト内容")
    end
  end

  it "新規編集提案を作成できること" do
    space_record = FactoryBot.create(:space_record)
    user_record = FactoryBot.create(:user_record, :activated, password: "passw0rd")
    FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)

    sign_in(user_record:)

    # ページ編集画面を開く
    visit edit_page_path(space_identifier: space_record.identifier, page_number: nil)

    # ページ内容を入力
    fill_in "pages_update_form[title]", with: "新しいページ"

    # CodeMirrorエディタに値を設定
    execute_script("document.querySelector('.cm-editor').CodeMirror.setValue('新しい内容')")

    # 編集提案ダイアログを開く
    click_button "編集提案する..."

    # ダイアログが表示されるまで待つ
    expect(page).to have_css("dialog[open]")

    within("dialog") do
      # 新規作成タブがアクティブなことを確認（将来実装）
      # フォームに入力
      fill_in "edit_suggestions_create_form[title]", with: "新しい編集提案"
      fill_in "edit_suggestions_create_form[description]", with: "テストの編集提案です"

      click_button "編集提案を作成"
    end

    # 編集提案詳細ページにリダイレクトされることを確認
    expect(page).to have_current_path(edit_suggestion_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number,
      id: EditSuggestionRecord.last.id
    ))

    # フラッシュメッセージを確認
    expect(page).to have_content("編集提案を作成しました")
  end
end
