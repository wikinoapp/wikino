# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/settings/general", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space = create(:space)
    topic = create(:topic, space:)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:)
    other_space = create(:space)
    create(:space_member, user:, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加していないとき、トピックの設定ページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:)
    create(:space_member, space:, user:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(200)
    expect(response.body).to include("基本情報")
  end

  it "ログインしている & スペースに参加している & トピックに参加しているとき、トピックの設定ページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:)
    space_member = create(:space_member, space:, user:)
    create(:topic_member, space:, topic:, space_member:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(200)
    expect(response.body).to include("基本情報")
  end
end
