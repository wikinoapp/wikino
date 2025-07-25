# typed: strict
# frozen_string_literal: true

module Spaces
  class EditForm < ApplicationForm
    include FormConcerns::SpaceIdentifierValidatable
    include FormConcerns::SpaceNameValidatable

    sig { returns(T.nilable(SpaceRecord)) }
    attr_accessor :space_record

    attribute :identifier, :string
    attribute :name, :string

    validates :space_record, presence: true
    validate :identifier_uniqueness

    sig { void }
    private def identifier_uniqueness
      return if space_record.nil?
      return if identifier.nil?

      if space_record.not_nil!.identifier_uniqueness?(identifier.not_nil!)
        errors.add(:identifier, :uniqueness)
      end
    end
  end
end