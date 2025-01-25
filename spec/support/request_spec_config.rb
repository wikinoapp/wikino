# typed: false
# frozen_string_literal: true

module RequestHelpers
  extend T::Sig

  def sign_in(user:, password: "passw0rd")
    post(
      user_session_list_path(skip_no_authentication: true),
      params: {
        user_session_form: {
          email: user.email,
          password:
        }
      }
    )

    expect(cookies[UserSession::TOKENS_COOKIE_KEY]).to be_present
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request

  config.before(:each, type: :request) do
    host! Wikino.config.host
  end
end
