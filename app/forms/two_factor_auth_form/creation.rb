# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Creation < ApplicationForm
    include FormConcerns::PasswordValidatable

    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :password, :string
    attribute :totp_code, :string

    validates :totp_code, presence: true, length: {is: 6}, format: {with: /\A\d{6}\z/}
    validate :authentication
    validate :user_two_factor_auth_record_exists
    validate :verify_totp_code

    sig { void }
    private def authentication
      return if user_record.nil?

      unless user_record.user_password_record&.authenticate(password)
        errors.add(:base, :unauthenticated)
      end
    end

    sig { void }
    private def user_two_factor_auth_record_exists
      return if user_record.nil?

      if user_record.user_two_factor_auth_record.nil?
        errors.add(:base, :user_two_factor_auth_record_not_found)
      end
    end

    sig { void }
    private def verify_totp_code
      return if user_record.nil?
      return if user_record.user_two_factor_auth_record.nil?

      two_factor_auth = UserTwoFactorAuthRepository.new.to_model(
        user_two_factor_auth_record: user_record.user_two_factor_auth_record.not_nil!
      )

      unless two_factor_auth.verify_code(totp_code)
        errors.add(:base, :invalid_totp_code)
      end
    end
  end
end
