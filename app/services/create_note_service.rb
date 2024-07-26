# typed: strict
# frozen_string_literal: true

class CreateNoteService < ApplicationService
  #   include NoteUpsertable
  #
  #   sig { params(form: NoteCreatingForm).void }
  #   def initialize(form:)
  #     @form = form
  #     @user = T.let(form.user, T.nilable(User))
  #   end
  #
  #   sig { returns(Result) }
  #   def call
  #     if form.invalid?
  #       return Result.new(note: nil, errors: errors_from_form(form))
  #     end
  #
  #     modified_at = created_at = updated_at = Time.current
  #     note = T.must(user).notes.new(title: form.title, modified_at:, created_at:, updated_at:)
  #     note.build_content(user:, body: form.body)
  #     note.save!
  #     note.link!
  #
  #     Result.new(note:, errors: [])
  #   end
  #
  #   private
  #
  #   sig { returns(T.nilable(User)) }
  #   attr_reader :user
  #
  #   sig { returns(NoteCreatingForm) }
  #   attr_reader :form
end
