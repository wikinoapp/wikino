# typed: false
# frozen_string_literal: true

module RequestHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    post(
      user_session_path,
      params: {
        user_session_form_creation: {
          email: user_record.email,
          password:
        }
      }
    )

    expect(response.status).to eq(302)

    # 2FAが有効な場合は2FA検証画面にリダイレクトされるはず
    if user_record.two_factor_enabled?
      expect(response).to redirect_to("/sign_in/two_factor/new")
      expect(session[:pending_user_id]).to eq(user_record.id)
    else
      expect(cookies[UserSession::TOKENS_COOKIE_KEY]).to be_present
    end
  end

  def sign_in_with_2fa(user_record:, password: "passw0rd")
    sign_in(user_record:, password:)

    # 2FAが有効な場合は2FA検証画面にリダイレクトされるはず
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in/two_factor/new")
    expect(session[:pending_user_id]).to eq(user_record.id)

    two_factor_auth_record = UserTwoFactorAuthRecord.find_by!(user_record:)
    totp = ROTP::TOTP.new(two_factor_auth_record.secret)
    correct_code = totp.now

    post(
      sign_in_two_factor_path,
      params: {
        user_session_form_two_factor_verification: {
          totp_code: correct_code
        }
      }
    )

    expect(response.status).to eq(302)
    expect(cookies[UserSession::TOKENS_COOKIE_KEY]).to be_present
  end

  def set_session(session_attrs = {})
    post(
      test_session_path,
      params: {session_attrs:}
    )
    expect(response).to have_http_status(:created)

    session_attrs.each_key do |key|
      expect(session[key]).to be_present
    end
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request

  config.before(:each, type: :request) do
    host! Wikino.config.host
  end
end
