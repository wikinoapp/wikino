# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/pages/:page_number/edit", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    page = create(:page, space:)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    page = create(:page, space:)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "オーナーとしてログインしている & ページのトピックに参加していないとき、編集ページが表示されること" do
    space = create(:space, :small)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:, title: "ページタイトル")
    user = create(:user, :owner, :with_password, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(200)
    expect(response.body).to include("ページタイトル")
  end

  it "オーナーとしてログインしている & ページのトピックに参加しているとき、編集ページが表示されること" do
    space = create(:space, :small)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:, title: "ページタイトル")
    user = create(:user, :owner, :with_password, space:)
    create(:topic_membership, space:, topic:, member: user)

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(200)
    expect(response.body).to include("ページタイトル")
  end
end
