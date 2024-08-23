# typed: strict
# frozen_string_literal: true

class EditNoteForm < ApplicationForm
  sig { returns(User) }
  attr_accessor :viewer

  attribute :list_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""
  attribute :draft, :boolean

  validates :list_number, presence: true

  with_options if: :published? do
    validates :title, presence: true
    validates :body, presence: true
  end

  sig { returns(T::Boolean) }
  private def published?
    !draft
  end
end
