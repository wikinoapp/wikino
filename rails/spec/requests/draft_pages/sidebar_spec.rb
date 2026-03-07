# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /draft_pages/sidebar", type: :request do
  it "ログインしているとき、下書きページ一覧が表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "My Draft Page")
    FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record:, space_member_record:)

    sign_in(user_record:)

    get "/draft_pages/sidebar"

    expect(response.status).to eq(200)
    expect(response.body).to include("My Draft Page")
    expect(response.body).to include("draft-pages")
  end

  it "下書きがないとき、空メッセージが表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)

    sign_in(user_record:)

    get "/draft_pages/sidebar"

    expect(response.status).to eq(200)
    expect(response.body).to include(I18n.t("messages.sidebar.draft_pages_empty"))
  end

  it "5件を超える下書きがあるとき、すべてを表示リンクが表示されること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :member, user_record:, space_record:)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    6.times do |i|
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:, title: "Draft #{i + 1}")
      FactoryBot.create(:draft_page_record, space_record:, topic_record:, page_record:, space_member_record:, modified_at: (6 - i).days.ago)
    end

    sign_in(user_record:)

    get "/draft_pages/sidebar"

    expect(response.status).to eq(200)
    expect(response.body).to include(I18n.t("messages.sidebar.draft_pages_show_all"))
  end

  it "ログインしていないとき、ログインページにリダイレクトされること" do
    get "/draft_pages/sidebar"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
