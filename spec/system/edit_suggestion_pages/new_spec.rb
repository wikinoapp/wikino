# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "EditSuggestionPages::New", type: :system do
  it "既存編集提案へのページ追加フォームを表示すること" do
    space_record = FactoryBot.create(:space_record)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space_record:,
      topic_record:,
      created_space_member_record: space_member_record,
      status: "draft",
      title: "既存の編集提案")

    sign_in(user_record:)

    # Turbo Frameでフォームを取得
    visit new_edit_suggestion_page_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number,
      id: edit_suggestion_record.id,
      page_title: "テストページ",
      page_body: "テスト内容"
    )

    # フォームが表示されることを確認
    within("turbo-frame#edit-suggestion-form") do
      expect(page).to have_content("既存の編集提案")
      expect(page).to have_field("edit_suggestion_pages_create_form[page_title]", type: :hidden, with: "テストページ")
      expect(page).to have_field("edit_suggestion_pages_create_form[page_body]", type: :hidden, with: "テスト内容")
    end
  end

  it "既存編集提案にページを追加できること" do
    space_record = FactoryBot.create(:space_record)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, space_record:, user_record:, role: "owner")
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:edit_suggestion_record,
      space_record:,
      topic_record:,
      created_space_member_record: space_member_record,
      status: "draft",
      title: "既存の編集提案",
      description: "テストの説明")

    sign_in(user_record:)

    # 直接既存編集提案へのページ追加フォームページにアクセス
    visit new_edit_suggestion_page_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number,
      page_title: "追加ページ",
      page_body: "追加内容"
    )

    # フォームが表示されることを確認
    within("turbo-frame#edit-suggestion-form") do
      # 既存の編集提案を選択
      select "既存の編集提案", from: "edit_suggestion_pages_create_form[edit_suggestion_id]"

      click_button "ページを追加"
    end

    # 編集提案一覧ページにリダイレクトされることを確認（ShowControllerが未実装のため）
    expect(page).to have_current_path(topic_edit_suggestion_list_path(
      space_identifier: space_record.identifier,
      topic_number: topic_record.number
    ))

    # 編集提案が表示されることを確認
    expect(page).to have_content("既存の編集提案")
  end
end
