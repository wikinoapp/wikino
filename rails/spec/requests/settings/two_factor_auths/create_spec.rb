# typed: false
# frozen_string_literal: true

RSpec.describe "POST /settings/two_factor_auth", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    post "/settings/two_factor_auth"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "2FAが既に有効なとき、設定画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    post "/settings/two_factor_auth", params: {
      two_factor_auths_creation_form: {
        password: "passw0rd",
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth")
  end

  it "パスワードが間違っているとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    post "/settings/two_factor_auth", params: {
      two_factor_auths_creation_form: {
        password: "wrong_password",
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードが間違っています")
  end

  it "TOTPコードが間違っているとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    post "/settings/two_factor_auth", params: {
      two_factor_auths_creation_form: {
        password: "passw0rd",
        totp_code: "000000"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("二要素認証が設定されていません")
  end

  it "正しいパスワードとTOTPコードのとき、2FAが有効化されること" do
    user_record = create(:user_record, :with_password)
    user_two_factor_auth_record = create(:user_two_factor_auth_record, {
      user_record:,
      enabled: false
    })

    sign_in(user_record:)

    # 正しいTOTPコードを生成
    totp = ROTP::TOTP.new(user_two_factor_auth_record.secret)
    correct_code = totp.now

    post "/settings/two_factor_auth", params: {
      two_factor_auths_creation_form: {
        password: "passw0rd",
        totp_code: correct_code
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth/recovery_codes")

    # 2FAが有効化されていることを確認
    user_two_factor_auth_record.reload
    expect(user_two_factor_auth_record.enabled).to be true
    expect(user_two_factor_auth_record.enabled_at).not_to be_nil
    expect(user_two_factor_auth_record.recovery_codes.size).to eq(10)
  end
end
