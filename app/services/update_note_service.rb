# typed: strict
# frozen_string_literal: true

class UpdateNoteService < ApplicationService
  #   include NoteUpsertable
  #
  #   sig { params(form: NoteUpdatingForm).void }
  #   def initialize(form:)
  #     @form = form
  #   end
  #
  #   sig { returns(Result) }
  #   def call
  #     if form.invalid?
  #       return Result.new(note: nil, errors: errors_from_form(form))
  #     end
  #
  #     note = T.must(form.note)
  #     note.title = form.title
  #     note.modified_at = note.updated_at = Time.current
  #
  #     note_content = T.must(note.content)
  #     note_content.body = form.body
  #     note_content.body_html = form.body_html
  #
  #     note.save!
  #     note_content.save!
  #     note.link!
  #
  #     Result.new(note:, errors: [])
  #   end
  #
  #   private
  #
  #   sig { returns(NoteUpdatingForm) }
  #   attr_reader :form
end
