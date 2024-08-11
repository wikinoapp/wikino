# typed: strict
# frozen_string_literal: true

class Session < ApplicationRecord
  COOKIE_KEY = :nonoto_session_token

  has_secure_token

  belongs_to :space
  belongs_to :user

  sig do
    params(
      space: Space,
      ip_address: T.nilable(String),
      user_agent: T.nilable(String),
      signed_in_at: ActiveSupport::TimeWithZone
    ).returns(Session)
  end
  def self.start!(space:, ip_address:, user_agent:, signed_in_at: Time.current)
    create!(
      space:,
      ip_address: ip_address || "",
      user_agent: user_agent || "",
      signed_in_at:
    )
  end
end
