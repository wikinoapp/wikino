# typed: strict
# frozen_string_literal: true

class NewEmailConfirmationForm < ApplicationForm
  attribute :email, :string

  validates :email, email: true, presence: true
end
