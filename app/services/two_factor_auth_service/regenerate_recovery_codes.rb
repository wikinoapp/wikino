# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class RegenerateRecoveryCodes < ApplicationService
    sig { params(user: User, password: String).returns(RegenerateResult) }
    def call(user:, password:)
      # This is a placeholder implementation
      # In a real implementation, this would:
      # 1. Verify the user's password
      # 2. Generate new recovery codes
      # 3. Update the user_two_factor_auth record
      # 4. Return the result

      RegenerateResult.new(
        success: true,
        error_message: nil,
        recovery_codes: []
      )
    end

    class RegenerateResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
      const :recovery_codes, T::Array[String]
    end
  end
end

