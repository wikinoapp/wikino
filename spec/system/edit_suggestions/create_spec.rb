# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "編集提案の作成" do
  it "ページ編集画面から新しい編集提案を作成できること" do
    user_record = FactoryBot.create(:user_record, :with_confirmed_email)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :admin, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, :admin, space_member_record:, topic_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "既存のページ", body: "既存の内容")

    sign_in_as(user_record)
    visit edit_page_path(space_record.identifier, page_record.number)

    # ページ編集フォームに入力
    fill_in "pages_edit_form[title]", with: "更新後のタイトル"
    fill_in "pages_edit_form[body]", with: "# 更新後のタイトル\n\n更新された内容です。"

    # 編集提案ボタンをクリック
    click_button "編集提案する..."

    # ダイアログが表示されることを確認
    expect(page).to have_css("dialog[open]")
    within("dialog") do
      fill_in "edit_suggestions_create_form[title]", with: "ページ更新の提案"
      fill_in "edit_suggestions_create_form[description]", with: "内容を更新しました"
      click_button "編集提案を作成"
    end

    # 編集提案詳細ページにリダイレクトされることを確認
    expect(page).to have_content("編集提案を作成しました")
    expect(page).to have_current_path(%r{/s/#{space_record.identifier}/topics/\d+/edit_suggestions/\d+})
    expect(page).to have_content("ページ更新の提案")
    expect(page).to have_content("内容を更新しました")
  end

  it "既存の編集提案にページを追加できること" do
    user_record = FactoryBot.create(:user_record, :with_confirmed_email)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :admin, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, :admin, space_member_record:, topic_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "既存のページ", body: "既存の内容")

    # 既存の編集提案を作成
    existing_edit_suggestion = FactoryBot.create(
      :edit_suggestion_record,
      :draft,
      space_record:,
      topic_record:,
      created_space_member_record: space_member_record,
      title: "既存の編集提案"
    )

    sign_in_as(user_record)
    visit edit_page_path(space_record.identifier, page_record.number)

    # ページ編集フォームに入力
    fill_in "pages_edit_form[title]", with: "更新後のタイトル"
    fill_in "pages_edit_form[body]", with: "# 更新後のタイトル\n\n更新された内容です。"

    # 編集提案ボタンをクリック
    click_button "編集提案する..."

    # ダイアログが表示されることを確認
    expect(page).to have_css("dialog[open]")
    within("dialog") do
      # 既存の編集提案を選択
      choose "existing_edit_suggestion_id_existing"
      choose "edit_suggestions_create_form_existing_edit_suggestion_id_#{existing_edit_suggestion.id}"
      click_button "編集提案を作成"
    end

    # 編集提案詳細ページにリダイレクトされることを確認
    expect(page).to have_content("編集提案を作成しました")
    expect(page).to have_current_path(edit_suggestion_path(space_record.identifier, topic_record.number, existing_edit_suggestion.id))
    expect(page).to have_content("既存の編集提案")
  end
end
