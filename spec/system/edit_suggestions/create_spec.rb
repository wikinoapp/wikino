# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "編集提案の作成" do
  it "ページ編集画面から新しい編集提案を作成できること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :owner, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, :admin, space_member_record:, topic_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "既存のページ", body: "既存の内容")

    sign_in(user_record:)

    # 直接新規編集提案フォームページにアクセス
    visit new_edit_suggestion_path(space_record.identifier, page_record.number)

    # フォームが表示されることを確認
    expect(page).to have_field("edit_suggestions_create_form[title]")
    expect(page).to have_field("edit_suggestions_create_form[description]")
    # page_titleとpage_bodyは隠しフィールドなので、visible: :hiddenで確認
    expect(page).to have_field("edit_suggestions_create_form[page_title]", type: :hidden, with: "既存のページ")
    expect(page).to have_field("edit_suggestions_create_form[page_body]", type: :hidden, with: "既存の内容")

    # フォームに入力
    fill_in "edit_suggestions_create_form[title]", with: "ページ更新の提案"
    fill_in "edit_suggestions_create_form[description]", with: "内容を更新しました"

    # フォームを送信
    click_button "作成する"

    # 編集提案一覧ページにリダイレクトされることを確認（ShowControllerが未実装のため）
    expect(page).to have_current_path(topic_edit_suggestion_list_path(space_record.identifier, topic_record.number))

    # 作成した編集提案が一覧に表示されることを確認
    expect(page).to have_content("ページ更新の提案")
    expect(page).to have_content("下書き")  # デフォルトステータスは下書き
  end

  it "既存の編集提案にページを追加できること", js: true do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :owner, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, :admin, space_member_record:, topic_record:)

    # ページを作成
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "既存のページ", body: "既存の内容")
    page_revision_record = FactoryBot.create(:page_revision_record, page_record:, body: page_record.body)

    # 既存の編集提案を作成
    existing_edit_suggestion = FactoryBot.create(
      :edit_suggestion_record,
      :draft,
      space_record:,
      topic_record:,
      created_space_member_record: space_member_record,
      title: "既存の編集提案"
    )

    sign_in(user_record:)

    # ページ編集画面にアクセス
    visit edit_page_path(
      space_identifier: space_record.identifier,
      page_number: page_record.number
    )

    # ページがロードされるまで待つ
    expect(page).to have_button("保存する")

    # textareaに直接入力（hiddenフィールドなのでJavaScriptで操作）
    page.execute_script(<<~JS)
      const textarea = document.querySelector('textarea[name="pages_edit_form[body]"]');
      textarea.value = "編集後の内容";
      textarea.dispatchEvent(new Event('input', { bubbles: true }));
    JS

    # フォームの変更が反映されるまで少し待つ
    sleep 0.5

    # 「編集提案する...」ボタンをクリック
    click_button "編集提案する..."

    # ダイアログが表示されるまで待つ
    expect(page).to have_css("[role=dialog]")

    # ダイアログ内で既存の編集提案に追加
    within("[role=dialog]") do
      # 既存の編集提案に追加タブをクリック
      click_link "既存の提案に追加"

      # フォームが表示されることを確認
      expect(page).to have_select("edit_suggestion_pages_create_form[edit_suggestion_id]")

      # 既存の編集提案を選択
      select existing_edit_suggestion.title, from: "edit_suggestion_pages_create_form[edit_suggestion_id]"

      # フォームを送信
      click_button "ページを追加"
    end

    # 編集提案一覧ページにリダイレクトされることを確認（ShowControllerが未実装のため）
    expect(page).to have_current_path(topic_edit_suggestion_list_path(space_record.identifier, topic_record.number))
    # 既存の編集提案が一覧に表示されることを確認
    expect(page).to have_content("既存の編集提案")
    expect(page).to have_content("下書き")
  end
end
