# typed: strict
# frozen_string_literal: true

module PasswordResetForm
  class Creation < ApplicationForm
    include FormConcerns::PasswordValidatable

    attribute :password, :string
  end
end
