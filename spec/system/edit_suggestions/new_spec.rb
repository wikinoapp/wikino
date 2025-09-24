# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "EditSuggestions::New", type: :system do
  it "編集提案の新規作成フォームを表示すること" do
    space_record = FactoryBot.create(:space_record)
    user_record = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)

    sign_in(user_record:)

    # ページを作成
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "テストページ", body: "テスト内容")

    # Turbo Frameでフォームを取得
    visit new_edit_suggestion_path(
      space_identifier: space_record.identifier,
      page_number: page_record.number
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
    user_record = FactoryBot.create(:user_record, :with_password)
    FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "既存ページ", body: "既存内容")

    sign_in(user_record:)

    # 直接新規編集提案フォームページにアクセス
    visit new_edit_suggestion_path(
      space_identifier: space_record.identifier,
      page_number: page_record.number
    )

    # フォームに入力
    within("turbo-frame#edit-suggestion-form") do
      fill_in "edit_suggestions_create_form[title]", with: "新しい編集提案"
      fill_in "edit_suggestions_create_form[description]", with: "テストの編集提案です"

      click_button "作成する"
    end

    # 編集提案一覧ページにリダイレクトされることを確認（ShowControllerが未実装のため）
    expect(page).to have_current_path(topic_edit_suggestion_list_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number
    ))

    # 作成した編集提案が表示されることを確認
    expect(page).to have_content("新しい編集提案")
    expect(page).to have_content("下書き")
  end
end
