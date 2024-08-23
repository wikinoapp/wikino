# typed: strict
# frozen_string_literal: true

class CreateNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :note, Note
  end

  sig { params(list: List, viewer: User).returns(Result) }
  def call(list:, viewer:)
    note = ActiveRecord::Base.transaction do
      note = viewer.create_note!(list:)
      note.add_editor(user: viewer)
      note
    end

    Result.new(note:)
  end
end
