# typed: strict
# frozen_string_literal: true

class CreateInitialNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :note, Note
  end

  sig { params(topic: Topic, viewer: User).returns(Result) }
  def call(topic:, viewer:)
    note = ActiveRecord::Base.transaction do
      new_note = Note.create_as_initial!(topic:)
      new_note.add_editor!(editor: viewer)
      new_note
    end

    Result.new(note:)
  end
end
