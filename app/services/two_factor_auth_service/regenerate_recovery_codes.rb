# typed: strict
# frozen_string_literal: true

require "securerandom"

module TwoFactorAuthService
  class RegenerateRecoveryCodes < ApplicationService
    sig { params(user: User, password: String).returns(RegenerateResult) }
    def call(user:, password:)
      # パスワードを検証
      user_record = UserRecord.find(user.database_id)
      unless user_record.user_password_record&.authenticate(password)
        return RegenerateResult.new(
          success: false,
          error_message: I18n.t("forms.errors.messages.incorrect"),
          recovery_codes: []
        )
      end

      # 2FAレコードを検索
      auth_record = UserTwoFactorAuthRecord.find_by(user_id: user.database_id)
      unless auth_record&.enabled
        return RegenerateResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.not_enabled"),
          recovery_codes: []
        )
      end

      # 新しいリカバリーコードを生成
      recovery_codes = generate_recovery_codes

      # リカバリーコードを更新
      auth_record.update!(recovery_codes: recovery_codes)

      RegenerateResult.new(
        success: true,
        error_message: nil,
        recovery_codes: recovery_codes
      )
    rescue => e
      Rails.logger.error("Failed to regenerate recovery codes for user #{user.database_id}: #{e.message}")
      RegenerateResult.new(
        success: false,
        error_message: I18n.t("messages._common.unexpected_error"),
        recovery_codes: []
      )
    end

    class RegenerateResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
      const :recovery_codes, T::Array[String]
    end

    private

    sig { returns(T::Array[String]) }
    def generate_recovery_codes
      # 10個のリカバリーコードを生成
      10.times.map do
        # 8文字の英数字小文字でランダム生成
        SecureRandom.alphanumeric(8).downcase
      end
    end
  end
end
