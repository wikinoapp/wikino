# typed: strict
# frozen_string_literal: true

module UserSessionForm
  class TwoFactorRecovery < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :code, :string

    validates :code, presence: true, length: {is: 6}, if: :totp_code?
    validates :code, presence: true, length: {is: 8}, if: :recovery_code?
    validate :verify_code

    sig { returns(T::Boolean) }
    private def totp_code?
      return false if code.nil?

      code.match?(/\A\d{6}\z/)
    end

    sig { returns(T::Boolean) }
    private def recovery_code?
      return false if code.nil?

      code.match?(/\A[a-z0-9]{8}\z/)
    end

    sig { void }
    private def verify_code
      return if code.blank? || user_record.nil?
      return unless user_record.two_factor_enabled?

      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(
        user_two_factor_auth_record: user_record.user_two_factor_auth_record.not_nil!
      )

      verified = if totp_code?
        # TOTPコードの検証
        two_factor_auth.verify_code(code)
      elsif recovery_code? && code
        # リカバリーコードの検証
        auth_record.recovery_code_valid?(code)
      else
        false
      end

      unless verified
        errors.add(:code, I18n.t("messages.two_factor_auth.invalid_code"))
      end
    end
  end
end
