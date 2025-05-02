# typed: strict
# frozen_string_literal: true

module EmailConfirmationForm
  class Creation < ApplicationForm
    attribute :email, :string

    validates :email, email: true, presence: true
  end
end
