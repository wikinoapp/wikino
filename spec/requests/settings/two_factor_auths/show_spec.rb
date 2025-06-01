# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/two_factor_auth", type: :request do
  it "ログインしているとき、2FA設定画面が表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/two_factor_auth"

    expect(response.status).to eq(200)
    expect(response.body).to include("二要素認証")
  end

  it "ログインしていないとき、ログインページにリダイレクトすること" do
    get "/settings/two_factor_auth"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "2FAが有効なとき、有効化状態が表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    get "/settings/two_factor_auth"

    expect(response.status).to eq(200)
    expect(response.body).to include("有効")
  end

  it "2FAが無効なとき、無効化状態が表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/two_factor_auth"

    expect(response.status).to eq(200)
    expect(response.body).to include("無効")
  end
end
