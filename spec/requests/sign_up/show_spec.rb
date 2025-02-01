# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_up", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_up"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていないとき、アカウント作成ページが表示されること" do
    get "/sign_up"

    expect(response.status).to eq(200)
    expect(response.body).to include("アカウント作成")
  end
end
