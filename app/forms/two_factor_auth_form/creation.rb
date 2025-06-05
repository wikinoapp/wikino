# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Creation < ApplicationForm
    include FormConcerns::PasswordValidatable
    include FormConcerns::PasswordAuthenticatable

    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :password, :string
    attribute :totp_code, :string

    validates :totp_code, presence: true, length: {is: 6}, format: {with: /\A\d{6}\z/}
    validate :user_two_factor_auth_record_exists
    validate :verify_totp_code

    sig { void }
    private def user_two_factor_auth_record_exists
      record = user_record
      return if record.nil?

      if record.user_two_factor_auth_record.nil?
        errors.add(:base, :user_two_factor_auth_record_not_found)
      end
    end

    sig { void }
    private def verify_totp_code
      code = totp_code
      return if code.nil?
      
      record = user_record
      return if record.nil?
      return if record.user_two_factor_auth_record.nil?

      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(
        user_two_factor_auth_record: record.user_two_factor_auth_record.not_nil!
      )

      unless two_factor_auth.verify_code(code)
        errors.add(:base, :invalid_totp_code)
      end
    end
  end
end
