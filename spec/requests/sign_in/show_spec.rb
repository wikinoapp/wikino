# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_in"

    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to("/s/#{space.identifier}")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/sign_in"

    expect(response.status).to eq(200)
    expect(response.body).to include("ログイン")
  end
end
