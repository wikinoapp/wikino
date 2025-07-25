# typed: strict
# frozen_string_literal: true

module UserSessions
  class TwoFactorVerificationForm < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :totp_code, :string

    validates :totp_code, presence: true, length: {is: 6}, format: {with: /\A\d{6}\z/}
    validate :verify_totp_code

    sig { void }
    private def verify_totp_code
      return if totp_code.blank?

      record = user_record
      return if record.nil?
      return unless record.two_factor_enabled?

      code = totp_code
      return if code.nil?

      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(
        user_two_factor_auth_record: record.user_two_factor_auth_record.not_nil!
      )

      unless two_factor_auth.verify_code(code)
        errors.add(:totp_code, :invalid_code)
      end
    end
  end
end