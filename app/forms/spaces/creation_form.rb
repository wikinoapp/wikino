# typed: strict
# frozen_string_literal: true

module Spaces
  class CreationForm < ApplicationForm
    include FormConcerns::SpaceIdentifierValidatable
    include FormConcerns::SpaceNameValidatable

    attribute :identifier, :string
    attribute :name, :string

    validate :identifier_uniqueness

    sig { void }
    private def identifier_uniqueness
      if SpaceRecord.exists?(identifier:)
        errors.add(:identifier, :uniqueness)
      end
    end
  end
end
