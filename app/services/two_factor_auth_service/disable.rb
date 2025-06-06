# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class Disable < ApplicationService
    sig { params(user_record: UserRecord).void }
    def call(user_record:)
      auth_record = user_record.user_two_factor_auth_record.not_nil!

      # 2FAを無効化し、リカバリーコードをクリア
      auth_record.update!(
        enabled: false,
        enabled_at: nil,
        recovery_codes: []
      )

      nil
    end
  end
end
