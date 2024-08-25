# typed: strict
# frozen_string_literal: true

class EditNoteForm < ApplicationForm
  sig { returns(User) }
  attr_accessor :viewer

  attribute :list_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :list, presence: true
  validates :title, presence: true
  validates :body, presence: true

  sig { returns(List) }
  def list
    viewer.viewable_lists.find_by(number: list_number)
  end
end
