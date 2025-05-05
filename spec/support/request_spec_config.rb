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
