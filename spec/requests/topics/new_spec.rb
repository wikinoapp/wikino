# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "同じスペースにログインしているとき、トピック作成ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("新規トピック")
  end
end
