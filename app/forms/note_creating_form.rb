# typed: strict
# frozen_string_literal: true

class NoteCreatingForm < ApplicationForm
  include NoteInputtable

  # @overload
  sig { returns(T::Boolean) }
  def persisted?
    false
  end
end
