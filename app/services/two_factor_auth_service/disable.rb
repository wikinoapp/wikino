# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class Disable < ApplicationService
    sig { params(user: User, password: String).returns(DisableResult) }
    def call(user:, password:)
      # This is a placeholder implementation
      # In a real implementation, this would:
      # 1. Verify the user's password
      # 2. Disable 2FA for the user
      # 3. Clear recovery codes
      # 4. Return the result

      DisableResult.new(
        success: true,
        error_message: nil
      )
    end

    class DisableResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
    end
  end
end

