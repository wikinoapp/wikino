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

    sig { void }
    private def authentication
      unless user_record&.user_password_record&.authenticate(password)
        errors.add(:base, :unauthenticated)
      end
    end
  end
end
