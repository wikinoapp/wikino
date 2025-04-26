# typed: strict
# frozen_string_literal: true

class UserSession < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  TOKENS_COOKIE_KEY = :user_session_tokens
  DEFAULT_IP_ADDRESS = "unknown"
  DEFAULT_USER_AGENT = "unknown"

  const :user, User
  const :token, String
  const :ip_address, String
  const :user_agent, String
  const :signed_in_at, ActiveSupport::TimeWithZone
end
