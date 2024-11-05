# typed: strict
# frozen_string_literal: true

module RequestHelpers
  extend T::Sig

  sig { params(user: User, password: String).void }
  def sign_in(user:, password: "passw0rd")
    expect(cookies[Session::TOKEN_COOKIE_KEY]).to be_nil
    expect(cookies[Session::USER_IDS_COOKIE_KEY]).to be_nil

    space_identifier = user.space.identifier
    email = user.email
    post(session_list_path, params: {session_form: {space_identifier:, email:, password:}})

    expect(cookies[Session::TOKEN_COOKIE_KEY]).to be_present
    expect(cookies[Session::USER_IDS_COOKIE_KEY]).to be_present
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request

  config.before(:each, type: :request) do
    host! Wikino.config.host
  end
end
