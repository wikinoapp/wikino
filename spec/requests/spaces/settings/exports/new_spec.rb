# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/settings/exports/new", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space = create(:space)

    get "/s/#{space.identifier}/settings/exports/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/new"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    other_space = create(:space)
    create(:space_member, user:, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/new"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加しているとき、エクスポート画面が表示されること" do
    user = create(:user, :with_password)
    space = create(:space)
    create(:space_member, user:, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("エクスポート")
  end
end
