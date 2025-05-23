# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    get "/"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていないとき、ランディングページが表示されること" do
    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("リンクで見つかるWikiアプリ")
  end
end
