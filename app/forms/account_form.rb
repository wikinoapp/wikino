# typed: strict
# frozen_string_literal: true

class AccountForm < ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  # validates :space_identifier,
  #   format: {with: Space::IDENTIFIER_FORMAT},
  #     length: {in: Space::IDENTIFIER_MIN_LENGTH..Space::IDENTIFIER_MAX_LENGTH},
  #     presence: true,
  #     unreserved_space_identifier: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
  validates :time_zone, presence: true
  # validate :space_identifier_uniqueness

  # sig { void }
  # private def space_identifier_uniqueness
  #   if Space.find_by(identifier:)
  #     errors.add(:space_identifier, :uniqueness)
  #   end
  # end
end
