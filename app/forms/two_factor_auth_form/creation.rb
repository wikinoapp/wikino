# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Creation < ApplicationForm
    include FormConcerns::PasswordValidatable

    attribute :password, :string
    attribute :totp_code, :string

    validates :password, presence: true
    validates :totp_code, presence: true, length: {is: 6}, format: {with: /\A\d{6}\z/}
  end
end
