# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/settings/general", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)
    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加していないとき、トピックの設定ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(200)
    expect(response.body).to include("基本情報")
  end

  it "ログインしている & スペースに参加している & トピックに参加しているとき、トピックの設定ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)
    space_member = create(:space_member_record, space_record: space, user_record: user)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(200)
    expect(response.body).to include("基本情報")
  end
end
