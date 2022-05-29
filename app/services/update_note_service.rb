# typed: strict
# frozen_string_literal: true

class UpdateNoteService < ApplicationService
  include NoteUpsertable

  sig { params(form: NoteUpdatingForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    if form.invalid?
      errors = form.errors.map do |error|
        if error.attribute == :title && error.type == :title_should_be_unique
          Error.new(code: "DUPLICATED_NOTE_ERROR", message: error.full_message, original_note: form.original_note)
        else
          Error.new(code: "INVALID_ERROR", message: error.full_message)
        end
      end

      return Result.new(note: nil, errors:)
    end

    note = T.must(form.note)
    note.title = form.title
    note.modified_at = note.updated_at = Time.current

    note_content = T.must(note.content)
    note_content.body = form.body
    note_content.body_html = form.body_html

    note.save!
    note_content.save!
    note.link!

    Result.new(note:, errors: [])
  end

  private

  sig { returns(NoteUpdatingForm) }
  attr_reader :form
end
