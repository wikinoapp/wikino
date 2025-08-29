# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/settings", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    other_space_record = create(:space_record)
    create(:space_member_record, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, space_record:, user_record:, role: SpaceMemberRole::Member.serialize)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加しているとき、トピックの設定ページが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings"

    expect(response.status).to eq(200)
    expect(response.body).to include("設定")
  end
end
