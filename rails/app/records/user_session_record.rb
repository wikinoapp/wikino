# typed: strict
# frozen_string_literal: true

class UserSessionRecord < ApplicationRecord
  self.table_name = "user_sessions"

  has_secure_token

  belongs_to :user_record, foreign_key: :user_id

  sig do
    params(
      ip_address: T.nilable(String),
      user_agent: T.nilable(String),
      signed_in_at: T.any(ActiveSupport::TimeWithZone, Time)
    ).returns(UserSessionRecord)
  end
  def self.start!(ip_address:, user_agent:, signed_in_at: Time.current)
    create!(
      ip_address: ip_address || "",
      user_agent: user_agent || "",
      signed_in_at:
    )
  end

  # sig { void }
  # private def authentication
  #   unless user&.user_password_record&.authenticate(password)
  #     errors.add(:base, :unauthenticated)
  #   end
  # end
end
