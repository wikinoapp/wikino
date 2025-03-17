# typed: strict
# frozen_string_literal: true

class NewSpaceForm < ApplicationForm
  attribute :identifier, :string
  attribute :name, :string

  validates :identifier,
    exclusion: {in: Space::IDENTIFIER_RESERVED_WORDS, message: :reserved},
    format: {with: Space::IDENTIFIER_FORMAT},
    length: {maximum: Space::IDENTIFIER_MAX_LENGTH},
    presence: true
  validates :name,
    length: {maximum: Space::NAME_MAX_LENGTH},
    presence: true
  validate :identifier_uniqueness

  sig { void }
  private def identifier_uniqueness
    if Space.exists?(identifier:)
      errors.add(:identifier, :uniqueness)
    end
  end
end
