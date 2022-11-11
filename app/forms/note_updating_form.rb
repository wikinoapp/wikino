# typed: strict
# frozen_string_literal: true

class NoteUpdatingForm < ApplicationForm
  include NoteInputtable

  sig { returns(T.nilable(::Note)) }
  attr_accessor :note

  validates :note, presence: true

  # @overload
  sig { returns(T::Boolean) }
  def persisted?
    true
  end

  private

  sig { returns(ActiveRecord::Relation) }
  def user_notes
    T.must(user).notes.where.not(id: T.must(note).id)
  end
end
