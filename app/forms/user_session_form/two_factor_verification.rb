# typed: strict
# frozen_string_literal: true

module UserSessionForm
  class TwoFactorVerification < ApplicationForm
    attribute :code, :string

    sig { params(user_record: T.nilable(UserRecord)).returns(T.nilable(UserRecord)) }
    attr_writer :user_record

    sig { returns(T.nilable(UserRecord)) }
    def user_record
      @user_record = T.let(@user_record, T.nilable(UserRecord))
    end

    validates :code, presence: true, length: {is: 6}, if: :totp_code?
    validates :code, presence: true, length: {is: 8}, if: :recovery_code?
    validate :verify_code

    sig { returns(T::Boolean) }
    def totp_code?
      code_value = code
      return false if code_value.nil?
      code_value.match?(/\A\d{6}\z/)
    end

    sig { returns(T::Boolean) }
    def recovery_code?
      code_value = code
      return false if code_value.nil?
      code_value.match?(/\A[a-z0-9]{8}\z/)
    end

    private

    sig { void }
    private def verify_code
      code_value = code
      record = user_record
      return if code_value.blank? || record.nil?

      auth_record = record.user_two_factor_auth_record
      return unless auth_record&.enabled

      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(user_two_factor_auth_record: auth_record)

      verified = if totp_code? && code_value
        # TOTPコードの検証
        two_factor_auth.verify_code(code_value)
      elsif recovery_code? && code_value
        # リカバリーコードの検証
        auth_record.recovery_code_valid?(code_value)
      else
        false
      end

      unless verified
        errors.add(:code, I18n.t("messages.two_factor_auth.invalid_code"))
      end
    end
  end
end
