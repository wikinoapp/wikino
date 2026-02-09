# typed: false
# frozen_string_literal: true

module RequestHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    user_session_record = user_record.user_session_records.start!(
      ip_address: "127.0.0.1",
      user_agent: "RSpec"
    )
    cookies[UserSession::TOKENS_COOKIE_KEY] = user_session_record.token
  end

  def sign_in_with_2fa(user_record:, password: "passw0rd")
    sign_in(user_record:, password:)
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
