# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class CreationForm < ApplicationForm
    attribute :email, :string

    validates :email, email: true, presence: true
  end
end
