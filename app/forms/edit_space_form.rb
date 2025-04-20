# typed: strict
# frozen_string_literal: true

class EditSpaceForm < ApplicationForm
  include FormConcerns::SpaceIdentifierValidatable
  include FormConcerns::SpaceNameValidatable

  sig { returns(T.nilable(FormConcerns::ISpace)) }
  attr_accessor :space

  attribute :identifier, :string
  attribute :name, :string

  validates :space, presence: true
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
