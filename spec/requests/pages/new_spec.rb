# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/pages/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:)

    other_space = create(:space)
    user = create(:user, :with_password)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    create(:space_member, space:, user:)
    topic = create(:topic, :public, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加しているとき、ページを作成してから編集ページにリダイレクトすること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    space_member = create(:space_member, space:, user:)
    topic = create(:topic, :public, space:)
    create(:topic_member, space:, topic:, space_member:)

    sign_in(user:)

    expect(PageRecord.count).to eq(0)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)

    expect(PageRecord.count).to eq(1)
    page = topic.pages.first

    expect(response).to redirect_to("/s/#{space.identifier}/pages/#{page.number}/edit")
  end
end
