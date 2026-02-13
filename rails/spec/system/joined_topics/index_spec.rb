# typed: false
# frozen_string_literal: true

RSpec.describe "参加中のトピック一覧", type: :system do
  it "ログインユーザーが参加中のトピック一覧を表示できること" do
    # ユーザーとスペースを作成
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record1 = FactoryBot.create(:space_record, name: "プロジェクトA")
    space_record2 = FactoryBot.create(:space_record, name: "プロジェクトB")
    space_record3 = FactoryBot.create(:space_record, name: "プロジェクトC")

    # スペースメンバーを作成
    space_member_record1 = FactoryBot.create(:space_member_record, user_record:, space_record: space_record1)
    space_member_record2 = FactoryBot.create(:space_member_record, user_record:, space_record: space_record2)

    # トピックを作成
    topic_record1 = FactoryBot.create(:topic_record, space_record: space_record1, name: "バグ修正", number: 1)
    topic_record2 = FactoryBot.create(:topic_record, space_record: space_record2, name: "新機能開発", number: 2)
    FactoryBot.create(:topic_record, space_record: space_record3, name: "リファクタリング", number: 3)

    # ユーザーをトピックに参加させる（topic_record3には参加しない）
    FactoryBot.create(:topic_member_record, space_member_record: space_member_record1, topic_record: topic_record1, space_record: space_record1)
    FactoryBot.create(:topic_member_record, space_member_record: space_member_record2, topic_record: topic_record2, space_record: space_record2)

    # ログイン
    sign_in(user_record:)

    # 参加中のトピック一覧ページにアクセス
    visit "/joined_topics"

    # 参加中のトピックが表示されることを確認
    expect(page).to have_content("プロジェクトA")
    expect(page).to have_content("バグ修正")
    expect(page).to have_content("プロジェクトB")
    expect(page).to have_content("新機能開発")

    # 参加していないトピックは表示されないことを確認
    expect(page).not_to have_content("プロジェクトC")
    expect(page).not_to have_content("リファクタリング")

    # トピックへのリンクが存在することを確認
    within("turbo-frame#joined-topics-fixed") do
      expect(page).to have_css("a.text-gray-700", count: 2)
      # 新規ページ作成へのリンクが存在することを確認
      expect(page).to have_css("a[href*='pages/new']", count: 2)
    end
  end

  it "variant=fixedパラメータでfixed版のturbo-frameが表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record, name: "テストスペース")
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:, name: "テストトピック")
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/joined_topics?variant=fixed"

    # turbo-frameのIDがfixedバリアントになっていることを確認
    expect(page).to have_css("turbo-frame#joined-topics-fixed")
  end

  it "variant=defaultパラメータでdefault版のturbo-frameが表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record, name: "テストスペース")
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:, name: "テストトピック")
    FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)

    sign_in(user_record:)

    visit "/joined_topics?variant=default"

    # turbo-frameのIDがdefaultバリアントになっていることを確認
    expect(page).to have_css("turbo-frame#joined-topics-default")
  end

  it "未ログインユーザーはアクセスできないこと" do
    visit "/joined_topics"

    # ログインページにリダイレクトされることを確認
    expect(page).to have_current_path("/sign_in")
  end

  it "参加中のトピックがない場合は空の一覧が表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)

    sign_in(user_record:)

    visit "/joined_topics"

    # turbo-frameは存在するが、トピックが表示されないことを確認
    expect(page).to have_css("turbo-frame#joined-topics-fixed")
    expect(page).to have_css(".flex.flex-col")
    expect(page).not_to have_css("a.text-gray-700[data-turbo-frame='_top']")
  end

  it "10件を超えるトピックがある場合は最新の10件のみ表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

    # 11件のトピックを作成して参加
    11.times do |i|
      topic_record = FactoryBot.create(:topic_record, space_record:, name: "トピック#{i + 1}")
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, space_record:)
    end

    sign_in(user_record:)

    visit "/joined_topics"

    # トピックの要素が10件のみ表示されることを確認
    topic_links = all("a.text-gray-700[data-turbo-frame='_top']")
    expect(topic_links.count).to eq(10)
  end
end
