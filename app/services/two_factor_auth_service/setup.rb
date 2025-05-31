# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class Setup < ApplicationService
    sig { params(user: User).returns(SetupResult) }
    def call(user:)
      # This is a placeholder implementation
      # In a real implementation, this would:
      # 1. Generate a new TOTP secret
      # 2. Create or update the user_two_factor_auth record
      # 3. Generate a QR code
      # 4. Return the result with provisioning URI and QR code

      SetupResult.new(
        success: true,
        secret: "PLACEHOLDER_SECRET",
        provisioning_uri: "otpauth://totp/Wikino:#{user.email}?secret=PLACEHOLDER_SECRET&issuer=Wikino",
        qr_code: nil
      )
    end

    class SetupResult < T::Struct
      const :success, T::Boolean
      const :secret, String
      const :provisioning_uri, String
      const :qr_code, T.nilable(String)
    end
  end
end

