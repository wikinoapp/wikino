# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Creation < ApplicationForm
    include ActiveModel::SecurePassword
    include FormConcerns::PasswordValidatable

    has_secure_password

    sig { params(params: T::Hash[T.untyped, T.untyped]).void }
    def initialize(params = {})
      super()
      @password = T.let(params[:password], T.nilable(String))
      @totp_code = T.let(params[:totp_code], T.nilable(String))
    end

    sig { returns(T.nilable(String)) }
    attr_accessor :password

    sig { returns(T.nilable(String)) }
    attr_accessor :totp_code

    validates :password, presence: true
    validates :totp_code, presence: true, length: {is: 6}, format: {with: /\A\d{6}\z/}
  end
end

