# typed: strict
# frozen_string_literal: true

class NewNotebookForm < ApplicationForm
  sig { returns(User) }
  attr_accessor :viewer

  attribute :identifier, :string
  attribute :visibility, :string
  attribute :name, :string, default: ""
  attribute :description, :string, default: ""

  validates :identifier, presence: true
  validates :visibility, presence: true
  validate :identifier_uniqueness

  sig { void }
  private def identifier_uniqueness
    if viewer && viewer.space.notebooks.find_by(identifier:)
      errors.add(:identifier, :uniqueness)
    end
  end
end
