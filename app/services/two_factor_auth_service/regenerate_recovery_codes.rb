# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class RegenerateRecoveryCodes < ApplicationService
    sig { params(user_record: UserRecord).void }
    def call(user_record:)
      auth_record = user_record.user_two_factor_auth_record.not_nil!
      recovery_codes = UserTwoFactorAuth.generate_recovery_codes

      auth_record.update!(recovery_codes:)

      nil
    end
  end
end
