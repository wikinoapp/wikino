# typed: strict
# frozen_string_literal: true

require "securerandom"

module TwoFactorAuths
  class EnableService < ApplicationService
    sig { params(user_record: UserRecord, password: String, totp_code: String).void }
    def call(user_record:, password:, totp_code:)
      auth_record = user_record.user_two_factor_auth_record.not_nil!
      recovery_codes = UserTwoFactorAuth.generate_recovery_codes

      # 2FAを有効化し、リカバリーコードを保存する
      auth_record.update!(
        enabled: true,
        enabled_at: Time.current,
        recovery_codes:
      )

      nil
    end
  end
end
