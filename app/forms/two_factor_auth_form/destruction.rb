# typed: strict
# frozen_string_literal: true

module TwoFactorAuthForm
  class Destruction < ApplicationForm
    include FormConcerns::PasswordValidatable

    attribute :password, :string

    validates :password, presence: true
  end
end
