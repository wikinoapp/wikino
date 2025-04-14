# typed: strict
# frozen_string_literal: true

class UserSessionRecord < ApplicationRecord
  TOKENS_COOKIE_KEY = :user_session_tokens

  self.table_name = "user_sessions"

  has_secure_token

  belongs_to :user

  sig do
    params(
      ip_address: T.nilable(String),
      user_agent: T.nilable(String),
      signed_in_at: T.any(ActiveSupport::TimeWithZone, Time)
    ).returns(UserSession)
  end
  def self.start!(ip_address:, user_agent:, signed_in_at: Time.current)
    create!(
      ip_address: ip_address || "",
      user_agent: user_agent || "",
      signed_in_at:
    )
  end
end
