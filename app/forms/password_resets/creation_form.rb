# typed: strict
# frozen_string_literal: true

module PasswordResets
  class CreationForm < ApplicationForm
    include FormConcerns::PasswordValidatable

    attribute :password, :string
  end
end