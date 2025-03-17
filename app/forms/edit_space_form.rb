# typed: strict
# frozen_string_literal: true

class EditSpaceForm < ApplicationForm
  sig { returns(T.nilable(FormConcerns::ISpace)) }
  attr_accessor :space

  attribute :identifier, :string
  attribute :name, :string

  validates :space, presence: true
  validates :identifier,
    exclusion: {in: Space::IDENTIFIER_RESERVED_WORDS},
    format: {with: Space::IDENTIFIER_FORMAT},
    length: {minimum: Space::IDENTIFIER_MIN_LENGTH, maximum: Space::IDENTIFIER_MAX_LENGTH},
    presence: true
  validates :name, presence: true
  validate :identifier_uniqueness

  sig { void }
  private def identifier_uniqueness
    return if space.nil?
    return if identifier.nil?

    if space.not_nil!.identifier_uniqueness?(identifier.not_nil!)
      errors.add(:identifier, :uniqueness)
    end
  end
end
