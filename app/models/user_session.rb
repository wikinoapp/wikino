# typed: strict
# frozen_string_literal: true

class UserSession < ApplicationModel
  TOKENS_COOKIE_KEY = :user_session_tokens

  sig { returns(User) }
  attr_accessor :user

  sig { returns(String) }
  attr_accessor :token
end
