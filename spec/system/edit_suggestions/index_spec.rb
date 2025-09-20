# typed: false
# frozen_string_literal: true

RSpec.describe "EditSuggestions::Index", type: :system do
  it "編集提案一覧ページでタブが正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:, name: "テストトピック")
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # タブナビゲーションが表示されることを確認
    expect(page).to have_css("nav")

    # 「ページ」タブが表示されることを確認
    expect(page).to have_link("ページ", href: "/s/#{space_record.identifier}/topics/#{topic_record.number}")

    # 「編集提案」タブが表示されることを確認
    expect(page).to have_link("編集提案", href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions")

    # 「編集提案」タブがアクティブになっていることを確認
    edit_suggestions_link = page.find_link("編集提案")
    expect(edit_suggestions_link[:class]).to include("border-primary")
    expect(edit_suggestions_link[:class]).to include("text-primary")

    # 「ページ」タブが非アクティブになっていることを確認
    pages_link = page.find_link("ページ")
    expect(pages_link[:class]).to include("text-muted-foreground")
  end

  it "オープン/クローズフィルターが正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # フィルタータブが表示されることを確認
    expect(page).to have_link("オープン")
    expect(page).to have_link("クローズ")

    # デフォルトでオープンタブがアクティブになっていることを確認
    open_link = page.find_link("オープン")
    expect(open_link[:class]).to include("border-primary")
    expect(open_link[:class]).to include("text-primary")
  end

  it "オープンフィルターで編集提案が正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    # オープンステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :open,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "オープンな編集提案",
      description: "これはオープンな編集提案です"
    )

    # 下書きステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :draft,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "下書きの編集提案"
    )

    # クローズステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :closed,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "クローズした編集提案"
    )

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # オープン・下書きの編集提案が表示されることを確認
    expect(page).to have_content("オープンな編集提案")
    expect(page).to have_content("下書きの編集提案")

    # クローズした編集提案が表示されないことを確認
    expect(page).not_to have_content("クローズした編集提案")

    # ステータスバッジが表示されることを確認
    expect(page).to have_content("オープン")
    expect(page).to have_content("下書き")
  end

  it "クローズフィルターで編集提案が正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    # オープンステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :open,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "オープンな編集提案"
    )

    # クローズステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :closed,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "クローズした編集提案"
    )

    # 反映済みステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :applied,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "反映済みの編集提案"
    )

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # クローズフィルターをクリック
    click_link "クローズ"

    # クローズ・反映済みの編集提案が表示されることを確認
    expect(page).to have_content("クローズした編集提案")
    expect(page).to have_content("反映済みの編集提案")

    # オープンな編集提案が表示されないことを確認
    expect(page).not_to have_content("オープンな編集提案")

    # ステータスバッジが表示されることを確認
    expect(page).to have_content("クローズ")
    expect(page).to have_content("反映済み")

    # クローズタブがアクティブになっていることを確認
    closed_link = page.find_link("クローズ")
    expect(closed_link[:class]).to include("border-primary")
    expect(closed_link[:class]).to include("text-primary")
  end

  it "編集提案がない場合に空の状態が表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # 空の状態メッセージが表示されることを確認
    expect(page).to have_content("編集提案はありません")
    expect(page).to have_content("このトピックにはまだ編集提案がありません。")
  end

  it "ゲストユーザーでも編集提案一覧が表示されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:, visibility: "public")

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # ページが正常に表示されることを確認
    expect(page).to have_content("編集提案")
    expect(page).to have_link("オープン")
    expect(page).to have_link("クローズ")
  end

  it "編集提案の詳細情報が正しく表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password, display_name: "テストユーザー")
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    # 編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :open,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "テスト編集提案",
      description: "これはテスト用の編集提案です。詳細な説明が含まれています。",
      created_at: Time.zone.parse("2024-01-15 10:30:00")
    )

    sign_in(user_record:)

    visit "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    # 編集提案の詳細情報が表示されることを確認
    expect(page).to have_content("テスト編集提案")
    expect(page).to have_content("これはテスト用の編集提案です。詳細な説明が含まれています。")
    expect(page).to have_content("テストユーザー")
    expect(page).to have_content("2024年01月15日")
    expect(page).to have_content("オープン")
  end
end
