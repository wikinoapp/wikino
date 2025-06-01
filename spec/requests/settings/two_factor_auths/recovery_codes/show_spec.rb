# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/two_factor_auth/recovery_codes", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "2FAが無効なとき、404エラーになること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(404)
  end

  it "2FAが有効なとき、リカバリーコード画面が表示されること" do
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

  it "セッションにリカバリーコードがあるとき、それらが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    # セッションにリカバリーコードを設定
    session_recovery_codes = ["code1234", "code5678"]
    set_session(recovery_codes: session_recovery_codes)

    get "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(200)
    session_recovery_codes.each do |code|
      expect(response.body).to include(code)
    end
  end
end
