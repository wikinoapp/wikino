# typed: strict
# frozen_string_literal: true

require "securerandom"

module TwoFactorAuthService
  class Enable < ApplicationService
    sig { params(user: User, password: String, totp_code: String).returns(EnableResult) }
    def call(user:, password:, totp_code:)
      # Step 1: Verify the user's password
      user_record = UserRecord.find(user.database_id)
      unless user_record.user_password_record&.authenticate(password)
        return EnableResult.new(
          success: false,
          error_message: I18n.t("forms.errors.messages.incorrect"),
          recovery_codes: []
        )
      end

      # Step 2: Find the user's 2FA record
      auth_record = UserTwoFactorAuthRecord.find_by(user_id: user.database_id)
      unless auth_record
        return EnableResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.setup_required"),
          recovery_codes: []
        )
      end

      # Step 3: Verify the TOTP code
      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(user_two_factor_auth_record: auth_record)
      unless two_factor_auth.verify_code(totp_code)
        return EnableResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.invalid_code"),
          recovery_codes: []
        )
      end

      # Step 4: Generate recovery codes
      recovery_codes = generate_recovery_codes

      # Step 5: Enable 2FA and save recovery codes
      auth_record.update!(
        enabled: true,
        enabled_at: Time.current,
        recovery_codes: recovery_codes
      )

      EnableResult.new(
        success: true,
        error_message: nil,
        recovery_codes: recovery_codes
      )
    rescue => e
      Rails.logger.error("Failed to enable 2FA for user #{user.database_id}: #{e.message}")
      EnableResult.new(
        success: false,
        error_message: I18n.t("messages._common.unexpected_error"),
        recovery_codes: []
      )
    end

    class EnableResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
      const :recovery_codes, T::Array[String]
    end

    private

    sig { returns(T::Array[String]) }
    def generate_recovery_codes
      # Generate 10 recovery codes
      10.times.map do
        # Generate a random 8-character alphanumeric code
        SecureRandom.alphanumeric(8).downcase
      end
    end
  end
end
