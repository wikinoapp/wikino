# typed: false
# frozen_string_literal: true

RSpec.describe "Topics::Show", type: :system do
  it "トピック詳細ページでタブが正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:, name: "テストトピック")
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    # タブナビゲーションが表示されることを確認
    expect(page).to have_css("nav")

    # 「ページ」タブが表示されることを確認
    expect(page).to have_link("ページ", href: "/s/#{space_record.identifier}/topics/#{topic_record.number}")

    # 「編集提案」タブが表示されることを確認
    expect(page).to have_link("編集提案", href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions")

    # 「ページ」タブがアクティブになっていることを確認
    pages_link = page.find_link("ページ")
    expect(pages_link[:class]).to include("border-primary-500")
    expect(pages_link[:class]).to include("text-primary-600")

    # 「編集提案」タブが非アクティブになっていることを確認
    edit_suggestions_link = page.find_link("編集提案")
    expect(edit_suggestions_link[:class]).to include("border-transparent")
    expect(edit_suggestions_link[:class]).to include("text-gray-500")
  end

  it "トピックページから編集提案タブへの遷移が動作すること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    # 編集提案タブをクリック
    click_link "編集提案"

    # 編集提案一覧ページに遷移することを確認
    expect(page).to have_current_path("/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions")
  end

  it "ゲストユーザーでもタブが表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, visibility: "public")

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    # タブナビゲーションが表示されることを確認
    expect(page).to have_css("nav")
    expect(page).to have_link("ページ")
    expect(page).to have_link("編集提案")
  end
end
