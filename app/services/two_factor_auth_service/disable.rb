# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class Disable < ApplicationService
    sig { params(user: User, password: String).returns(DisableResult) }
    def call(user:, password:)
      # パスワードを検証
      user_record = UserRecord.find(user.database_id)
      unless user_record.user_password_record&.authenticate(password)
        return DisableResult.new(
          success: false,
          error_message: I18n.t("forms.errors.messages.incorrect")
        )
      end

      # 2FAレコードを検索
      auth_record = UserTwoFactorAuthRecord.find_by(user_id: user.database_id)
      unless auth_record&.enabled
        return DisableResult.new(
          success: false,
          error_message: I18n.t("messages.two_factor_auth.not_enabled")
        )
      end

      # 2FAを無効化し、リカバリーコードをクリア
      auth_record.update!(
        enabled: false,
        enabled_at: nil,
        recovery_codes: []
      )

      DisableResult.new(
        success: true,
        error_message: nil
      )
    rescue => e
      Rails.logger.error("Failed to disable 2FA for user #{user.database_id}: #{e.message}")
      DisableResult.new(
        success: false,
        error_message: I18n.t("messages._common.unexpected_error")
      )
    end

    class DisableResult < T::Struct
      const :success, T::Boolean
      const :error_message, T.nilable(String)
    end
  end
end
