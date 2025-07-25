# typed: strict
# frozen_string_literal: true

module TwoFactorAuths
  class DestructionForm < ApplicationForm
    include FormConcerns::PasswordValidatable
    include FormConcerns::PasswordAuthenticatable

    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :password, :string

    validates :user_record, presence: true
  end
end