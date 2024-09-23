# typed: strict
# frozen_string_literal: true

class EditNoteForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

  attribute :list_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :list, presence: true
  validates :title, presence: true
  validates :body, presence: true

  sig { returns(T.nilable(List)) }
  def list
    viewer&.viewable_lists&.find_by(number: list_number)
  end

  sig { returns(List::PrivateRelation) }
  def viewable_lists
    viewer.not_nil!.viewable_lists
  end
end
