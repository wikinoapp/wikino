# typed: strict
# frozen_string_literal: true

class EditNoteForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

  attribute :notebook_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :notebook, presence: true
  validates :title, presence: true
  validates :body, presence: true

  sig { returns(T.nilable(Notebook)) }
  def notebook
    viewer&.viewable_notebooks&.find_by(number: notebook_number)
  end

  sig { returns(Notebook::PrivateRelation) }
  def viewable_notebooks
    viewer.not_nil!.viewable_notebooks
  end
end
