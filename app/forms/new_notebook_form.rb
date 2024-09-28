# typed: strict
# frozen_string_literal: true

class NewNotebookForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :name, presence: true
  validates :visibility, presence: true
end
