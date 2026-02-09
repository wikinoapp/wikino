# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/two_factor_auth/recovery_codes", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & 2FAが無効なとき、二要素認証の設定ページにリダイレクトされること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth")
  end

  it "ログインしている & 2FAが有効なとき、リカバリーコード画面が表示されること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(200)
    expect(response.body).to include("リカバリーコード")

    # リカバリーコードが表示されることを確認
    two_factor_auth_record.recovery_codes.each do |code|
      expect(response.body).to include(code)
    end
  end
end
