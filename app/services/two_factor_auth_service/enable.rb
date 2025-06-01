# typed: strict
# frozen_string_literal: true

require "securerandom"

module TwoFactorAuthService
  class Enable < ApplicationService
    sig { params(user: User, password: String, totp_code: String).returns(EnableResult) }
    def call(user:, password:, totp_code:)
      # ステップ1: ユーザーのパスワードを検証する
      user_record = UserRecord.find(user.database_id)
      unless user_record.user_password_record&.authenticate(password)
        return EnableResult.new(
          success: false,
          error_message: I18n.t("forms.errors.messages.incorrect"),
          recovery_codes: []
        )
      end

      # ステップ2: ユーザーの2FAレコードを検索する
      auth_record = UserTwoFactorAuthRecord.find_by(user_id: user.database_id)
      unless auth_record
        return EnableResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.setup_required"),
          recovery_codes: []
        )
      end

      # ステップ3: TOTPコードを検証する
      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(user_two_factor_auth_record: auth_record)
      unless two_factor_auth.verify_code(totp_code)
        return EnableResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.invalid_code"),
          recovery_codes: []
        )
      end

      # ステップ4: リカバリーコードを生成する
      recovery_codes = generate_recovery_codes

      # ステップ5: 2FAを有効化し、リカバリーコードを保存する
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
      # 10個のリカバリーコードを生成する
      10.times.map do
        # ランダムな8文字の英数字コードを生成する
        SecureRandom.alphanumeric(8).downcase
      end
    end
  end
end
