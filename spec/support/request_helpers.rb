# typed: false
# frozen_string_literal: true

module RequestHelpers
  extend T::Sig

  def sign_in(user:, password: "passw0rd")
    space_identifier = user.space.identifier
    email = user.email
    post(
      session_list_path(skip_no_authentication: true),
      params: {session_form: {space_identifier:, email:, password:}}
    )

    expect(cookies[Session::TOKENS_COOKIE_KEY]).to be_present
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request

  config.before(:each, type: :request) do
    host! Wikino.config.host
  end
end
