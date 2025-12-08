# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    get "/sign_in"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/sign_in"

    expect(response.status).to eq(200)
    expect(response.body).to include("ログイン")
  end
end
