# typed: strict
# frozen_string_literal: true

class CreateInitialNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :note, Note
  end

  sig { params(list: List, viewer: User).returns(Result) }
  def call(list:, viewer:)
    note = ActiveRecord::Base.transaction do
      new_note = viewer.create_initial_note!(list:)
      new_note.add_editor!(editor: viewer)
      new_note
    end

    Result.new(note:)
  end
end
