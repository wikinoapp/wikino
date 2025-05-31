# typed: strict
# frozen_string_literal: true

class UserTwoFactorAuthRepository < ApplicationRepository
  sig { params(user_two_factor_auth_record: UserTwoFactorAuthRecord).returns(UserTwoFactorAuth) }
  def to_model(user_two_factor_auth_record:)
    UserTwoFactorAuth.new(
      database_id: user_two_factor_auth_record.id,
      user_id: user_two_factor_auth_record.user_id,
      secret: user_two_factor_auth_record.secret,
      enabled: user_two_factor_auth_record.enabled,
      enabled_at: user_two_factor_auth_record.enabled_at,
      recovery_codes: user_two_factor_auth_record.recovery_codes
    )
  end

  sig { params(user_record: UserRecord).returns(T.nilable(UserTwoFactorAuth)) }
  def find_by_user(user_record:)
    record = user_record.user_two_factor_auth_record
    return nil unless record

    to_model(user_two_factor_auth_record: record)
  end
end