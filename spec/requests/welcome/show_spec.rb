# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/"

    expect(response).to have_http_status(:found)
    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))
  end

  it "ログインしていないとき、ランディングページが表示されること" do
    get "/"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("Wikinoにようこそ！")
  end
end
