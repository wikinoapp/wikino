# typed: strict
# frozen_string_literal: true

class EditNoteForm < ApplicationForm
  sig { returns(User) }
  attr_accessor :viewer

  attribute :list_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :list_number, presence: true
  validates :title, presence: true
  validates :body, presence: true
end
