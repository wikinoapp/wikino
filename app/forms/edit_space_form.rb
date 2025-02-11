# typed: strict
# frozen_string_literal: true

class EditSpaceForm < ApplicationForm
  attribute :identifier, :string
  attribute :name, :string

  validates :identifier,
    exclusion: {in: Space::RESERVED_IDENTIFIERS},
    format: {with: Space::IDENTIFIER_FORMAT},
    length: {minimum: Space::IDENTIFIER_MIN_LENGTH, maximum: Space::IDENTIFIER_MAX_LENGTH},
    presence: true
  validates :name, presence: true
  validate :identifier_uniqueness

  sig { void }
  private def identifier_uniqueness
    if Space.exists?(identifier:)
      errors.add(:identifier, :uniqueness)
    end
  end
end
