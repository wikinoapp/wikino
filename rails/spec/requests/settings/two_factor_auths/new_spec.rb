# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/two_factor_auth/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    get "/settings/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & 2FAが無効なとき、セットアップ画面が表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/two_factor_auth/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("二要素認証の設定")
    # QRコードが表示されているはず
    expect(response.body).to include("<svg")
  end

  it "ログインしている & 2FAが既に有効なとき、設定画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    get "/settings/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth")
  end
end
