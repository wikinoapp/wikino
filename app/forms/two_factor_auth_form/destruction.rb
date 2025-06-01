# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Destruction < ApplicationForm
    include ActiveModel::SecurePassword
    include FormConcerns::PasswordValidatable

    has_secure_password

    sig { params(params: T::Hash[T.untyped, T.untyped]).void }
    def initialize(params = {})
      super()
      @password = T.let(params[:password], T.nilable(String))
    end

    sig { returns(T.nilable(String)) }
    attr_accessor :password

    validates :password, presence: true
  end
end
