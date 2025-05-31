# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class Enable < ApplicationService
    sig { params(user: User, password: String, totp_code: String).returns(EnableResult) }
    def call(user:, password:, totp_code:)
      # This is a placeholder implementation
      # In a real implementation, this would:
      # 1. Verify the user's password
      # 2. Verify the TOTP code
      # 3. Generate recovery codes
      # 4. Enable 2FA for the user
      # 5. Return the result

      EnableResult.new(
        success: true,
        error_message: nil,
        recovery_codes: []
      )
    end

    class EnableResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
      const :recovery_codes, T::Array[String]
    end
  end
end

