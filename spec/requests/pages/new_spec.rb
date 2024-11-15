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

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "同じスペースにログインしているとき、ページを作成してから編集ページにリダイレクトすること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:)
    user = create(:user, :with_password, space:)

    sign_in(user:)

    expect(Page.count).to eq(0)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)

    expect(Page.count).to eq(1)
    page = topic.pages.first

    expect(response).to redirect_to("/s/#{space.identifier}/pages/#{page.number}/edit")
  end
end
